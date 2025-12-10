package aletheia

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey int

const (
	userContextKey contextKey = iota + 1
	sessionContextKey
	organizationContextKey
	requestIDContextKey
)

// User context helpers

// NewContextWithUser attaches a user to the context.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the authenticated user from the context, or nil.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// UserIDFromContext returns the authenticated user's ID, or a zero UUID.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	if user := UserFromContext(ctx); user != nil {
		return user.ID
	}
	return uuid.UUID{}
}

// MustUserFromContext returns the user from context or panics.
// Use only in code paths where authentication is guaranteed.
func MustUserFromContext(ctx context.Context) *User {
	user := UserFromContext(ctx)
	if user == nil {
		panic("user required in context but not found")
	}
	return user
}

// Session context helpers

// NewContextWithSession attaches a session to the context.
func NewContextWithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// SessionFromContext returns the current session from the context, or nil.
func SessionFromContext(ctx context.Context) *Session {
	session, _ := ctx.Value(sessionContextKey).(*Session)
	return session
}

// Organization context helpers

// NewContextWithOrganization attaches an organization to the context.
func NewContextWithOrganization(ctx context.Context, org *Organization) context.Context {
	return context.WithValue(ctx, organizationContextKey, org)
}

// OrganizationFromContext returns the current organization from the context, or nil.
func OrganizationFromContext(ctx context.Context) *Organization {
	org, _ := ctx.Value(organizationContextKey).(*Organization)
	return org
}

// OrganizationIDFromContext returns the current organization's ID, or a zero UUID.
func OrganizationIDFromContext(ctx context.Context) uuid.UUID {
	if org := OrganizationFromContext(ctx); org != nil {
		return org.ID
	}
	return uuid.UUID{}
}

// Request ID context helpers

// NewContextWithRequestID attaches a request ID to the context.
func NewContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext returns the request ID from the context, or empty string.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

// Convenience helpers

// IsAuthenticated returns true if a user is present in the context.
func IsAuthenticated(ctx context.Context) bool {
	return UserFromContext(ctx) != nil
}

// HasOrganization returns true if an organization is present in the context.
func HasOrganization(ctx context.Context) bool {
	return OrganizationFromContext(ctx) != nil
}
