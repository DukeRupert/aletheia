package config

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Session  SessionConfig
	Logger   LoggerConfig
}

type ServerConfig struct {
	Host string
	Port int
	Env  string
}

type DatabaseConfig struct {
	Path string
}

type AuthConfig struct {
	JWTSecret     string
	JWTExpiration time.Duration
}

type SessionConfig struct {
	CookieName string
	Duration   time.Duration
	Secure     bool
}

type LoggerConfig struct {
	Level slog.Level
}

// Load loads configuration from environment variables and command-line flags
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Command-line flags
	var flagHost = flag.String("host", getEnv("SERVER_HOST", "localhost"), "server host")
	var flagPort = flag.Int("port", getEnvInt("SERVER_PORT", 8080), "server port")
	var flagEnv = flag.String("env", getEnv("ENVIRONMENT", "prod"), "environment: prod, dev")
	var flagLogLevel = flag.String("log_level", getEnv("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	var flagDatabase = flag.String("database", getEnv("DATABASE_PATH", "sqlite.db"), "sqlite database file path")
	flag.Parse()

	// Set up logging
	var programLevel = new(slog.LevelVar) // Info by default
	switch *flagLogLevel {
	case "error":
		programLevel.Set(slog.LevelError)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "debug":
		programLevel.Set(slog.LevelDebug)
	default:
		programLevel.Set(slog.LevelInfo)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: *flagHost,
			Port: *flagPort,
			Env:  *flagEnv,
		},
		Database: DatabaseConfig{
			Path: *flagDatabase,
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			JWTExpiration: 24 * time.Hour * 7, // 7 days
		},
		Session: SessionConfig{
			CookieName: "session_token",
			Duration:   24 * time.Hour * 7, // 7 days
			Secure:     *flagEnv == "prod" || *flagEnv == "production",
		},
		Logger: LoggerConfig{
			Level: programLevel.Level(),
		},
	}

	// Validate JWT secret in production
	if (cfg.Server.Env == "prod" || cfg.Server.Env == "production") && cfg.Auth.JWTSecret == "your-secret-key-change-in-production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production environment")
	}

	return cfg, nil
}

func (c *Config) GetLogger() slog.Handler {
	var handler slog.Handler
	logLevel := c.Logger.Level
	switch c.Server.Env {
	case "prod", "production":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: logLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String("time", a.Value.Time().Format(time.RFC3339Nano))
				}
				return a
			},
		})
	default:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	}
	return handler
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
