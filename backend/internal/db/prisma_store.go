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

func (s *PrismaStore) GetOrCreateUserByEmail(ctx context.Context, email string) (*store.User, error) {
	found, err := s.client.User.FindUnique(
		User.Email.Equals(email),
	).Exec(ctx)

	if err == nil {
		return &store.User{
			ID:        uint64(found.ID),
			Email:     found.Email,
			CreatedAt: found.CreatedAt,
		}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	created, err := s.client.User.CreateOne(
		User.Email.Set(email),
		// PasswordHash isn't part of Phase 2's scope (no login yet), so we
		// store an empty placeholder. Revisit if/when full auth is added.
		User.PasswordHash.Set(""),
	).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &store.User{
		ID:        uint64(created.ID),
		Email:     created.Email,
		CreatedAt: created.CreatedAt,
	}, nil
}

func (s *PrismaStore) CreateAPIKey(ctx context.Context, k *store.ApiKey) error {
	tier := k.RateLimitTier
	if tier == "" {
		tier = "standard"
	}

	created, err := s.client.APIKey.CreateOne(
		APIKey.KeyHash.Set(k.KeyHash),
		APIKey.User.Link(User.ID.Equals(types.BigInt(k.UserID))),
		APIKey.RateLimitTier.Set(tier),
	).Exec(ctx)
	if err != nil {
		return err
	}

	k.ID = uint64(created.ID)
	k.RateLimitTier = created.RateLimitTier
	k.CreatedAt = created.CreatedAt
	return nil
}

func (s *PrismaStore) GetAPIKeyByHash(ctx context.Context, hash string) (*store.ApiKey, error) {
	found, err := s.client.APIKey.FindUnique(
		APIKey.KeyHash.Equals(hash),
	).Exec(ctx)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	return &store.ApiKey{
		ID:            uint64(found.ID),
		KeyHash:       found.KeyHash,
		UserID:        uint64(found.UserID),
		RateLimitTier: found.RateLimitTier,
		CreatedAt:     found.CreatedAt,
	}, nil
}

// CreateClickEvent persists a click event. Called by the analytics worker
// (Phase 3) after consuming events from the Redis Stream. Each event is
// written individually — the worker handles batching at the stream-read
// level (reading N events per XReadGroup call) rather than at the DB level.
//
// Prisma's CreateOne signature for ClickEvent requires the link relation
// as a separate first arg (ClickEventWithPrismaLinkSetParam), with the
// remaining optional fields as variadic ClickEventSetParam — same pattern
// as CreateLink above.
func (s *PrismaStore) CreateClickEvent(ctx context.Context, e *store.ClickEvent) error {
	params := []ClickEventSetParam{
		ClickEvent.Timestamp.Set(e.Timestamp),
	}
	if e.Referrer != "" {
		params = append(params, ClickEvent.Referrer.Set(e.Referrer))
	}
	if e.Country != "" {
		params = append(params, ClickEvent.Country.Set(e.Country))
	}
	if e.DeviceType != "" {
		params = append(params, ClickEvent.DeviceType.Set(e.DeviceType))
	}
	if e.IPHash != "" {
		params = append(params, ClickEvent.IPHash.Set(e.IPHash))
	}

	created, err := s.client.ClickEvent.CreateOne(
		ClickEvent.Link.Link(Link.ID.Equals(types.BigInt(e.LinkID))),
		params...,
	).Exec(ctx)
	if err != nil {
		return err
	}

	e.ID = uint64(created.ID)
	return nil
}

// GetLinkStats returns aggregated click analytics for a link.
func (s *PrismaStore) GetLinkStats(ctx context.Context, shortCode string) (*store.LinkStats, error) {
	// First find the link.
	link, err := s.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	stats := &store.LinkStats{
		Link: *link,
	}

	// Fetch all click events for this link.
	allEvents, err := s.client.ClickEvent.FindMany(
		ClickEvent.LinkID.Equals(types.BigInt(link.ID)),
	).Exec(ctx)
	if err == nil {
		stats.TotalClicks = int64(len(allEvents))
	}

	// Count unique IPs and collect referrers.
	uniqueIPs := map[string]bool{}
	referrerCounts := map[string]int{}
	for _, ev := range allEvents {
		if ipHash, ok := ev.IPHash(); ok && ipHash != "" {
			uniqueIPs[ipHash] = true
		}
		if ref, ok := ev.Referrer(); ok && ref != "" {
			referrerCounts[ref]++
		}
	}
	stats.UniqueIPs = int64(len(uniqueIPs))

	// Top referrers (sorted by count, top 5).
	type refCount struct {
		ref   string
		count int
	}
	var refs []refCount
	for ref, cnt := range referrerCounts {
		refs = append(refs, refCount{ref, cnt})
	}
	// Simple insertion sort (fine for <50 items).
	for i := 1; i < len(refs); i++ {
		for j := i; j > 0 && refs[j].count > refs[j-1].count; j-- {
			refs[j], refs[j-1] = refs[j-1], refs[j]
		}
	}
	for i, r := range refs {
		if i >= 5 {
			break
		}
		stats.ReferrerTop = append(stats.ReferrerTop, r.ref)
	}

	// Group events by day for the daily click chart.
	dailyCounts := map[string]int64{}
	for _, ev := range allEvents {
		day := ev.Timestamp.Format("2006-01-02")
		dailyCounts[day]++
	}
	for day, cnt := range dailyCounts {
		parsed, _ := time.Parse("2006-01-02", day)
		stats.DailyClicks = append(stats.DailyClicks, store.DailyClicks{
			Date:  parsed,
			Count: cnt,
		})
	}
	// Sort daily clicks by date ascending.
	for i := 1; i < len(stats.DailyClicks); i++ {
		for j := i; j > 0 && stats.DailyClicks[j].Date.Before(stats.DailyClicks[j-1].Date); j-- {
			stats.DailyClicks[j], stats.DailyClicks[j-1] = stats.DailyClicks[j-1], stats.DailyClicks[j]
		}
	}

	return stats, nil
}

// GetTopLinks returns the N most-clicked links.
func (s *PrismaStore) GetTopLinks(ctx context.Context, limit int) ([]*store.LinkStats, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Get all links.
	links, err := s.client.Link.FindMany().Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Count clicks per link.
	clickCounts := map[uint64]int64{}
	allEvents, err := s.client.ClickEvent.FindMany().Exec(ctx)
	if err == nil {
		for _, ev := range allEvents {
			clickCounts[uint64(ev.LinkID)]++
		}
	}

	// Build LinkStats for each, sort by click count, take top N.
	var results []*store.LinkStats
	for _, l := range links {
		link := &store.Link{
			ID:          uint64(l.ID),
			ShortCode:   l.ShortCode,
			LongURL:     l.LongURL,
			CustomAlias: l.CustomAlias,
			CreatedAt:   l.CreatedAt,
		}
		if uid, ok := l.UserID(); ok {
			id := uint64(uid)
			link.UserID = &id
		}

		results = append(results, &store.LinkStats{
			Link:        *link,
			TotalClicks: clickCounts[uint64(l.ID)],
		})
	}

	// Sort by clicks descending (insertion sort, fine for small N).
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].TotalClicks > results[j-1].TotalClicks; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// GetUserLinks returns all links created by a specific user.
func (s *PrismaStore) GetUserLinks(ctx context.Context, userID uint64) ([]*store.Link, error) {
	found, err := s.client.Link.FindMany(
		Link.UserID.Equals(types.BigInt(userID)),
	).Exec(ctx)
	if err != nil {
		return nil, err
	}

	var links []*store.Link
	for _, l := range found {
		link := &store.Link{
			ID:          uint64(l.ID),
			ShortCode:   l.ShortCode,
			LongURL:     l.LongURL,
			CustomAlias: l.CustomAlias,
			CreatedAt:   l.CreatedAt,
		}
		if uid, ok := l.UserID(); ok {
			id := uint64(uid)
			link.UserID = &id
		}
		if exp, ok := l.ExpiresAt(); ok {
			link.ExpiresAt = &exp
		}
		links = append(links, link)
	}
	return links, nil
}

// GetRecentClickEvents returns the most recent click events for a link.
func (s *PrismaStore) GetRecentClickEvents(ctx context.Context, linkID uint64, limit int) ([]*store.ClickEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	found, err := s.client.ClickEvent.FindMany(
		ClickEvent.LinkID.Equals(types.BigInt(linkID)),
	).Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Sort by timestamp descending (most recent first).
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].Timestamp.After(found[j-1].Timestamp); j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	if len(found) > limit {
		found = found[:limit]
	}

	var events []*store.ClickEvent
	for _, ev := range found {
		e := &store.ClickEvent{
			ID:        uint64(ev.ID),
			LinkID:    linkID,
			Timestamp: ev.Timestamp,
		}
		if r, ok := ev.Referrer(); ok {
			e.Referrer = r
		}
		if c, ok := ev.Country(); ok {
			e.Country = c
		}
		if d, ok := ev.DeviceType(); ok {
			e.DeviceType = d
		}
		if iph, ok := ev.IPHash(); ok {
			e.IPHash = iph
		}
		events = append(events, e)
	}
	return events, nil
}
