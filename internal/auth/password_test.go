package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	if hash == password {
		t.Fatal("HashPassword returned plaintext password")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("HashPassword should fail for empty password")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Test correct password
	err = VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed for correct password: %v", err)
	}

	// Test incorrect password
	err = VerifyPassword("wrongPassword", hash)
	if err == nil {
		t.Fatal("VerifyPassword should fail for incorrect password")
	}
}

func TestVerifyPasswordDifferentHashes(t *testing.T) {
	password := "mySecurePassword123"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Hashes should be different (bcrypt uses random salt)
	if hash1 == hash2 {
		t.Fatal("Multiple hashes of same password should differ")
	}

	// But both should verify correctly
	if err := VerifyPassword(password, hash1); err != nil {
		t.Fatal("First hash should verify")
	}

	if err := VerifyPassword(password, hash2); err != nil {
		t.Fatal("Second hash should verify")
	}
}
