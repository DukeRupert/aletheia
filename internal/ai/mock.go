package ai

import (
	"context"
	"log/slog"
)

// mockAIService is a mock implementation for development and testing
type mockAIService struct {
	logger *slog.Logger
}

// newMockAIService creates a new mock AI service
func newMockAIService(logger *slog.Logger) *mockAIService {
	return &mockAIService{
		logger: logger,
	}
}

// AnalyzePhoto returns mock violations for testing
func (s *mockAIService) AnalyzePhoto(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	s.logger.Info("🤖 MOCK AI: Analyzing photo",
		slog.Int("safety_codes_provided", len(request.SafetyCodes)),
		slog.String("has_image_data", func() string {
			if len(request.ImageData) > 0 {
				return "yes"
			}
			return "no"
		}()),
		slog.String("image_url", request.ImageURL),
	)

	// Return mock violations for testing
	mockViolations := []DetectedViolation{
		{
			SafetyCode:  "OSHA 1926.501",
			Description: "Worker observed at elevated height without proper fall protection system. No guardrails, safety nets, or personal fall arrest system visible.",
			Severity:    SeverityCritical,
			Confidence:  0.92,
			Location:    "Upper left section of image, approximately 15 feet above ground level",
		},
		{
			SafetyCode:  "OSHA 1926.100",
			Description: "Worker not wearing required hard hat in construction zone where overhead hazards are present.",
			Severity:    SeverityHigh,
			Confidence:  0.87,
			Location:    "Center of image, worker near scaffolding",
		},
	}

	s.logger.Info("🤖 MOCK AI: Analysis complete",
		slog.Int("violations_detected", len(mockViolations)),
	)

	return &AnalysisResponse{
		Violations:      mockViolations,
		AnalysisDetails: "Mock AI analysis - this is simulated data for testing purposes",
		TokensUsed:      0,
	}, nil
}
