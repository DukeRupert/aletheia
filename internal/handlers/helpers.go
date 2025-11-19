package handlers

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// parseUUID converts a string UUID to pgtype.UUID
func parseUUID(s string) (pgtype.UUID, error) {
	var pguuid pgtype.UUID
	err := pguuid.Scan(s)
	return pguuid, err
}

// uuidToPgUUID converts a uuid.UUID to pgtype.UUID
func uuidToPgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}
