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
	"github.com/dukerupert/aletheia/internal/email"
	"github.com/dukerupert/aletheia/internal/handlers"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/dukerupert/aletheia/internal/templates"

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

	// Initialize storage service (configured via STORAGE_PROVIDER env var)
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
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:        cfg.Email.Provider,
		PostmarkToken:   cfg.Email.PostmarkToken,
		PostmarkAccount: cfg.Email.PostmarkAccount,
		FromAddress:     cfg.Email.FromAddress,
		FromName:        cfg.Email.FromName,
		VerifyBaseURL:   cfg.Email.VerifyBaseURL,
	})
	logger.Info("email service initialized", slog.String("provider", cfg.Email.Provider))

	// Initialize template renderer
	renderer, err := templates.NewTemplateRenderer("web/templates")
	if err != nil {
		log.Fatal("failed to initialize templates:", err)
	}

	e := echo.New()
	e.Renderer = renderer

	// HTMX response middleware
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
	e.Static("/static", "web/static")
	e.Static("/uploads", "./uploads")

	// Initialize handlers
	pageHandler := handlers.NewPageHandler()
	uploadHandler := handlers.NewUploadHandler(fileStorage, pool, logger)
	authHandler := handlers.NewAuthHandler(pool, logger, emailService)
	orgHandler := handlers.NewOrganizationHandler(pool, logger)
	projectHandler := handlers.NewProjectHandler(pool, logger)
	inspectionHandler := handlers.NewInspectionHandler(pool, logger)
	safetyCodeHandler := handlers.NewSafetyCodeHandler(pool, logger)

	// Page routes (public)
	e.GET("/", pageHandler.HomePage)
	e.GET("/login", pageHandler.LoginPage)
	e.GET("/register", pageHandler.RegisterPage)

	// Public API routes
	e.POST("/api/auth/register", authHandler.Register)
	e.POST("/api/auth/login", authHandler.Login)
	e.POST("/api/auth/verify-email", authHandler.VerifyEmail)
	e.POST("/api/auth/resend-verification", authHandler.ResendVerification)
	e.POST("/api/auth/request-password-reset", authHandler.RequestPasswordReset)
	e.POST("/api/auth/verify-reset-token", authHandler.VerifyResetToken)
	e.POST("/api/auth/reset-password", authHandler.ResetPassword)

	// Protected page routes (require session)
	protectedPages := e.Group("")
	protectedPages.Use(session.SessionMiddleware(pool))
	protectedPages.GET("/dashboard", pageHandler.DashboardPage)

	// Protected API routes (require session)
	protected := e.Group("/api")
	protected.Use(session.SessionMiddleware(pool))
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
	protected.DELETE("/photos/:id", uploadHandler.DeletePhoto)

	// Safety code routes
	protected.POST("/safety-codes", safetyCodeHandler.CreateSafetyCode)
	protected.GET("/safety-codes", safetyCodeHandler.ListSafetyCodes)
	protected.GET("/safety-codes/:id", safetyCodeHandler.GetSafetyCode)
	protected.PUT("/safety-codes/:id", safetyCodeHandler.UpdateSafetyCode)
	protected.DELETE("/safety-codes/:id", safetyCodeHandler.DeleteSafetyCode)

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
