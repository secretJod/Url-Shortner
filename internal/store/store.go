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

// ClickEvent mirrors the Prisma ClickEvent model. These are written
// asynchronously by the analytics worker (Phase 3), not on the redirect
// hot path — the redirect handler pushes events to a Redis Stream and
// the worker drains them into Postgres.
type ClickEvent struct {
	ID         uint64
	LinkID     uint64
	Timestamp  time.Time
	Referrer   string
	Country    string
	DeviceType string
	IPHash     string
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

// ClickEventStore persists click events. Implemented by PrismaStore and
// used by the analytics worker (Phase 3) to drain the Redis Stream into
// Postgres in batches.
type ClickEventStore interface {
	// CreateClickEvent persists a single click event. Called by the
	// analytics worker after consuming from the Redis Stream.
	CreateClickEvent(ctx context.Context, e *ClickEvent) error
}

// LinkStats aggregates click analytics for a single link.
type LinkStats struct {
	Link         Link
	TotalClicks  int64
	UniqueIPs    int64
	ReferrerTop  []string
	DailyClicks  []DailyClicks
}

// DailyClicks groups click counts by calendar day.
type DailyClicks struct {
	Date  time.Time
	Count int64
}

// StatsStore provides analytics queries over links and click events.
// Implemented by PrismaStore and used by the stats/admin handlers (Phase 4).
type StatsStore interface {
	// GetLinkStats returns aggregated click stats for a single link.
	// Returns ErrNotFound if the link doesn't exist.
	GetLinkStats(ctx context.Context, shortCode string) (*LinkStats, error)

	// GetTopLinks returns the N most-clicked links.
	GetTopLinks(ctx context.Context, limit int) ([]*LinkStats, error)

	// GetUserLinks returns all links created by a specific user.
	GetUserLinks(ctx context.Context, userID uint64) ([]*Link, error)

	// GetRecentClickEvents returns the most recent click events for a link.
	GetRecentClickEvents(ctx context.Context, linkID uint64, limit int) ([]*ClickEvent, error)
}

// Store combines all the storage interfaces the app needs. Handlers and
// middleware depend on this (or the narrower interfaces above), never on
// Prisma directly.
type Store interface {
	LinkStore
	UserStore
	ApiKeyStore
	ClickEventStore
	StatsStore
}
