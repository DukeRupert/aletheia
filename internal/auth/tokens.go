package auth

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	// VerificationTokenLength is the length of verification tokens in bytes
	VerificationTokenLength = 32
)

// GenerateVerificationToken generates a cryptographically secure random token
// for email verification or password reset
func GenerateVerificationToken() (string, error) {
	b := make([]byte, VerificationTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
