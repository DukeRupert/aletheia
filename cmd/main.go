package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/handlers"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Configure logger
	logger := slog.New(cfg.GetLogger())
	slog.SetDefault(logger)
	logger.Debug("logger initialized", slog.String("level", cfg.Logger.Level.String()))

	// Create pool configuration
	connString := cfg.GetConnectionString()
	logger.Debug("connecting to database", slog.String("url", connString))

	// Parse the connection string and create pool config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatal(err)
	}

	// Configure pool settings (optional but recommended)
	poolConfig.MaxConns = 25                      // Maximum connections in pool
	poolConfig.MinConns = 5                       // Minimum idle connections
	poolConfig.MaxConnLifetime = time.Hour        // Max lifetime of a connection
	poolConfig.MaxConnIdleTime = time.Minute * 30 // Max idle time before closing
	poolConfig.HealthCheckPeriod = time.Minute    // How often to check connection health

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}

	logger.Info("database connection pool established")

	// Initialize storage (local for now)
	fileStorage, err := storage.NewLocalStorage("./uploads", "http://localhost:1323/uploads")
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Serve static files from uploads directory
	e.Static("/uploads", "./uploads")

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(fileStorage)
	authHandler := handlers.NewAuthHandler(pool, logger)

	// Public routes
	e.POST("/api/auth/register", authHandler.Register)
	e.POST("/api/auth/login", authHandler.Login)

	// Protected routes (require session)
	protected := e.Group("/api")
	protected.Use(session.SessionMiddleware(pool))
	protected.POST("/upload", uploadHandler.UploadImage)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/profile", authHandler.UpdateProfile)

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))

	// Start server in a goroutine
	go func() {
		logger.Info("starting server", slog.String("address", ":1323"))
		if err := e.Start(":1323"); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown Echo server
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.String("err", err.Error()))
	}

	// Close database pool
	pool.Close()
	logger.Info("database pool closed")

	logger.Info("server exited gracefully")
}
