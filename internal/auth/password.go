package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyPassword compares a password with a hash to check if they match
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePassword validates password strength and complexity.
//
// Requirements:
// - Minimum 8 characters
// - Maximum 128 characters (to prevent DoS via bcrypt)
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
//
// Returns nil if valid, error with specific message if invalid.
func ValidatePassword(password string) error {
	// Check minimum length
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Check maximum length (bcrypt has a 72-byte limit, but we set lower for usability)
	if len(password) > 128 {
		return errors.New("password must not exceed 128 characters")
	}

	// Check for required character types
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
	)

	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	// Validate all requirements are met
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	return nil
}
