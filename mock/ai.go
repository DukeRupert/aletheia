package mock

import (
	"context"

	"github.com/dukerupert/aletheia"
)

// Compile-time interface check
var _ aletheia.AIService = (*AIService)(nil)

// AIService is a mock implementation of aletheia.AIService.
type AIService struct {
	AnalyzePhotoFn func(ctx context.Context, photoURL string, safetyCodes []*aletheia.SafetyCode) (*aletheia.AnalysisResult, error)
}

func (s *AIService) AnalyzePhoto(ctx context.Context, photoURL string, safetyCodes []*aletheia.SafetyCode) (*aletheia.AnalysisResult, error) {
	if s.AnalyzePhotoFn != nil {
		return s.AnalyzePhotoFn(ctx, photoURL, safetyCodes)
	}
	// Return empty result by default
	return &aletheia.AnalysisResult{
		Violations: []aletheia.DetectedViolation{},
		Summary:    "No violations detected (mock)",
	}, nil
}
