// Package db contains the Prisma-backed implementation of store.LinkStore.
//
// IMPORTANT: this package imports the *generated* Prisma client, which does
// not exist until you run:
//
//	go run github.com/steebchen/prisma-client-go generate
//
// That command downloads Prisma's query-engine binary, which this dev
// sandbox's network allowlist blocks (only github.com/codeload.github.com
// etc. are reachable here — see PROJECT_OVERVIEW.md's "Dev environment
// note"). Run the generate command on your own machine, which has normal
// internet access, before building this package.
package db

import (
	"context"
	"errors"
	"time"

	"github.com/steebchen/prisma-client-go/runtime/types"

	"github.com/yourorg/urlshortener/internal/store"
)

// PrismaStore implements store.LinkStore backed by Postgres via Prisma.
type PrismaStore struct {
	client *PrismaClient
}

// NewPrismaStore connects to Postgres using DATABASE_URL and returns a
// ready-to-use LinkStore. Call Close() on shutdown.
func NewPrismaStore() (*PrismaStore, error) {
	client := NewClient() // generated constructor, from ./internal/db (post-codegen)
	if err := client.Prisma.Connect(); err != nil {
		return nil, err
	}
	return &PrismaStore{client: client}, nil
}

func (s *PrismaStore) Close() error {
	return s.client.Prisma.Disconnect()
}

func (s *PrismaStore) CreateLink(ctx context.Context, l *store.Link) error {
	// All optional Set()/relation params must go in ONE slice spread with
	// `...` — Go doesn't allow mixing individual variadic args with a
	// spread slice in the same call, so CustomAlias lives here too even
	// though it's always set.
	params := []LinkSetParam{
		Link.CustomAlias.Set(l.CustomAlias),
	}
	if l.UserID != nil {
		params = append(params, Link.User.Link(User.ID.Equals(types.BigInt(*l.UserID))))
	}
	if l.ExpiresAt != nil {
		params = append(params, Link.ExpiresAt.Set(*l.ExpiresAt))
	}
	if l.PasswordHash != nil {
		params = append(params, Link.PasswordHash.Set(*l.PasswordHash))
	}

	created, err := s.client.Link.CreateOne(
		Link.ShortCode.Set(l.ShortCode),
		Link.LongURL.Set(l.LongURL),
		params...,
	).Exec(ctx)
	if err != nil {
		return err
	}

	l.ID = uint64(created.ID)
	l.CreatedAt = created.CreatedAt
	return nil
}

func (s *PrismaStore) GetLinkByShortCode(ctx context.Context, shortCode string) (*store.Link, error) {
	found, err := s.client.Link.FindUnique(
		Link.ShortCode.Equals(shortCode),
	).Exec(ctx)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	result := &store.Link{
		ID:          uint64(found.ID),
		ShortCode:   found.ShortCode,
		LongURL:     found.LongURL,
		CustomAlias: found.CustomAlias,
		CreatedAt:   found.CreatedAt,
	}
	if expiresAt, ok := found.ExpiresAt(); ok {
		result.ExpiresAt = &expiresAt
	}
	if passwordHash, ok := found.PasswordHash(); ok {
		result.PasswordHash = &passwordHash
	}

	// Treat expired links as not found — caller shouldn't redirect to them.
	if result.ExpiresAt != nil && result.ExpiresAt.Before(time.Now()) {
		return nil, store.ErrNotFound
	}

	return result, nil
}
