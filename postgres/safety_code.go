package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that SafetyCodeService implements aletheia.SafetyCodeService.
var _ aletheia.SafetyCodeService = (*SafetyCodeService)(nil)

// SafetyCodeService implements aletheia.SafetyCodeService using PostgreSQL.
type SafetyCodeService struct {
	db *DB
}

func (s *SafetyCodeService) FindSafetyCodeByID(ctx context.Context, id uuid.UUID) (*aletheia.SafetyCode, error) {
	safetyCode, err := s.db.queries.GetSafetyCode(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Safety code not found")
		}
		return nil, aletheia.Internal("Failed to fetch safety code", err)
	}
	return toDomainSafetyCode(safetyCode), nil
}

func (s *SafetyCodeService) FindSafetyCodeByCode(ctx context.Context, code string) (*aletheia.SafetyCode, error) {
	safetyCode, err := s.db.queries.GetSafetyCodeByCode(ctx, code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Safety code not found")
		}
		return nil, aletheia.Internal("Failed to fetch safety code", err)
	}
	return toDomainSafetyCode(safetyCode), nil
}

func (s *SafetyCodeService) FindSafetyCodes(ctx context.Context, filter aletheia.SafetyCodeFilter) ([]*aletheia.SafetyCode, int, error) {
	var codes []database.SafetyCode
	var err error

	// Choose query based on filter criteria
	if filter.Country != nil && filter.StateProvince != nil {
		codes, err = s.db.queries.ListSafetyCodesByLocation(ctx, database.ListSafetyCodesByLocationParams{
			Country:       toPgText(*filter.Country),
			StateProvince: toPgText(*filter.StateProvince),
		})
	} else if filter.Country != nil {
		codes, err = s.db.queries.ListSafetyCodesByCountry(ctx, toPgText(*filter.Country))
	} else if filter.StateProvince != nil {
		codes, err = s.db.queries.ListSafetyCodesByStateProvince(ctx, toPgText(*filter.StateProvince))
	} else {
		codes, err = s.db.queries.ListSafetyCodes(ctx)
	}

	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list safety codes", err)
	}

	// Apply offset/limit in memory
	total := len(codes)
	if filter.Offset > 0 && filter.Offset < len(codes) {
		codes = codes[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(codes) {
		codes = codes[:filter.Limit]
	}

	return toDomainSafetyCodes(codes), total, nil
}

func (s *SafetyCodeService) CreateSafetyCode(ctx context.Context, safetyCode *aletheia.SafetyCode) error {
	dbCode, err := s.db.queries.CreateSafetyCode(ctx, database.CreateSafetyCodeParams{
		Code:          safetyCode.Code,
		Description:   safetyCode.Description,
		Country:       toPgText(safetyCode.Country),
		StateProvince: toPgText(safetyCode.StateProvince),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return aletheia.Conflict("Safety code already exists")
		}
		return aletheia.Internal("Failed to create safety code", err)
	}

	// Update safety code with generated values
	safetyCode.ID = fromPgUUID(dbCode.ID)
	safetyCode.CreatedAt = fromPgTimestamp(dbCode.CreatedAt)
	safetyCode.UpdatedAt = fromPgTimestamp(dbCode.UpdatedAt)

	return nil
}

func (s *SafetyCodeService) UpdateSafetyCode(ctx context.Context, id uuid.UUID, upd aletheia.SafetyCodeUpdate) (*aletheia.SafetyCode, error) {
	params := database.UpdateSafetyCodeParams{
		ID: toPgUUID(id),
	}

	if upd.Code != nil {
		params.Code = *upd.Code
	}
	if upd.Description != nil {
		params.Description = *upd.Description
	}
	if upd.Country != nil {
		params.Country = toPgText(*upd.Country)
	}
	if upd.StateProvince != nil {
		params.StateProvince = toPgText(*upd.StateProvince)
	}

	code, err := s.db.queries.UpdateSafetyCode(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Safety code not found")
		}
		if isUniqueViolation(err) {
			return nil, aletheia.Conflict("Safety code already exists")
		}
		return nil, aletheia.Internal("Failed to update safety code", err)
	}

	return toDomainSafetyCode(code), nil
}

func (s *SafetyCodeService) DeleteSafetyCode(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeleteSafetyCode(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Safety code not found")
		}
		if isForeignKeyViolation(err) {
			return aletheia.Conflict("Safety code is referenced by violations")
		}
		return aletheia.Internal("Failed to delete safety code", err)
	}
	return nil
}

func (s *SafetyCodeService) GetAllSafetyCodes(ctx context.Context) ([]*aletheia.SafetyCode, error) {
	codes, err := s.db.queries.ListSafetyCodes(ctx)
	if err != nil {
		return nil, aletheia.Internal("Failed to list safety codes", err)
	}
	return toDomainSafetyCodes(codes), nil
}
