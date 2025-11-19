package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewQueue creates a new queue instance based on the provider configuration
func NewQueue(ctx context.Context, logger *slog.Logger, cfg Config) (Queue, error) {
	switch cfg.Provider {
	case "postgres":
		pool, err := pgxpool.New(ctx, cfg.PostgresConnectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to ping postgres: %w", err)
		}

		logger.Info("initialized PostgreSQL queue",
			slog.Int("worker_count", cfg.WorkerCount),
			slog.Duration("poll_interval", cfg.PollInterval),
		)

		return NewPostgresQueue(pool, logger, cfg), nil

	case "redis":
		// Future implementation
		return nil, fmt.Errorf("redis queue not yet implemented")

	default:
		return nil, fmt.Errorf("unknown queue provider: %s", cfg.Provider)
	}
}
