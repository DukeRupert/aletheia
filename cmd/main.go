package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia/internal/config"

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

	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

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

	e.Logger.Fatal(e.Start(":1323"))
}
