package aletheia

import (
	"context"

	"github.com/google/uuid"
)

// AIService defines operations for AI-powered analysis.
type AIService interface {
	// AnalyzePhoto analyzes a photo for safety violations.
	// The photoURL should be a publicly accessible URL.
	// SafetyCodes are the codes to check against.
	AnalyzePhoto(ctx context.Context, photoURL string, safetyCodes []*SafetyCode) (*AnalysisResult, error)
}

// AnalysisResult contains the results of an AI photo analysis.
type AnalysisResult struct {
	// Violations are the detected violations.
	Violations []DetectedViolation `json:"violations"`

	// Summary is a human-readable summary of the analysis.
	Summary string `json:"summary"`

	// AnalysisTime is how long the analysis took in milliseconds.
	AnalysisTimeMs int64 `json:"analysisTimeMs,omitempty"`
}

// DetectedViolation represents a violation detected by AI analysis.
type DetectedViolation struct {
	// SafetyCodeID is the ID of the matched safety code.
	SafetyCodeID uuid.UUID `json:"safetyCodeId"`

	// SafetyCode is the matched safety code (for reference).
	SafetyCode string `json:"safetyCode,omitempty"`

	// Description describes the specific violation found.
	Description string `json:"description"`

	// Severity is the severity level of the violation.
	Severity Severity `json:"severity"`

	// Confidence is the confidence score (0.0 to 1.0).
	Confidence float64 `json:"confidence"`

	// Location describes where in the photo the violation was detected.
	Location string `json:"location,omitempty"`

	// BoundingBox defines the region of interest in the image.
	BoundingBox *BoundingBox `json:"boundingBox,omitempty"`
}

// BoundingBox defines a rectangular region in an image.
// Coordinates are normalized (0.0 to 1.0) relative to image dimensions.
type BoundingBox struct {
	X      float64 `json:"x"`      // Left edge (0.0 to 1.0)
	Y      float64 `json:"y"`      // Top edge (0.0 to 1.0)
	Width  float64 `json:"width"`  // Width (0.0 to 1.0)
	Height float64 `json:"height"` // Height (0.0 to 1.0)
}

// AIConfig holds configuration for AI services.
type AIConfig struct {
	// Provider is the AI provider ("mock" or "claude").
	Provider string

	// Claude-specific configuration
	ClaudeAPIKey string
	ClaudeModel  string

	// ConfidenceThreshold is the minimum confidence for reporting violations.
	ConfidenceThreshold float64
}

// DefaultAIConfig returns the default AI configuration.
func DefaultAIConfig() AIConfig {
	return AIConfig{
		Provider:            "mock",
		ClaudeModel:         "claude-sonnet-4-20250514",
		ConfidenceThreshold: 0.7,
	}
}
