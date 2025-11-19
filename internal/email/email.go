package email

import (
	"log/slog"
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendVerificationEmail(to, token string) error
	SendPasswordResetEmail(to, token string) error
}

// MockEmailService is a mock implementation that logs instead of sending emails
type MockEmailService struct {
	logger *slog.Logger
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService(logger *slog.Logger) *MockEmailService {
	return &MockEmailService{
		logger: logger,
	}
}

// SendVerificationEmail logs the verification email instead of sending it
func (s *MockEmailService) SendVerificationEmail(to, token string) error {
	s.logger.Info("📧 MOCK EMAIL: Verification email",
		slog.String("to", to),
		slog.String("token", token),
		slog.String("verify_url", "http://localhost:1323/verify?token="+token),
	)
	return nil
}

// SendPasswordResetEmail logs the password reset email instead of sending it
func (s *MockEmailService) SendPasswordResetEmail(to, token string) error {
	s.logger.Info("📧 MOCK EMAIL: Password reset email",
		slog.String("to", to),
		slog.String("token", token),
		slog.String("reset_url", "http://localhost:1323/reset-password?token="+token),
	)
	return nil
}
