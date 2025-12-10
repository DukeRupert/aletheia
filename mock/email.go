package mock

import (
	"context"

	"github.com/dukerupert/aletheia"
)

// Compile-time interface check
var _ aletheia.EmailService = (*EmailService)(nil)

// EmailService is a mock implementation of aletheia.EmailService.
type EmailService struct {
	SendVerificationEmailFn  func(ctx context.Context, to, name, token string) error
	SendPasswordResetEmailFn func(ctx context.Context, to, name, token string) error
	SendWelcomeEmailFn       func(ctx context.Context, to, name string) error
	SendInspectionReportFn   func(ctx context.Context, to []string, subject string, reportURL string) error

	// Tracking sent emails for assertions
	SentEmails []SentEmail
}

// SentEmail records details of a sent email for testing assertions.
type SentEmail struct {
	Type      string
	To        string
	ToList    []string
	Name      string
	Token     string
	Subject   string
	ReportURL string
}

func (s *EmailService) SendVerificationEmail(ctx context.Context, to, name, token string) error {
	s.SentEmails = append(s.SentEmails, SentEmail{
		Type:  "verification",
		To:    to,
		Name:  name,
		Token: token,
	})
	if s.SendVerificationEmailFn != nil {
		return s.SendVerificationEmailFn(ctx, to, name, token)
	}
	return nil
}

func (s *EmailService) SendPasswordResetEmail(ctx context.Context, to, name, token string) error {
	s.SentEmails = append(s.SentEmails, SentEmail{
		Type:  "password_reset",
		To:    to,
		Name:  name,
		Token: token,
	})
	if s.SendPasswordResetEmailFn != nil {
		return s.SendPasswordResetEmailFn(ctx, to, name, token)
	}
	return nil
}

func (s *EmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	s.SentEmails = append(s.SentEmails, SentEmail{
		Type: "welcome",
		To:   to,
		Name: name,
	})
	if s.SendWelcomeEmailFn != nil {
		return s.SendWelcomeEmailFn(ctx, to, name)
	}
	return nil
}

func (s *EmailService) SendInspectionReport(ctx context.Context, to []string, subject string, reportURL string) error {
	s.SentEmails = append(s.SentEmails, SentEmail{
		Type:      "inspection_report",
		ToList:    to,
		Subject:   subject,
		ReportURL: reportURL,
	})
	if s.SendInspectionReportFn != nil {
		return s.SendInspectionReportFn(ctx, to, subject, reportURL)
	}
	return nil
}

// Reset clears all sent emails.
func (s *EmailService) Reset() {
	s.SentEmails = nil
}

// LastEmail returns the last sent email, or nil if none.
func (s *EmailService) LastEmail() *SentEmail {
	if len(s.SentEmails) == 0 {
		return nil
	}
	return &s.SentEmails[len(s.SentEmails)-1]
}

// EmailsSentTo returns all emails sent to the given address.
func (s *EmailService) EmailsSentTo(to string) []SentEmail {
	var result []SentEmail
	for _, email := range s.SentEmails {
		if email.To == to {
			result = append(result, email)
		}
	}
	return result
}
