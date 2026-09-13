package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const tokenPrefix = "pgw_"
const registrationTokenPrefix = "pgwr_"

// generateToken returns a new opaque bearer token. Only its hash is stored.
func generateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// generateRegistrationToken returns a new opaque bearer token. Only its hash is stored.
func generateRegistrationToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return registrationTokenPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// generatePassword returns a random temporary password for admin-created users.
func generatePassword() string {
	return rand.Text()
}
