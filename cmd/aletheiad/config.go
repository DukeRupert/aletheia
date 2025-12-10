package main

import (
	"fmt"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server settings
	Host        string
	Port        int
	Environment string
	LogLevel    string

	// Database settings
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string

	// Auth settings
	JWTSecret     string
	JWTExpiration time.Duration

	// Session settings
	SessionCookieName string
	SessionDuration   time.Duration
	SessionSecure     bool

	// Email settings
	EmailProvider        string
	EmailPostmarkToken   string
	EmailPostmarkAccount string
	EmailFromAddress     string
	EmailFromName        string
	EmailVerifyBaseURL   string

	// Storage settings
	StorageProvider  string
	StorageLocalPath string
	StorageLocalURL  string
	StorageS3Bucket  string
	StorageS3Region  string
	StorageS3BaseURL string

	// AI settings
	AIProvider     string
	AIClaudeAPIKey string
	AIClaudeModel  string
	AIMaxTokens    int
	AITemperature  float64

	// Queue settings
	QueueProvider          string
	QueueWorkerCount       int
	QueuePollInterval      time.Duration
	QueueJobTimeout        time.Duration
	QueueEnableRateLimits  bool
	QueueShutdownTimeout   time.Duration
	QueueCleanupInterval   time.Duration
	QueueCleanupRetention  time.Duration
	QueueMaxJobsPerHour    int
	QueueMaxConcurrentJobs int
}

// LoadConfig loads configuration from environment variables.
func LoadConfig(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		// Server settings
		Host:        envString(getenv, "SERVER_HOST", "localhost"),
		Port:        envInt(getenv, "SERVER_PORT", 8080),
		Environment: envString(getenv, "ENVIRONMENT", "dev"),
		LogLevel:    envString(getenv, "LOG_LEVEL", "info"),

		// Database settings
		DBUser:     envString(getenv, "DB_USER", "postgres"),
		DBPassword: envString(getenv, "DB_PASSWORD", ""),
		DBHost:     envString(getenv, "DB_HOSTNAME", "localhost"),
		DBPort:     envString(getenv, "DB_PORT", "5432"),
		DBName:     envString(getenv, "DB_NAME", "postgres"),

		// Auth settings
		JWTSecret:     envString(getenv, "JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration: 7 * 24 * time.Hour,

		// Session settings
		SessionCookieName: "session_token",
		SessionDuration:   7 * 24 * time.Hour,

		// Email settings
		EmailProvider:        envString(getenv, "EMAIL_PROVIDER", "mock"),
		EmailPostmarkToken:   envString(getenv, "POSTMARK_SERVER_TOKEN", ""),
		EmailPostmarkAccount: envString(getenv, "POSTMARK_ACCOUNT_TOKEN", ""),
		EmailFromAddress:     envString(getenv, "EMAIL_FROM_ADDRESS", "noreply@example.com"),
		EmailFromName:        envString(getenv, "EMAIL_FROM_NAME", "Aletheia"),
		EmailVerifyBaseURL:   envString(getenv, "EMAIL_VERIFY_BASE_URL", "http://localhost:1323"),

		// Storage settings
		StorageProvider:  envString(getenv, "STORAGE_PROVIDER", "local"),
		StorageLocalPath: envString(getenv, "STORAGE_LOCAL_PATH", "./uploads"),
		StorageLocalURL:  envString(getenv, "STORAGE_LOCAL_URL", "http://localhost:1323/uploads"),
		StorageS3Bucket:  envString(getenv, "STORAGE_S3_BUCKET", ""),
		StorageS3Region:  envString(getenv, "STORAGE_S3_REGION", "us-east-1"),
		StorageS3BaseURL: envString(getenv, "STORAGE_S3_BASE_URL", ""),

		// AI settings
		AIProvider:     envString(getenv, "AI_PROVIDER", "mock"),
		AIClaudeAPIKey: envString(getenv, "CLAUDE_API_KEY", ""),
		AIClaudeModel:  envString(getenv, "CLAUDE_MODEL", "claude-3-5-sonnet-20241022"),
		AIMaxTokens:    envInt(getenv, "AI_MAX_TOKENS", 4096),
		AITemperature:  envFloat(getenv, "AI_TEMPERATURE", 0.3),

		// Queue settings
		QueueProvider:          envString(getenv, "QUEUE_PROVIDER", "postgres"),
		QueueWorkerCount:       envInt(getenv, "QUEUE_WORKER_COUNT", 3),
		QueuePollInterval:      envDuration(getenv, "QUEUE_POLL_INTERVAL", time.Second),
		QueueJobTimeout:        envDuration(getenv, "QUEUE_JOB_TIMEOUT", 60*time.Second),
		QueueEnableRateLimits:  envBool(getenv, "QUEUE_ENABLE_RATE_LIMITING", true),
		QueueShutdownTimeout:   10 * time.Second,
		QueueCleanupInterval:   time.Hour,
		QueueCleanupRetention:  7 * 24 * time.Hour,
		QueueMaxJobsPerHour:    envInt(getenv, "QUEUE_MAX_JOBS_PER_HOUR", 100),
		QueueMaxConcurrentJobs: envInt(getenv, "QUEUE_MAX_CONCURRENT_JOBS", 10),
	}

	// Session secure only in production
	cfg.SessionSecure = cfg.Environment == "prod" || cfg.Environment == "production"

	// Validate production requirements
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DatabaseURL returns the PostgreSQL connection string.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// validate checks production requirements.
func (c *Config) validate() error {
	if c.Environment == "prod" || c.Environment == "production" {
		if c.JWTSecret == "your-secret-key-change-in-production" {
			return fmt.Errorf("JWT_SECRET must be set in production environment")
		}
	}
	return nil
}

// Helper functions for loading environment variables with defaults.

func envString(getenv func(string) string, key, defaultValue string) string {
	if value := getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func envInt(getenv func(string) string, key string, defaultValue int) int {
	if value := getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func envFloat(getenv func(string) string, key string, defaultValue float64) float64 {
	if value := getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func envBool(getenv func(string) string, key string, defaultValue bool) bool {
	if value := getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func envDuration(getenv func(string) string, key string, defaultValue time.Duration) time.Duration {
	if value := getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
