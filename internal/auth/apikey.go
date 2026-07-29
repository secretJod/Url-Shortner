// Package auth handles API key generation and hashing.
//
// Design: the raw key is shown to the user exactly once, at creation time.
// Only a SHA-256 hash of it is ever persisted, so a database leak alone
// can't be used to impersonate a user (same principle as password hashing,
// just without the need for bcrypt's slow-by-design cost — API keys are
// high-entropy random values, not human-chosen passwords, so a fast hash
// is fine and keeps auth-check latency low on every request).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// keyPrefix makes keys recognizable (e.g. in logs, accidental commits,
// secret-scanning tools) the way "sk_" prefixes work for Stripe etc.
const keyPrefix = "usk_" // "url-shortener key"

const rawKeyBytes = 32 // 256 bits of entropy

// GenerateAPIKey creates a new random API key. Returns the raw key (show
// this to the user ONCE) and its hash (store ONLY this).
func GenerateAPIKey() (rawKey string, hash string, err error) {
	buf := make([]byte, rawKeyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawKey = keyPrefix + hex.EncodeToString(buf)
	hash = HashKey(rawKey)
	return rawKey, hash, nil
}

// HashKey deterministically hashes a raw API key for lookup/storage.
func HashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

var ErrMalformedKey = errors.New("auth: malformed API key")

// ExtractBearerToken pulls the raw key out of an `Authorization: Bearer <key>` header.
func ExtractBearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", ErrMalformedKey
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", ErrMalformedKey
	}
	return token, nil
}
