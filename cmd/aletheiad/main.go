package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	aletheiahttp "github.com/dukerupert/aletheia/http"
	"github.com/dukerupert/aletheia/internal/migrations"
	"github.com/dukerupert/aletheia/internal/templates"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Stderr, os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run is the main entry point for the application, designed for testability.
// It accepts all external dependencies (IO, args, env) as parameters.
func run(
	ctx context.Context,
	stdout, stderr io.Writer,
	args []string,
	getenv func(string) string,
) error {
	// Load configuration
	cfg, err := LoadConfig(getenv)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Configure logger
	logger := newLogger(stderr, cfg)
	slog.SetDefault(logger)
	logger.Debug("logger initialized", slog.String("level", cfg.LogLevel))
	logger.Debug("application configuration",
		slog.String("environment", cfg.Environment),
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port))

	// Create database connection pool
	pool, err := newDatabasePool(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("creating database pool: %w", err)
	}
	defer pool.Close()

	// Run migrations
	if err := runMigrations(pool, logger); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Initialize template renderer
	renderer, err := templates.NewTemplateRenderer("web/templates")
	if err != nil {
		return fmt.Errorf("initializing templates: %w", err)
	}
	logger.Info("template renderer initialized")

	// Initialize services
	services, err := initServices(ctx, pool, cfg, logger)
	if err != nil {
		return fmt.Errorf("initializing services: %w", err)
	}

	// Create HTTP server configuration
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	serverCfg := aletheiahttp.Config{
		Addr:                addr,
		Logger:              logger,
		Renderer:            renderer,
		SessionDuration:     cfg.SessionDuration,
		SessionSecure:       cfg.SessionSecure,
		UserService:         services.UserService,
		SessionService:      services.SessionService,
		OrganizationService: services.OrganizationService,
		ProjectService:      services.ProjectService,
		InspectionService:   services.InspectionService,
		PhotoService:        services.PhotoService,
		ViolationService:    services.ViolationService,
		SafetyCodeService:   services.SafetyCodeService,
		FileStorage:         services.FileStorage,
		EmailService:        services.EmailService,
		AIService:           services.AIService,
		Queue:               services.Queue,
	}

	// Create HTTP server
	server := aletheiahttp.NewServer(serverCfg)

	// Create channel for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting server", slog.String("addr", addr))
		if err := server.Open(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	// Graceful shutdown
	logger.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Close(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		return fmt.Errorf("server shutdown: %w", err)
	}

	logger.Info("server exited gracefully")
	return nil
}

// newLogger creates a configured slog.Logger based on environment.
func newLogger(w io.Writer, cfg *Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	if cfg.Environment == "prod" || cfg.Environment == "production" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String("time", a.Value.Time().Format(time.RFC3339Nano))
				}
				return a
			},
		})
	} else {
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
}

// newDatabasePool creates a configured pgxpool connection pool.
func newDatabasePool(ctx context.Context, cfg *Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	connString := cfg.DatabaseURL()
	logger.Debug("connecting to database")

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	logger.Info("database connection pool established")
	return pool, nil
}

// runMigrations runs database migrations using goose.
func runMigrations(pool *pgxpool.Pool, logger *slog.Logger) error {
	logger.Info("running database migrations...")

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("database migrations completed")
	return nil
}
