package main

import (
	"context"
	"log/slog"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Services holds all application services.
type Services struct {
	UserService         aletheia.UserService
	SessionService      aletheia.SessionService
	OrganizationService aletheia.OrganizationService
	ProjectService      aletheia.ProjectService
	InspectionService   aletheia.InspectionService
	PhotoService        aletheia.PhotoService
	ViolationService    aletheia.ViolationService
	SafetyCodeService   aletheia.SafetyCodeService
	FileStorage         aletheia.FileStorage
	EmailService        aletheia.EmailService
	AIService           aletheia.AIService
	Queue               aletheia.Queue
}

// initServices initializes all application services.
func initServices(ctx context.Context, pool *pgxpool.Pool, cfg *Config, logger *slog.Logger) (*Services, error) {
	// Initialize database wrapper with all domain services
	db := postgres.NewDB(pool)
	logger.Info("database services initialized")

	// Initialize file storage
	fileStorage, err := initFileStorage(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	logger.Info("file storage initialized", slog.String("provider", cfg.StorageProvider))

	// Initialize email service
	emailService := initEmailService(cfg, logger)
	logger.Info("email service initialized", slog.String("provider", cfg.EmailProvider))

	// Initialize AI service
	aiService := initAIService(cfg, logger)
	logger.Info("AI service initialized", slog.String("provider", cfg.AIProvider))

	// Initialize queue
	queue := initQueue(pool, cfg, logger)
	logger.Info("queue service initialized", slog.String("provider", cfg.QueueProvider))

	return &Services{
		UserService:         db.UserService,
		SessionService:      db.SessionService,
		OrganizationService: db.OrganizationService,
		ProjectService:      db.ProjectService,
		InspectionService:   db.InspectionService,
		PhotoService:        db.PhotoService,
		ViolationService:    db.ViolationService,
		SafetyCodeService:   db.SafetyCodeService,
		FileStorage:         fileStorage,
		EmailService:        emailService,
		AIService:           aiService,
		Queue:               queue,
	}, nil
}

// initFileStorage creates the appropriate file storage implementation.
func initFileStorage(ctx context.Context, cfg *Config, logger *slog.Logger) (aletheia.FileStorage, error) {
	logger.Debug("storage service configuration",
		slog.String("provider", cfg.StorageProvider),
		slog.String("local_path", cfg.StorageLocalPath),
		slog.String("s3_bucket", cfg.StorageS3Bucket),
		slog.String("s3_region", cfg.StorageS3Region))

	storageCfg := aletheia.StorageConfig{
		Provider:  cfg.StorageProvider,
		LocalPath: cfg.StorageLocalPath,
		LocalURL:  cfg.StorageLocalURL,
		S3Bucket:  cfg.StorageS3Bucket,
		S3Region:  cfg.StorageS3Region,
		S3BaseURL: cfg.StorageS3BaseURL,
	}

	return postgres.NewFileStorage(ctx, logger, storageCfg)
}

// initEmailService creates the appropriate email service implementation.
func initEmailService(cfg *Config, logger *slog.Logger) aletheia.EmailService {
	logger.Debug("email service configuration",
		slog.String("provider", cfg.EmailProvider),
		slog.String("from_address", cfg.EmailFromAddress),
		slog.String("from_name", cfg.EmailFromName))

	emailCfg := aletheia.EmailConfig{
		Provider:            cfg.EmailProvider,
		FromAddress:         cfg.EmailFromAddress,
		FromName:            cfg.EmailFromName,
		VerifyBaseURL:       cfg.EmailVerifyBaseURL,
		PostmarkServerToken: cfg.EmailPostmarkToken,
	}

	return postgres.NewEmailService(logger, emailCfg)
}

// initAIService creates the appropriate AI service implementation.
func initAIService(cfg *Config, logger *slog.Logger) aletheia.AIService {
	logger.Debug("AI service configuration",
		slog.String("provider", cfg.AIProvider))

	aiCfg := aletheia.AIConfig{
		Provider:     cfg.AIProvider,
		ClaudeAPIKey: cfg.AIClaudeAPIKey,
		ClaudeModel:  cfg.AIClaudeModel,
	}

	return postgres.NewAIService(logger, aiCfg)
}

// initQueue creates the queue implementation.
func initQueue(pool *pgxpool.Pool, cfg *Config, logger *slog.Logger) aletheia.Queue {
	logger.Debug("initializing queue service")

	queueCfg := aletheia.QueueConfig{
		Provider:           cfg.QueueProvider,
		WorkerCount:        cfg.QueueWorkerCount,
		PollInterval:       cfg.QueuePollInterval,
		JobTimeout:         cfg.QueueJobTimeout,
		EnableRateLimiting: cfg.QueueEnableRateLimits,
	}

	return postgres.NewQueue(pool, logger, queueCfg)
}
