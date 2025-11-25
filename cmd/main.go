package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukerupert/aletheia/internal/ai"
	"github.com/dukerupert/aletheia/internal/audit"
	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/email"
	apperrors "github.com/dukerupert/aletheia/internal/errors"
	"github.com/dukerupert/aletheia/internal/handlers"
	intmiddleware "github.com/dukerupert/aletheia/internal/middleware"
	"github.com/dukerupert/aletheia/internal/migrations"
	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/dukerupert/aletheia/internal/templates"
	"github.com/dukerupert/aletheia/internal/validation"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	logger.Debug("application configuration",
		slog.String("environment", cfg.App.Env),
		slog.String("host", cfg.App.Host),
		slog.Int("port", cfg.App.Port))

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

	// Run migrations
	logger.Info("running database migrations...")
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal("failed to set goose dialect:", err)
	}

	// Get stdlib sql.DB from pgx pool for goose
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	if err := goose.Up(sqlDB, "."); err != nil {
		log.Fatal("failed to run migrations:", err)
	}
	logger.Info("database migrations completed")

	// Initialize storage service (configured via STORAGE_PROVIDER env var)
	logger.Debug("storage service configuration",
		slog.String("provider", cfg.Storage.Provider),
		slog.String("local_path", cfg.Storage.LocalPath),
		slog.String("s3_bucket", cfg.Storage.S3Bucket),
		slog.String("s3_region", cfg.Storage.S3Region))
	fileStorage, err := storage.NewFileStorage(context.Background(), logger, storage.StorageConfig{
		Provider:  cfg.Storage.Provider,
		LocalPath: cfg.Storage.LocalPath,
		LocalURL:  cfg.Storage.LocalURL,
		S3Bucket:  cfg.Storage.S3Bucket,
		S3Region:  cfg.Storage.S3Region,
		S3BaseURL: cfg.Storage.S3BaseURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Initialize email service (configured via EMAIL_PROVIDER env var)
	logger.Debug("email service configuration",
		slog.String("provider", cfg.Email.Provider),
		slog.String("from_address", cfg.Email.FromAddress),
		slog.String("from_name", cfg.Email.FromName))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:        cfg.Email.Provider,
		PostmarkToken:   cfg.Email.PostmarkToken,
		PostmarkAccount: cfg.Email.PostmarkAccount,
		FromAddress:     cfg.Email.FromAddress,
		FromName:        cfg.Email.FromName,
		VerifyBaseURL:   cfg.Email.VerifyBaseURL,
	})
	logger.Info("email service initialized", slog.String("provider", cfg.Email.Provider))

	// Initialize AI service (configured via AI_PROVIDER env var)
	logger.Debug("AI service configuration",
		slog.String("provider", cfg.AI.Provider))
	aiService := ai.NewAIService(logger, ai.AIConfig{
		Provider:     cfg.AI.Provider,
		ClaudeAPIKey: cfg.AI.ClaudeAPIKey,
		ClaudeModel:  cfg.AI.ClaudeModel,
		MaxTokens:    cfg.AI.MaxTokens,
		Temperature:  cfg.AI.Temperature,
	})
	logger.Info("AI service initialized", slog.String("provider", cfg.AI.Provider))

	// Initialize queue (using Postgres with existing pool)
	logger.Debug("initializing queue service")
	queueConfig := queue.Config{
		Provider:                 "postgres",
		WorkerCount:              3,
		PollInterval:             time.Second,
		JobTimeout:               60 * time.Second,
		EnableRateLimiting:       true,
		ShutdownTimeout:          10 * time.Second,
		CleanupInterval:          time.Hour,
		CleanupRetention:         7 * 24 * time.Hour,
		DefaultMaxJobsPerHour:    100,
		DefaultMaxConcurrentJobs: 10,
	}
	queueService := queue.NewPostgresQueue(pool, logger, queueConfig)
	logger.Info("queue service initialized")

	// Initialize template renderer
	logger.Debug("initializing template renderer", slog.String("path", "web/templates"))
	renderer, err := templates.NewTemplateRenderer("web/templates")
	if err != nil {
		log.Fatal("failed to initialize templates:", err)
	}
	logger.Info("template renderer initialized")

	// Initialize Prometheus metrics
	logger.Debug("initializing Prometheus metrics")
	intmiddleware.InitMetrics()
	logger.Info("metrics initialized")

	// Initialize session cache for improved performance
	logger.Debug("initializing session cache")
	sessionCache := session.NewSessionCache(pool)
	logger.Info("session cache initialized", slog.Int("ttl_minutes", 5))

	// Initialize audit logger for compliance and security tracking
	logger.Debug("initializing audit logger")
	auditLogger := audit.NewAuditLogger(pool, logger)
	logger.Info("audit logger initialized")

	// TODO: Uncomment to enable scheduled audit log cleanup (runs daily at 2 AM)
	// Retention period: 2555 days (7 years - common compliance requirement)
	// go func() {
	// 	ticker := time.NewTicker(24 * time.Hour)
	// 	defer ticker.Stop()
	// 	for range ticker.C {
	// 		// Run cleanup at 2 AM
	// 		now := time.Now()
	// 		if now.Hour() == 2 {
	// 			ctx := context.Background()
	// 			deleted, err := auditLogger.CleanupOldAuditLogs(ctx, 2555)
	// 			if err != nil {
	// 				logger.Error("audit log cleanup failed", slog.String("error", err.Error()))
	// 			} else {
	// 				logger.Info("audit log cleanup completed", slog.Int64("deleted", deleted))
	// 			}
	// 		}
	// 	}
	// }()

	logger.Debug("initializing Echo web server")
	e := echo.New()
	e.Renderer = renderer
	e.Validator = validation.NewValidator()
	logger.Info("input validator initialized")

	// Error handling middleware - must be first to catch all errors and panics
	logger.Debug("configuring error handling middleware")
	e.Use(apperrors.ErrorHandlerMiddleware(logger))
	logger.Info("error handling middleware initialized")

	// Request ID middleware - must be early in the chain for tracing
	logger.Debug("configuring request ID middleware")
	e.Use(intmiddleware.RequestIDMiddleware(logger))

	// Metrics middleware - must be after request ID for proper correlation
	logger.Debug("configuring metrics middleware")
	e.Use(intmiddleware.MetricsMiddleware())

	// Rate limiting middleware - protect against abuse and DoS attacks
	logger.Debug("configuring rate limiting middleware")
	rateLimiter := intmiddleware.NewRateLimiter(logger)
	e.Use(rateLimiter.Middleware())
	logger.Info("rate limiting initialized",
		slog.Float64("rate_per_sec", 100.0),
		slog.Int("burst", 200))

	// HTMX response middleware
	logger.Debug("configuring HTMX middleware")
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set HX-Request header detection
			if c.Request().Header.Get("HX-Request") == "true" {
				c.Set("IsHTMX", true)
			}
			return next(c)
		}
	})

	// Serve static files
	logger.Debug("configuring static file routes",
		slog.String("css_js_images", "/static"),
		slog.String("uploads", "/uploads"))
	e.Static("/static", "web/static")
	e.Static("/uploads", "./uploads")

	// Initialize handlers
	logger.Debug("initializing HTTP handlers")
	queries := database.New(pool)
	pageHandler := handlers.NewPageHandler(pool, logger)
	uploadHandler := handlers.NewUploadHandler(fileStorage, pool, logger)

	// TODO: Pass auditLogger to handlers that need audit logging
	// These handlers perform state-changing operations and should log:
	// - authHandler: user registration, login, logout, password changes
	// - orgHandler: organization CRUD, member management
	// - projectHandler: project CRUD
	// - inspectionHandler: inspection CRUD, status changes
	// - safetyCodeHandler: safety code CRUD
	// - photoHandler: photo uploads, deletions
	// - violationHandler: violation confirmations, dismissals, manual creation
	//
	// Example updated constructor:
	//   authHandler := handlers.NewAuthHandler(pool, logger, emailService, auditLogger)
	//
	// Then in handler methods, log audit entries:
	//   auditLogger.LogCreate(ctx, userID, orgID, "user", user.ID,
	//       map[string]interface{}{"email": user.Email}, c)

	authHandler := handlers.NewAuthHandler(pool, logger, emailService, cfg)
	orgHandler := handlers.NewOrganizationHandler(pool, logger)
	projectHandler := handlers.NewProjectHandler(pool, logger)
	inspectionHandler := handlers.NewInspectionHandler(pool, logger)
	safetyCodeHandler := handlers.NewSafetyCodeHandler(pool, logger)
	photoHandler := handlers.NewPhotoHandler(pool, queries, queueService, logger)
	violationHandler := handlers.NewViolationHandler(pool, queries, logger)
	jobHandler := handlers.NewJobHandler(pool, queries, queueService, logger)
	healthHandler := handlers.NewHealthHandler(pool, fileStorage, queueService, logger)

	// Suppress unused variable warning until audit logging is integrated
	_ = auditLogger

	logger.Info("all handlers initialized")

	// Register queue job handlers
	logger.Debug("registering queue job handlers")
	workerPool := queue.NewWorkerPool(queueService, logger, queueConfig)
	photoAnalysisJobHandler := handlers.NewPhotoAnalysisJobHandler(queries, aiService, fileStorage, logger)
	workerPool.RegisterHandler("analyze_photo", photoAnalysisJobHandler.Handle)
	logger.Info("queue job handlers registered")

	// Start worker pool
	logger.Debug("starting worker pool")
	go workerPool.Start(context.Background(), []string{"photo_analysis"})
	logger.Info("worker pool started")

	// Health check endpoints
	logger.Debug("configuring health check endpoints")
	e.GET("/health", healthHandler.HealthCheck)                  // Basic uptime check
	e.GET("/health/live", healthHandler.LivenessCheck)           // Kubernetes liveness probe
	e.GET("/health/ready", healthHandler.ReadinessCheck)         // Kubernetes readiness probe
	e.GET("/health/detailed", healthHandler.DetailedHealthCheck) // Detailed system info

	// Prometheus metrics endpoint
	logger.Debug("configuring Prometheus metrics endpoint")
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Page routes (public)
	e.GET("/", pageHandler.HomePage)
	e.GET("/login", pageHandler.LoginPage)
	e.GET("/register", pageHandler.RegisterPage)
	e.GET("/verify", pageHandler.VerifyEmailPage)
	e.GET("/forgot-password", pageHandler.ForgotPasswordPage)
	e.GET("/reset-password", pageHandler.ResetPasswordPage)

	// Public API routes - Auth endpoints with strict rate limiting
	logger.Debug("configuring auth endpoints with strict rate limiting")
	strictLimiter := intmiddleware.NewStrictRateLimiter(logger)
	authGroup := e.Group("/api/auth")
	authGroup.Use(strictLimiter.Middleware())
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/verify-email", authHandler.VerifyEmail)
	authGroup.POST("/resend-verification", authHandler.ResendVerification)
	authGroup.POST("/request-password-reset", authHandler.RequestPasswordReset)
	authGroup.POST("/verify-reset-token", authHandler.VerifyResetToken)
	authGroup.POST("/reset-password", authHandler.ResetPassword)
	logger.Info("auth endpoints configured with strict rate limiting",
		slog.Float64("rate_per_min", 5.0),
		slog.Int("burst", 10))

	// Protected page routes (require session)
	protectedPages := e.Group("")
	protectedPages.Use(session.CachedSessionMiddleware(sessionCache))
	protectedPages.GET("/dashboard", pageHandler.DashboardPage)
	protectedPages.GET("/profile", pageHandler.ProfilePage)
	protectedPages.GET("/organizations", pageHandler.OrganizationsPage)
	protectedPages.GET("/organizations/new", pageHandler.NewOrganizationPage)
	protectedPages.GET("/projects", pageHandler.ProjectsPage)
	protectedPages.GET("/projects/new", pageHandler.NewProjectPage)
	protectedPages.GET("/projects/:id", pageHandler.ProjectDetailPage)
	protectedPages.GET("/inspections", pageHandler.AllInspectionsPage)
	protectedPages.GET("/inspections/:id", pageHandler.InspectionDetailPage)
	protectedPages.GET("/projects/:projectId/inspections", pageHandler.InspectionsPage)
	protectedPages.GET("/projects/:projectId/inspections/new", pageHandler.NewInspectionPage)
	protectedPages.GET("/photos/:id", pageHandler.PhotoDetailPage)

	// Protected API routes (require session)
	logger.Debug("configuring protected API routes with per-user rate limiting")
	userLimiter := intmiddleware.NewPerUserRateLimiter(logger)
	protected := e.Group("/api")
	protected.Use(session.CachedSessionMiddleware(sessionCache))
	protected.Use(userLimiter.Middleware())
	logger.Info("protected API routes configured with per-user rate limiting",
		slog.Float64("rate_per_min", 100.0),
		slog.Int("burst", 150))
	protected.POST("/upload", uploadHandler.UploadImage)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/profile", authHandler.UpdateProfile)

	// Organization routes
	protected.POST("/organizations", orgHandler.CreateOrganization)
	protected.GET("/organizations", orgHandler.ListOrganizations)
	protected.GET("/organizations/:id", orgHandler.GetOrganization)
	protected.PUT("/organizations/:id", orgHandler.UpdateOrganization)
	protected.DELETE("/organizations/:id", orgHandler.DeleteOrganization)

	// Organization member routes
	protected.GET("/organizations/:id/members", orgHandler.ListOrganizationMembers)
	protected.POST("/organizations/:id/members", orgHandler.AddOrganizationMember)
	protected.PUT("/organizations/:id/members/:memberId", orgHandler.UpdateOrganizationMember)
	protected.DELETE("/organizations/:id/members/:memberId", orgHandler.RemoveOrganizationMember)

	// Project routes
	protected.POST("/projects", projectHandler.CreateProject)
	protected.GET("/projects/:id", projectHandler.GetProject)
	protected.GET("/organizations/:orgId/projects", projectHandler.ListProjects)
	protected.PUT("/projects/:id", projectHandler.UpdateProject)
	protected.DELETE("/projects/:id", projectHandler.DeleteProject)

	// Inspection routes
	protected.POST("/inspections", inspectionHandler.CreateInspection)
	protected.GET("/inspections/:id", inspectionHandler.GetInspection)
	protected.GET("/projects/:projectId/inspections", inspectionHandler.ListInspections)
	protected.PUT("/inspections/:id/status", inspectionHandler.UpdateInspectionStatus)

	// Photo routes
	protected.GET("/inspections/:inspectionId/photos", uploadHandler.ListPhotos)
	protected.GET("/photos/:id", uploadHandler.GetPhoto)
	protected.GET("/photos/:photo_id/status", photoHandler.GetPhotoStatus)
	protected.DELETE("/photos/:photo_id", photoHandler.DeletePhoto)
	protected.POST("/photos/analyze", photoHandler.AnalyzePhoto)
	protected.GET("/photos/analyze/:job_id", photoHandler.GetPhotoAnalysisStatus)

	// Safety code routes
	protected.POST("/safety-codes", safetyCodeHandler.CreateSafetyCode)
	protected.GET("/safety-codes", safetyCodeHandler.ListSafetyCodes)
	protected.GET("/safety-codes/:id", safetyCodeHandler.GetSafetyCode)
	protected.PUT("/safety-codes/:id", safetyCodeHandler.UpdateSafetyCode)
	protected.DELETE("/safety-codes/:id", safetyCodeHandler.DeleteSafetyCode)

	// Violation review routes
	protected.POST("/violations/:id/confirm", violationHandler.ConfirmViolation)
	protected.POST("/violations/:id/dismiss", violationHandler.DismissViolation)
	protected.POST("/violations/:id/pending", violationHandler.SetViolationPending)
	protected.PATCH("/violations/:violation_id", violationHandler.UpdateViolation)
	protected.POST("/violations/manual", violationHandler.CreateManualViolation)
	protected.GET("/inspections/:inspection_id/violations", violationHandler.ListViolationsByInspection)

	// Job status routes
	protected.GET("/jobs/status", jobHandler.GetJobStatus)
	protected.POST("/jobs/:job_id/cancel", jobHandler.CancelJob)

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Use request-scoped logger that includes request_id
			requestLogger := intmiddleware.GetRequestLogger(c)

			if v.Error == nil {
				requestLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				requestLogger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
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
		address := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
		logger.Info("starting server", slog.String("address", address))
		if err := e.Start(address); err != nil && err != http.ErrServerClosed {
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

	// Shutdown rate limiter cleanup goroutines
	rateLimiter.Shutdown()
	strictLimiter.Shutdown()
	userLimiter.Shutdown()
	logger.Info("rate limiters shutdown")

	// Close database pool
	pool.Close()
	logger.Info("database pool closed")

	logger.Info("server exited gracefully")
}
