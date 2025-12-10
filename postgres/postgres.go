// Package postgres provides PostgreSQL implementations of domain service interfaces.
package postgres

import (
	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the database connection pool and exposes domain services.
type DB struct {
	pool    *pgxpool.Pool
	queries *database.Queries

	// Domain services (initialized in NewDB)
	UserService         aletheia.UserService
	OrganizationService aletheia.OrganizationService
	ProjectService      aletheia.ProjectService
	InspectionService   aletheia.InspectionService
	PhotoService        aletheia.PhotoService
	ViolationService    aletheia.ViolationService
	SafetyCodeService   aletheia.SafetyCodeService
	SessionService      aletheia.SessionService
}

// NewDB creates a new database wrapper with all services initialized.
func NewDB(pool *pgxpool.Pool) *DB {
	queries := database.New(pool)
	db := &DB{
		pool:    pool,
		queries: queries,
	}

	// Initialize services with reference back to DB
	db.UserService = &UserService{db: db}
	db.OrganizationService = &OrganizationService{db: db}
	db.ProjectService = &ProjectService{db: db}
	db.InspectionService = &InspectionService{db: db}
	db.PhotoService = &PhotoService{db: db}
	db.ViolationService = &ViolationService{db: db}
	db.SafetyCodeService = &SafetyCodeService{db: db}
	db.SessionService = &SessionService{db: db}

	return db
}

// Pool returns the underlying connection pool.
// Use sparingly - prefer using service methods.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Queries returns the sqlc queries object.
// Use sparingly - prefer using service methods.
func (db *DB) Queries() *database.Queries {
	return db.queries
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.pool.Close()
}
