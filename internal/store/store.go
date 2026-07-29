// Package store defines the persistence interface for Links. Handlers
// depend on this interface, not on Prisma directly, so the storage
// backend can be swapped (or mocked in tests) without touching handler code.
package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("store: link not found")
	ErrAliasTaken   = errors.New("store: short code / alias already taken")
	ErrInvalidAlias = errors.New("store: custom alias format invalid")
)

// Link mirrors the Prisma Link model, decoupled from generated Prisma types
// so the rest of the app never imports generated code directly.
type Link struct {
	ID           uint64
	ShortCode    string
	LongURL      string
	UserID       *uint64
	CustomAlias  bool
	ExpiresAt    *time.Time
	PasswordHash *string
	CreatedAt    time.Time
}

// User mirrors the Prisma User model.
type User struct {
	ID        uint64
	Email     string
	CreatedAt time.Time
}

// ApiKey mirrors the Prisma ApiKey model. RateLimitTier drives which
// rate-limit bucket the middleware applies (see internal/redis/ratelimit.go
// and internal/middleware/ratelimit.go).
type ApiKey struct {
	ID            uint64
	KeyHash       string
	UserID        uint64
	RateLimitTier string
	CreatedAt     time.Time
}

type LinkStore interface {
	// CreateLink persists a new link. shortCode must already be finalized
	// (base62-encoded ID, or a validated custom alias) before calling this.
	CreateLink(ctx context.Context, l *Link) error

	// GetLinkByShortCode fetches a link for the redirect hot path.
	// Returns ErrNotFound if no such link exists.
	GetLinkByShortCode(ctx context.Context, shortCode string) (*Link, error)
}

type UserStore interface {
	// GetOrCreateUserByEmail returns the existing user for this email, or
	// creates one if it doesn't exist yet. Kept deliberately simple (no
	// password) since Phase 2 only needs enough identity to own API keys.
	GetOrCreateUserByEmail(ctx context.Context, email string) (*User, error)
}

type ApiKeyStore interface {
	// CreateAPIKey persists a new API key. Only the hash is ever stored —
	// callers must not pass the raw key to this method.
	CreateAPIKey(ctx context.Context, k *ApiKey) error

	// GetAPIKeyByHash looks up an API key by its hash, for auth middleware.
	// Returns ErrNotFound if no such key exists.
	GetAPIKeyByHash(ctx context.Context, hash string) (*ApiKey, error)
}

// Store combines all the storage interfaces the app needs. Handlers and
// middleware depend on this (or the narrower interfaces above), never on
// Prisma directly.
type Store interface {
	LinkStore
	UserStore
	ApiKeyStore
}
