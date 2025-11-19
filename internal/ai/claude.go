package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// claudeService implements AIService using Claude (Anthropic)
type claudeService struct {
	client      *anthropic.Client
	logger      *slog.Logger
	model       string
	maxTokens   int
	temperature float64
}

// newClaudeService creates a new Claude AI service
func newClaudeService(logger *slog.Logger, config AIConfig) *claudeService {
	client := anthropic.NewClient(
		option.WithAPIKey(config.ClaudeAPIKey),
	)

	return &claudeService{
		client:      &client,
		logger:      logger,
		model:       config.ClaudeModel,
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
	}
}

// AnalyzePhoto analyzes a photo for safety violations using Claude's vision API
func (s *claudeService) AnalyzePhoto(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	s.logger.Info("analyzing photo with Claude",
		slog.String("model", s.model),
		slog.Int("safety_codes_count", len(request.SafetyCodes)))

	// Build the system prompt with safety code context
	systemPrompt := s.buildSystemPrompt(request.SafetyCodes)

	// Build the user prompt
	userPrompt := s.buildUserPrompt(request.InspectionContext)

	// Prepare image content
	if len(request.ImageData) == 0 && request.ImageURL == "" {
		return nil, fmt.Errorf("either ImageData or ImageURL must be provided")
	}

	if request.ImageURL != "" {
		// Use image URL (not directly supported by Claude SDK, would need to download first)
		return nil, fmt.Errorf("image URL not yet supported - please provide ImageData")
	}

	// Use image data - encode as base64
	base64Image := base64.StdEncoding.EncodeToString(request.ImageData)

	// Determine media type (assume JPEG for now, could be enhanced)
	mediaType := "image/jpeg"

	// Create the message with vision
	message, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(s.model),
		MaxTokens: int64(s.maxTokens),
		System: []anthropic.TextBlockParam{
			{
				Text: systemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userPrompt),
				anthropic.NewImageBlockBase64(mediaType, base64Image),
			),
		},
		Temperature: anthropic.Float(s.temperature),
	})

	if err != nil {
		s.logger.Error("failed to analyze photo with Claude",
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to analyze photo: %w", err)
	}

	// Extract text response
	var responseText string
	for _, content := range message.Content {
		if content.Type == "text" {
			responseText += content.Text
		}
	}

	s.logger.Info("Claude analysis complete",
		slog.Int("input_tokens", int(message.Usage.InputTokens)),
		slog.Int("output_tokens", int(message.Usage.OutputTokens)))

	// Parse the response into violations
	violations, analysisDetails := s.parseClaudeResponse(responseText, request.SafetyCodes)

	return &AnalysisResponse{
		Violations:      violations,
		AnalysisDetails: analysisDetails,
		TokensUsed:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
	}, nil
}

// buildSystemPrompt creates the system prompt with safety code context
func (s *claudeService) buildSystemPrompt(safetyCodes []SafetyCodeContext) string {
	var sb strings.Builder

	sb.WriteString("You are an expert construction safety inspector AI. Your task is to analyze construction site photos and identify potential safety violations.\n\n")
	sb.WriteString("You have deep knowledge of construction safety standards including OSHA regulations and can identify hazards such as:\n")
	sb.WriteString("- Fall protection issues (missing guardrails, improper harness use, etc.)\n")
	sb.WriteString("- Personal protective equipment violations (missing hard hats, safety glasses, etc.)\n")
	sb.WriteString("- Scaffolding and ladder safety issues\n")
	sb.WriteString("- Electrical hazards\n")
	sb.WriteString("- Excavation and trench hazards\n")
	sb.WriteString("- Equipment safety issues\n")
	sb.WriteString("- Housekeeping and general site safety\n\n")

	if len(safetyCodes) > 0 {
		sb.WriteString("Focus on violations related to these safety codes:\n\n")
		for _, code := range safetyCodes {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", code.Code, code.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("For each violation you identify, provide:\n")
	sb.WriteString("1. The relevant safety code (if applicable)\n")
	sb.WriteString("2. A clear description of what violation you observed\n")
	sb.WriteString("3. Severity level: critical, high, medium, or low\n")
	sb.WriteString("4. Your confidence level (0.0 to 1.0)\n")
	sb.WriteString("5. Location in the image where the violation appears\n\n")
	sb.WriteString("Respond ONLY with a JSON array of violations. Each violation should be a JSON object with these fields:\n")
	sb.WriteString(`{"safety_code": "OSHA X", "description": "...", "severity": "high", "confidence": 0.85, "location": "..."}`)
	sb.WriteString("\n\nIf no violations are found, return an empty array [].")

	return sb.String()
}

// buildUserPrompt creates the user prompt with inspection context
func (s *claudeService) buildUserPrompt(inspectionContext string) string {
	prompt := "Please analyze this construction site photo for safety violations."

	if inspectionContext != "" {
		prompt += fmt.Sprintf("\n\nInspection Context: %s", inspectionContext)
	}

	prompt += "\n\nRespond with a JSON array of violations as specified in the system instructions."

	return prompt
}

// parseClaudeResponse parses Claude's response into DetectedViolation structs
func (s *claudeService) parseClaudeResponse(response string, safetyCodes []SafetyCodeContext) ([]DetectedViolation, string) {
	// Try to extract JSON array from the response
	// Claude might wrap it in markdown code blocks
	jsonStr := s.extractJSON(response)

	var rawViolations []struct {
		SafetyCode  string  `json:"safety_code"`
		Description string  `json:"description"`
		Severity    string  `json:"severity"`
		Confidence  float64 `json:"confidence"`
		Location    string  `json:"location"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawViolations); err != nil {
		s.logger.Error("failed to parse Claude response as JSON",
			slog.String("error", err.Error()),
			slog.String("response", response))
		return []DetectedViolation{}, response
	}

	// Convert to DetectedViolation structs
	violations := make([]DetectedViolation, 0, len(rawViolations))
	for _, raw := range rawViolations {
		violation := DetectedViolation{
			SafetyCode:  raw.SafetyCode,
			Description: raw.Description,
			Confidence:  raw.Confidence,
			Location:    raw.Location,
		}

		// Map severity string to enum
		switch strings.ToLower(raw.Severity) {
		case "critical":
			violation.Severity = SeverityCritical
		case "high":
			violation.Severity = SeverityHigh
		case "medium":
			violation.Severity = SeverityMedium
		case "low":
			violation.Severity = SeverityLow
		default:
			violation.Severity = SeverityMedium
		}

		// Try to match safety code to provided safety code IDs
		for _, sc := range safetyCodes {
			if strings.Contains(strings.ToUpper(raw.SafetyCode), strings.ToUpper(sc.Code)) {
				violation.SafetyCodeID = sc.Code
				break
			}
		}

		violations = append(violations, violation)
	}

	return violations, fmt.Sprintf("Claude identified %d potential violations", len(violations))
}

// extractJSON attempts to extract JSON from a response that might be wrapped in markdown
func (s *claudeService) extractJSON(response string) string {
	// Remove markdown code blocks if present
	response = strings.TrimSpace(response)

	// Check for ```json wrapper
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response
}
