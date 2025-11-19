package config

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Session  SessionConfig
	Email    EmailConfig
	Logger   LoggerConfig
}

type AppConfig struct {
	Host string
	Port int
	Env  string
}

type DatabaseConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	DbName   string
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

type EmailConfig struct {
	Provider         string // "mock" or "postmark"
	PostmarkToken    string
	PostmarkAccount  string
	FromAddress      string
	FromName         string
	VerifyBaseURL    string // Base URL for verification links (e.g., "http://localhost:1323")
}

type LoggerConfig struct {
	Level slog.Level
}

var (
	flagsInitialized = false
	flagHost         *string
	flagPort         *int
	flagEnv          *string
	flagLogLevel     *string
	flagDbUser       *string
	flagDbPassword   *string
	flagDbHost       *string
	flagDbPort       *string
	flagDbName       *string
)

func initFlags() {
	if !flagsInitialized {
		flagHost = flag.String("host", getEnv("SERVER_HOST", "localhost"), "server host")
		flagPort = flag.Int("port", getEnvInt("SERVER_PORT", 8080), "server port")
		flagEnv = flag.String("env", getEnv("ENVIRONMENT", "prod"), "environment: prod, dev")
		flagLogLevel = flag.String("log_level", getEnv("LOG_LEVEL", "info"), "log level: debug, info, warn, error")
		flagDbUser = flag.String("db_user", getEnv("DB_USER", "postgres"), "postgres database username")
		flagDbPassword = flag.String("db_password", getEnv("DB_PASSWORD", ""), "postgres database password")
		flagDbHost = flag.String("db_hostname", getEnv("DB_HOSTNAME", "localhost"), "postgres database hostname")
		flagDbPort = flag.String("db_port", getEnv("DB_PORT", "5432"), "postgres database port")
		flagDbName = flag.String("db_name", getEnv("DB_NAME", "postgres"), "postgres database name")
		flagsInitialized = true
	}
	if !flag.Parsed() {
		flag.Parse()
	}
}

// Load loads configuration from environment variables and command-line flags
func Load() (*Config, error) {
	// Try to load .env from current directory, then walk up to find it (max 2 levels)
	err := godotenv.Load()
	if err != nil {
		// Walk up directories to find .env (max 2 parent directories)
		dir, _ := os.Getwd()
		found := false
		for i := 0; i < 2; i++ {
			dir = filepath.Join(dir, "..")
			if err := godotenv.Load(filepath.Join(dir, ".env")); err == nil {
				found = true
				break
			}
		}
		if !found {
			log.Println("Warning: .env file not found, using environment variables and defaults")
		}
	}

	// Initialize and parse flags
	initFlags()

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
		App: AppConfig{
			Host: *flagHost,
			Port: *flagPort,
			Env:  *flagEnv,
		},
		Database: DatabaseConfig{
			Username: *flagDbUser,
			Password: *flagDbPassword,
			Host:     *flagDbHost,
			Port:     *flagDbPort,
			DbName:   *flagDbName,
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
		Email: EmailConfig{
			Provider:        getEnv("EMAIL_PROVIDER", "mock"),
			PostmarkToken:   getEnv("POSTMARK_SERVER_TOKEN", ""),
			PostmarkAccount: getEnv("POSTMARK_ACCOUNT_TOKEN", ""),
			FromAddress:     getEnv("EMAIL_FROM_ADDRESS", "noreply@example.com"),
			FromName:        getEnv("EMAIL_FROM_NAME", "Aletheia"),
			VerifyBaseURL:   getEnv("EMAIL_VERIFY_BASE_URL", "http://localhost:1323"),
		},
		Logger: LoggerConfig{
			Level: programLevel.Level(),
		},
	}

	// Validate JWT secret in production
	if (cfg.App.Env == "prod" || cfg.App.Env == "production") && cfg.Auth.JWTSecret == "your-secret-key-change-in-production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production environment")
	}

	return cfg, nil
}

func (c *Config) GetLogger() slog.Handler {
	var handler slog.Handler
	logLevel := c.Logger.Level
	switch c.App.Env {
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

func (c *Config) GetConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", c.Database.Username, c.Database.Password, c.Database.Host, c.Database.Port, c.Database.DbName)
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
