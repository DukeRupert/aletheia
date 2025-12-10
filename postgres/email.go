package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dukerupert/aletheia"
)

// Compile-time interface check
var _ aletheia.EmailService = (*MockEmailService)(nil)

// NewEmailService creates an email service based on the provider configuration.
func NewEmailService(logger *slog.Logger, cfg aletheia.EmailConfig) aletheia.EmailService {
	switch cfg.Provider {
	case "postmark":
		// TODO: Implement Postmark email service
		logger.Warn("postmark email service not yet implemented, falling back to mock")
		return &MockEmailService{logger: logger, cfg: cfg}
	default:
		return &MockEmailService{logger: logger, cfg: cfg}
	}
}

// MockEmailService is a mock implementation that logs instead of sending emails.
type MockEmailService struct {
	logger *slog.Logger
	cfg    aletheia.EmailConfig
}

// SendVerificationEmail logs the verification email instead of sending it.
func (s *MockEmailService) SendVerificationEmail(ctx context.Context, to, name, token string) error {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.cfg.VerifyBaseURL, token)
	s.logger.Info("MOCK EMAIL: Verification email",
		slog.String("to", to),
		slog.String("name", name),
		slog.String("token", token),
		slog.String("verify_url", verifyURL))
	return nil
}

// SendPasswordResetEmail logs the password reset email instead of sending it.
func (s *MockEmailService) SendPasswordResetEmail(ctx context.Context, to, name, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.VerifyBaseURL, token)
	s.logger.Info("MOCK EMAIL: Password reset email",
		slog.String("to", to),
		slog.String("name", name),
		slog.String("token", token),
		slog.String("reset_url", resetURL))
	return nil
}

// SendWelcomeEmail logs the welcome email instead of sending it.
func (s *MockEmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	s.logger.Info("MOCK EMAIL: Welcome email",
		slog.String("to", to),
		slog.String("name", name))
	return nil
}

// SendInspectionReport logs the inspection report email instead of sending it.
func (s *MockEmailService) SendInspectionReport(ctx context.Context, to []string, subject string, reportURL string) error {
	s.logger.Info("MOCK EMAIL: Inspection report",
		slog.Any("to", to),
		slog.String("subject", subject),
		slog.String("report_url", reportURL))
	return nil
}
