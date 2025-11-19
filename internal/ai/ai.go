package ai

import (
	"context"
	"log/slog"
)

// AIService defines the interface for AI vision processing services
type AIService interface {
	// AnalyzePhoto analyzes a photo for safety violations
	// Returns a list of detected violations with their descriptions, severity, and confidence
	AnalyzePhoto(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error)
}

// AnalysisRequest contains the data needed for photo analysis
type AnalysisRequest struct {
	// ImageData is the raw image bytes
	ImageData []byte
	// ImageURL is an alternative to ImageData - provide either ImageData or ImageURL
	ImageURL string
	// SafetyCodes is the list of safety codes to check against
	SafetyCodes []SafetyCodeContext
	// InspectionContext provides additional context about the inspection
	InspectionContext string
}

// SafetyCodeContext provides context about a safety code for the AI
type SafetyCodeContext struct {
	Code        string
	Description string
	Country     string
}

// AnalysisResponse contains the results of the AI analysis
type AnalysisResponse struct {
	// Violations is the list of detected safety violations
	Violations []DetectedViolation
	// AnalysisDetails contains additional information about the analysis
	AnalysisDetails string
	// TokensUsed tracks API usage (if applicable)
	TokensUsed int
}

// DetectedViolation represents a safety violation detected by the AI
type DetectedViolation struct {
	// SafetyCodeID is the ID of the safety code that was violated (if matched)
	SafetyCodeID string
	// SafetyCode is the code identifier (e.g., "OSHA 1926.501")
	SafetyCode string
	// Description is the AI's description of the violation
	Description string
	// Severity indicates how serious the violation is
	Severity ViolationSeverity
	// Confidence is the AI's confidence in this detection (0.0 - 1.0)
	Confidence float64
	// Location describes where in the image the violation was detected
	Location string
}

// ViolationSeverity represents the severity level of a violation
type ViolationSeverity string

const (
	SeverityCritical ViolationSeverity = "critical"
	SeverityHigh     ViolationSeverity = "high"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityLow      ViolationSeverity = "low"
)

// AIConfig holds configuration for AI services
type AIConfig struct {
	Provider string // "mock", "claude", "openai", etc.

	// Claude/Anthropic configuration
	ClaudeAPIKey string
	ClaudeModel  string // e.g., "claude-3-5-sonnet-20241022"

	// OpenAI configuration (for future use)
	OpenAIAPIKey string
	OpenAIModel  string

	// Common settings
	MaxTokens   int
	Temperature float64
}

// NewAIService creates an AI service based on the provider configuration
func NewAIService(logger *slog.Logger, config AIConfig) AIService {
	switch config.Provider {
	case "claude":
		return newClaudeService(logger, config)
	default:
		return newMockAIService(logger)
	}
}
