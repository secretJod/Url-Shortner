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

type LinkStore interface {
	// CreateLink persists a new link. shortCode must already be finalized
	// (base62-encoded ID, or a validated custom alias) before calling this.
	CreateLink(ctx context.Context, l *Link) error

	// GetLinkByShortCode fetches a link for the redirect hot path.
	// Returns ErrNotFound if no such link exists.
	GetLinkByShortCode(ctx context.Context, shortCode string) (*Link, error)
}
