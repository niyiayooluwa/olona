package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// GenerateRandomToken generates a 256-bit (32-byte) cryptographically secure random token
// and returns both the raw token (sent to the client) and its SHA-256 hash (stored in the database).
func GenerateRandomToken() (raw string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(bytes)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken deterministically computes the SHA-256 hash of a raw token.
// Used for indexed O(1) lookups in PostgreSQL (WHERE token_hash = $1).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// GenerateOTP generates a cryptographically random 6-digit numeric OTP (e.g. "482913").
func GenerateOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
