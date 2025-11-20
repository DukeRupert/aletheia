package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/dukerupert/aletheia/internal/ai"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// PhotoAnalysisJobHandler handles photo analysis jobs from the queue
type PhotoAnalysisJobHandler struct {
	db      *database.Queries
	ai      ai.AIService
	storage storage.FileStorage
	logger  *slog.Logger
}

// NewPhotoAnalysisJobHandler creates a new photo analysis job handler
func NewPhotoAnalysisJobHandler(db *database.Queries, aiService ai.AIService, storageService storage.FileStorage, logger *slog.Logger) *PhotoAnalysisJobHandler {
	return &PhotoAnalysisJobHandler{
		db:      db,
		ai:      aiService,
		storage: storageService,
		logger:  logger,
	}
}

// Handle processes a photo analysis job
func (h *PhotoAnalysisJobHandler) Handle(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	// Extract photo ID from payload
	photoIDStr, ok := job.Payload["photo_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid photo_id in job payload")
	}

	photoID, err := uuid.Parse(photoIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid photo_id format: %w", err)
	}

	h.logger.Info("starting photo analysis",
		slog.String("photo_id", photoID.String()),
		slog.String("job_id", job.ID.String()),
	)

	// Fetch photo from database
	photo, err := h.db.GetPhoto(ctx, pgtype.UUID{Bytes: photoID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch photo: %w", err)
	}

	// Fetch safety codes for analysis context
	safetyCodes, err := h.db.ListSafetyCodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch safety codes: %w", err)
	}

	// Convert safety codes to AI context format
	codeContexts := make([]ai.SafetyCodeContext, 0, len(safetyCodes))
	for _, code := range safetyCodes {
		codeContexts = append(codeContexts, ai.SafetyCodeContext{
			Code:        code.Code,
			Description: code.Description,
			Country:     code.Country.String,
		})
	}

	// Prepare AI analysis request
	// Use the photo URL for analysis (AI service will fetch it)
	analysisReq := ai.AnalysisRequest{
		ImageURL:          photo.StorageUrl,
		SafetyCodes:       codeContexts,
		InspectionContext: fmt.Sprintf("Construction site inspection at project location"),
	}

	// Perform AI analysis
	h.logger.Debug("calling AI service for photo analysis",
		slog.String("photo_id", photoID.String()),
	)

	analysisResp, err := h.ai.AnalyzePhoto(ctx, analysisReq)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Store detected violations in database
	violationsCreated := 0
	for _, violation := range analysisResp.Violations {
		// Find matching safety code by code string
		var safetyCodeID pgtype.UUID
		if violation.SafetyCode != "" {
			safetyCode, err := h.db.GetSafetyCodeByCode(ctx, violation.SafetyCode)
			if err == nil {
				safetyCodeID = safetyCode.ID
			}
		}

		// Prepare location field
		locationText := pgtype.Text{
			String: violation.Location,
			Valid:  violation.Location != "",
		}

		// Create detected violation record
		confidenceInt := new(big.Int).SetInt64(int64(violation.Confidence * 10000))
		_, err := h.db.CreateDetectedViolation(ctx, database.CreateDetectedViolationParams{
			PhotoID:         photo.ID,
			Description:     violation.Description,
			ConfidenceScore: pgtype.Numeric{Int: confidenceInt, Exp: -4, Valid: true},
			SafetyCodeID:    safetyCodeID,
			Status:          database.ViolationStatusPending,
			Severity:        database.ViolationSeverity(violation.Severity),
			Location:        locationText,
		})

		if err != nil {
			h.logger.Error("failed to create detected violation",
				slog.String("photo_id", photoID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		violationsCreated++
	}

	h.logger.Info("photo analysis completed",
		slog.String("photo_id", photoID.String()),
		slog.Int("violations_detected", len(analysisResp.Violations)),
		slog.Int("violations_stored", violationsCreated),
		slog.Int("tokens_used", analysisResp.TokensUsed),
	)

	// Return job result
	return map[string]interface{}{
		"photo_id":            photoID.String(),
		"violations_detected": len(analysisResp.Violations),
		"violations_stored":   violationsCreated,
		"tokens_used":         analysisResp.TokensUsed,
		"analysis_details":    analysisResp.AnalysisDetails,
	}, nil
}
