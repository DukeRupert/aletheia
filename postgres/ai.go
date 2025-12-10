package postgres

import (
	"context"
	"log/slog"

	"github.com/dukerupert/aletheia"
)

// Compile-time interface check
var _ aletheia.AIService = (*MockAIService)(nil)

// NewAIService creates an AI service based on the provider configuration.
func NewAIService(logger *slog.Logger, cfg aletheia.AIConfig) aletheia.AIService {
	switch cfg.Provider {
	case "claude":
		// TODO: Implement Claude AI service
		logger.Warn("claude AI service not yet implemented, falling back to mock")
		return &MockAIService{logger: logger}
	default:
		return &MockAIService{logger: logger}
	}
}

// MockAIService is a mock implementation that returns empty results.
type MockAIService struct {
	logger *slog.Logger
}

// AnalyzePhoto returns a mock analysis result.
func (s *MockAIService) AnalyzePhoto(ctx context.Context, photoURL string, safetyCodes []*aletheia.SafetyCode) (*aletheia.AnalysisResult, error) {
	s.logger.Info("MOCK AI: Analyzing photo",
		slog.String("photo_url", photoURL),
		slog.Int("safety_codes_count", len(safetyCodes)))

	return &aletheia.AnalysisResult{
		Violations: []aletheia.DetectedViolation{},
		Summary:    "No violations detected (mock analysis)",
	}, nil
}
