package aletheia

import "context"

// EmailService defines operations for sending emails.
type EmailService interface {
	// SendVerificationEmail sends an email verification link to a user.
	SendVerificationEmail(ctx context.Context, to, name, token string) error

	// SendPasswordResetEmail sends a password reset link to a user.
	SendPasswordResetEmail(ctx context.Context, to, name, token string) error

	// SendWelcomeEmail sends a welcome email to a new user.
	SendWelcomeEmail(ctx context.Context, to, name string) error

	// SendInspectionReport sends an inspection report to recipients.
	SendInspectionReport(ctx context.Context, to []string, subject string, reportURL string) error
}

// EmailConfig holds configuration for email services.
type EmailConfig struct {
	// Provider is the email provider ("mock" or "postmark").
	Provider string

	// FromAddress is the sender email address.
	FromAddress string

	// FromName is the sender display name.
	FromName string

	// VerifyBaseURL is the base URL for verification links.
	VerifyBaseURL string

	// ResetBaseURL is the base URL for password reset links.
	ResetBaseURL string

	// Postmark-specific configuration
	PostmarkServerToken string
}

// Email represents an email message.
type Email struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
}
