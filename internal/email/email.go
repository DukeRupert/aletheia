package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/keighl/postmark"
)

//go:embed templates/*.html
var templateFS embed.FS

var (
	verifyEmailTemplate *template.Template
	resetPasswordTemplate *template.Template
)

func init() {
	var err error
	verifyEmailTemplate, err = template.ParseFS(templateFS, "templates/verify.html")
	if err != nil {
		panic(fmt.Sprintf("failed to parse verify email template: %v", err))
	}

	resetPasswordTemplate, err = template.ParseFS(templateFS, "templates/reset-password.html")
	if err != nil {
		panic(fmt.Sprintf("failed to parse reset password template: %v", err))
	}
}

// EmailService defines the interface for sending emails
type EmailService interface {
	SendVerificationEmail(to, token string) error
	SendPasswordResetEmail(to, token string) error
}

// EmailConfig holds configuration for email services
type EmailConfig struct {
	Provider        string // "mock" or "postmark"
	PostmarkToken   string
	PostmarkAccount string
	FromAddress     string
	FromName        string
	VerifyBaseURL   string
}

// NewEmailService creates an email service based on the provider configuration
func NewEmailService(logger *slog.Logger, config EmailConfig) EmailService {
	switch config.Provider {
	case "postmark":
		return newPostmarkEmailService(logger, config)
	default:
		return newMockEmailService(logger, config)
	}
}

// mockEmailService is a mock implementation that logs instead of sending emails
type mockEmailService struct {
	logger *slog.Logger
	config EmailConfig
}

// newMockEmailService creates a new mock email service
func newMockEmailService(logger *slog.Logger, config EmailConfig) *mockEmailService {
	return &mockEmailService{
		logger: logger,
		config: config,
	}
}

// SendVerificationEmail logs the verification email instead of sending it
func (s *mockEmailService) SendVerificationEmail(to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.config.VerifyBaseURL, token)
	s.logger.Info("📧 MOCK EMAIL: Verification email",
		slog.String("to", to),
		slog.String("token", token),
		slog.String("verify_url", verifyURL),
	)
	return nil
}

// SendPasswordResetEmail logs the password reset email instead of sending it
func (s *mockEmailService) SendPasswordResetEmail(to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.VerifyBaseURL, token)
	s.logger.Info("📧 MOCK EMAIL: Password reset email",
		slog.String("to", to),
		slog.String("token", token),
		slog.String("reset_url", resetURL),
	)
	return nil
}

// postmarkEmailService sends emails via Postmark
type postmarkEmailService struct {
	client *postmark.Client
	logger *slog.Logger
	config EmailConfig
}

// newPostmarkEmailService creates a new Postmark email service
func newPostmarkEmailService(logger *slog.Logger, config EmailConfig) *postmarkEmailService {
	client := postmark.NewClient(config.PostmarkToken, config.PostmarkAccount)
	return &postmarkEmailService{
		client: client,
		logger: logger,
		config: config,
	}
}

// SendVerificationEmail sends a verification email via Postmark
func (s *postmarkEmailService) SendVerificationEmail(to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.config.VerifyBaseURL, token)

	// Render HTML body from template
	var htmlBody bytes.Buffer
	data := map[string]string{"VerifyURL": verifyURL}
	if err := verifyEmailTemplate.Execute(&htmlBody, data); err != nil {
		s.logger.Error("failed to render verification email template",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to render email template: %w", err)
	}

	email := postmark.Email{
		From:     fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress),
		To:       to,
		Subject:  "Verify your email address",
		TextBody: fmt.Sprintf("Please verify your email address by clicking this link: %s", verifyURL),
		HtmlBody: htmlBody.String(),
		Tag:        "email-verification",
		TrackOpens: true,
	}

	_, err := s.client.SendEmail(email)
	if err != nil {
		s.logger.Error("failed to send verification email via Postmark",
			slog.String("to", to),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.Info("verification email sent via Postmark",
		slog.String("to", to),
	)
	return nil
}

// SendPasswordResetEmail sends a password reset email via Postmark
func (s *postmarkEmailService) SendPasswordResetEmail(to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.VerifyBaseURL, token)

	// Render HTML body from template
	var htmlBody bytes.Buffer
	data := map[string]string{"ResetURL": resetURL}
	if err := resetPasswordTemplate.Execute(&htmlBody, data); err != nil {
		s.logger.Error("failed to render password reset email template",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to render email template: %w", err)
	}

	email := postmark.Email{
		From:     fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromAddress),
		To:       to,
		Subject:  "Reset your password",
		TextBody: fmt.Sprintf("Reset your password by clicking this link: %s", resetURL),
		HtmlBody: htmlBody.String(),
		Tag:        "password-reset",
		TrackOpens: true,
	}

	_, err := s.client.SendEmail(email)
	if err != nil {
		s.logger.Error("failed to send password reset email via Postmark",
			slog.String("to", to),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.Info("password reset email sent via Postmark",
		slog.String("to", to),
	)
	return nil
}
