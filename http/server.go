package http

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/labstack/echo/v4"
)

// Server represents the HTTP server with all its dependencies.
type Server struct {
	echo   *echo.Echo
	ln     net.Listener
	logger *slog.Logger

	// Configuration
	Addr   string
	Domain string

	// Session configuration
	SessionDuration time.Duration
	SessionSecure   bool

	// Domain services
	userService         aletheia.UserService
	organizationService aletheia.OrganizationService
	projectService      aletheia.ProjectService
	inspectionService   aletheia.InspectionService
	photoService        aletheia.PhotoService
	violationService    aletheia.ViolationService
	safetyCodeService   aletheia.SafetyCodeService
	sessionService      aletheia.SessionService

	// External services
	fileStorage  aletheia.FileStorage
	emailService aletheia.EmailService
	aiService    aletheia.AIService
	queue        aletheia.Queue
}

// Config holds the configuration for creating a new Server.
type Config struct {
	Addr   string
	Domain string
	Logger *slog.Logger

	// Session configuration
	SessionDuration time.Duration
	SessionSecure   bool

	// Template renderer
	Renderer echo.Renderer

	// Domain services
	UserService         aletheia.UserService
	OrganizationService aletheia.OrganizationService
	ProjectService      aletheia.ProjectService
	InspectionService   aletheia.InspectionService
	PhotoService        aletheia.PhotoService
	ViolationService    aletheia.ViolationService
	SafetyCodeService   aletheia.SafetyCodeService
	SessionService      aletheia.SessionService

	// External services
	FileStorage  aletheia.FileStorage
	EmailService aletheia.EmailService
	AIService    aletheia.AIService
	Queue        aletheia.Queue
}

// NewServer creates a new HTTP server with the given configuration.
func NewServer(cfg Config) *Server {
	s := &Server{
		Addr:                cfg.Addr,
		Domain:              cfg.Domain,
		logger:              cfg.Logger,
		SessionDuration:     cfg.SessionDuration,
		SessionSecure:       cfg.SessionSecure,
		userService:         cfg.UserService,
		organizationService: cfg.OrganizationService,
		projectService:      cfg.ProjectService,
		inspectionService:   cfg.InspectionService,
		photoService:        cfg.PhotoService,
		violationService:    cfg.ViolationService,
		safetyCodeService:   cfg.SafetyCodeService,
		sessionService:      cfg.SessionService,
		fileStorage:         cfg.FileStorage,
		emailService:        cfg.EmailService,
		aiService:           cfg.AIService,
		queue:               cfg.Queue,
	}

	// Set default session duration if not specified
	if s.SessionDuration == 0 {
		s.SessionDuration = 24 * time.Hour
	}

	s.echo = echo.New()
	s.echo.HideBanner = true
	s.echo.HidePort = true

	// Set template renderer if provided
	if cfg.Renderer != nil {
		s.echo.Renderer = cfg.Renderer
	}

	// Register middleware and routes
	s.registerMiddleware()
	s.registerRoutes()

	return s
}

// Echo returns the underlying Echo instance.
// Use sparingly - prefer registering routes through Server methods.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Open starts the HTTP server.
func (s *Server) Open() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln

	go func() {
		if err := s.echo.Server.Serve(s.ln); err != nil {
			s.logger.Error("server error", slog.String("error", err.Error()))
		}
	}()

	s.logger.Info("server started", slog.String("addr", s.Addr))
	return nil
}

// Close gracefully shuts down the HTTP server.
func (s *Server) Close(ctx context.Context) error {
	if err := s.echo.Shutdown(ctx); err != nil {
		return err
	}
	s.logger.Info("server stopped")
	return nil
}

// URL returns the URL of the server.
func (s *Server) URL() string {
	if s.ln == nil {
		return ""
	}
	return "http://" + s.ln.Addr().String()
}
