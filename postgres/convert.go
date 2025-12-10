package postgres

import (
	"math/big"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUID conversions

// toPgUUID converts a google/uuid.UUID to pgtype.UUID.
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

// fromPgUUID converts a pgtype.UUID to google/uuid.UUID.
func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.UUID{}
	}
	return uuid.UUID(id.Bytes)
}

// Text conversions

// toPgText converts a string to pgtype.Text.
func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// toPgTextPtr converts a string pointer to pgtype.Text.
func toPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// fromPgText converts a pgtype.Text to string.
func fromPgText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// fromPgTextPtr converts a pgtype.Text to string pointer (nil if not valid).
func fromPgTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// Timestamp conversions

// toPgTimestamp converts a time.Time to pgtype.Timestamptz.
func toPgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// toPgTimestampPtr converts a time.Time pointer to pgtype.Timestamptz.
func toPgTimestampPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// fromPgTimestamp converts a pgtype.Timestamptz to time.Time.
func fromPgTimestamp(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// fromPgTimestampPtr converts a pgtype.Timestamptz to time.Time pointer (nil if not valid).
func fromPgTimestampPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// Numeric conversions

// toPgNumeric converts a float64 to pgtype.Numeric.
func toPgNumeric(f float64) pgtype.Numeric {
	if f == 0 {
		return pgtype.Numeric{Valid: false}
	}
	// Convert float to big.Int by multiplying by scale
	scale := int32(6) // 6 decimal places
	scaleFactor := new(big.Float).SetFloat64(1000000)
	bf := new(big.Float).SetFloat64(f)
	bf.Mul(bf, scaleFactor)
	bi, _ := bf.Int(nil)
	return pgtype.Numeric{
		Int:   bi,
		Exp:   -scale,
		Valid: true,
	}
}

// Domain type conversions

// User conversions

func toDomainUser(u database.User) *aletheia.User {
	return &aletheia.User{
		ID:           fromPgUUID(u.ID),
		Email:        u.Email,
		Username:     u.Username,
		FirstName:    fromPgText(u.FirstName),
		LastName:     fromPgText(u.LastName),
		Status:       aletheia.UserStatus(u.Status),
		StatusReason: fromPgText(u.StatusReason),
		CreatedAt:    fromPgTimestamp(u.CreatedAt),
		UpdatedAt:    fromPgTimestamp(u.UpdatedAt),
		LastLoginAt:  fromPgTimestampPtr(u.LastLoginAt),
		VerifiedAt:   fromPgTimestampPtr(u.VerifiedAt),
	}
}

func toDomainUsers(users []database.User) []*aletheia.User {
	result := make([]*aletheia.User, len(users))
	for i, u := range users {
		result[i] = toDomainUser(u)
	}
	return result
}

// Organization conversions

func toDomainOrganization(o database.Organization) *aletheia.Organization {
	return &aletheia.Organization{
		ID:        fromPgUUID(o.ID),
		Name:      o.Name,
		CreatedAt: fromPgTimestamp(o.CreatedAt),
		UpdatedAt: fromPgTimestamp(o.UpdatedAt),
	}
}

func toDomainOrganizations(orgs []database.Organization) []*aletheia.Organization {
	result := make([]*aletheia.Organization, len(orgs))
	for i, o := range orgs {
		result[i] = toDomainOrganization(o)
	}
	return result
}

func toDomainOrganizationMember(m database.OrganizationMember) *aletheia.OrganizationMember {
	return &aletheia.OrganizationMember{
		ID:             fromPgUUID(m.ID),
		OrganizationID: fromPgUUID(m.OrganizationID),
		UserID:         fromPgUUID(m.UserID),
		Role:           aletheia.OrganizationRole(m.Role),
		CreatedAt:      fromPgTimestamp(m.CreatedAt),
	}
}

func toDomainOrganizationMembers(members []database.OrganizationMember) []*aletheia.OrganizationMember {
	result := make([]*aletheia.OrganizationMember, len(members))
	for i, m := range members {
		result[i] = toDomainOrganizationMember(m)
	}
	return result
}

// Project conversions

func toDomainProject(p database.Project) *aletheia.Project {
	return &aletheia.Project{
		ID:             fromPgUUID(p.ID),
		OrganizationID: fromPgUUID(p.OrganizationID),
		Name:           p.Name,
		Description:    fromPgText(p.Description),
		ProjectType:    fromPgText(p.ProjectType),
		Status:         fromPgText(p.Status),
		Address:        fromPgText(p.Address),
		City:           fromPgText(p.City),
		State:          fromPgText(p.State),
		ZipCode:        fromPgText(p.ZipCode),
		Country:        fromPgText(p.Country),
		CreatedAt:      fromPgTimestamp(p.CreatedAt),
		UpdatedAt:      fromPgTimestamp(p.UpdatedAt),
	}
}

func toDomainProjects(projects []database.Project) []*aletheia.Project {
	result := make([]*aletheia.Project, len(projects))
	for i, p := range projects {
		result[i] = toDomainProject(p)
	}
	return result
}

// Inspection conversions

func toDomainInspection(i database.Inspection) *aletheia.Inspection {
	return &aletheia.Inspection{
		ID:          fromPgUUID(i.ID),
		ProjectID:   fromPgUUID(i.ProjectID),
		InspectorID: fromPgUUID(i.InspectorID),
		Status:      aletheia.InspectionStatus(i.Status),
		CreatedAt:   fromPgTimestamp(i.CreatedAt),
		UpdatedAt:   fromPgTimestamp(i.UpdatedAt),
	}
}

func toDomainInspections(inspections []database.Inspection) []*aletheia.Inspection {
	result := make([]*aletheia.Inspection, len(inspections))
	for i, insp := range inspections {
		result[i] = toDomainInspection(insp)
	}
	return result
}

// Photo conversions

func toDomainPhoto(p database.Photo) *aletheia.Photo {
	return &aletheia.Photo{
		ID:           fromPgUUID(p.ID),
		InspectionID: fromPgUUID(p.InspectionID),
		StorageURL:   p.StorageUrl,
		ThumbnailURL: fromPgText(p.ThumbnailUrl),
		CreatedAt:    fromPgTimestamp(p.CreatedAt),
	}
}

func toDomainPhotos(photos []database.Photo) []*aletheia.Photo {
	result := make([]*aletheia.Photo, len(photos))
	for i, p := range photos {
		result[i] = toDomainPhoto(p)
	}
	return result
}

// Violation conversions

func toDomainViolation(v database.DetectedViolation) *aletheia.Violation {
	var confidence float64
	if v.ConfidenceScore.Valid {
		// Convert pgtype.Numeric to float64
		f, _ := v.ConfidenceScore.Float64Value()
		confidence = f.Float64
	}

	return &aletheia.Violation{
		ID:              fromPgUUID(v.ID),
		PhotoID:         fromPgUUID(v.PhotoID),
		SafetyCodeID:    fromPgUUID(v.SafetyCodeID),
		Description:     v.Description,
		Severity:        aletheia.Severity(v.Severity),
		Status:          aletheia.ViolationStatus(v.Status),
		ConfidenceScore: confidence,
		Location:        fromPgText(v.Location),
		CreatedAt:       fromPgTimestamp(v.CreatedAt),
	}
}

func toDomainViolations(violations []database.DetectedViolation) []*aletheia.Violation {
	result := make([]*aletheia.Violation, len(violations))
	for i, v := range violations {
		result[i] = toDomainViolation(v)
	}
	return result
}

// SafetyCode conversions

func toDomainSafetyCode(s database.SafetyCode) *aletheia.SafetyCode {
	return &aletheia.SafetyCode{
		ID:            fromPgUUID(s.ID),
		Code:          s.Code,
		Description:   s.Description,
		Country:       fromPgText(s.Country),
		StateProvince: fromPgText(s.StateProvince),
		CreatedAt:     fromPgTimestamp(s.CreatedAt),
		UpdatedAt:     fromPgTimestamp(s.UpdatedAt),
	}
}

func toDomainSafetyCodes(codes []database.SafetyCode) []*aletheia.SafetyCode {
	result := make([]*aletheia.SafetyCode, len(codes))
	for i, s := range codes {
		result[i] = toDomainSafetyCode(s)
	}
	return result
}

// Session conversions

func toDomainSession(s database.Session) *aletheia.Session {
	return &aletheia.Session{
		ID:        int(s.ID),
		UserID:    fromPgUUID(s.UserID),
		Token:     s.Token,
		ExpiresAt: fromPgTimestamp(s.ExpiresAt),
		CreatedAt: fromPgTimestamp(s.CreatedAt),
	}
}

func toDomainSessions(sessions []database.Session) []*aletheia.Session {
	result := make([]*aletheia.Session, len(sessions))
	for i, s := range sessions {
		result[i] = toDomainSession(s)
	}
	return result
}
