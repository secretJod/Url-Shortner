package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a short code isn't in the cache — the
// caller should fall back to Postgres and then backfill the cache.
var ErrCacheMiss = errors.New("redis: cache miss")

func cacheKey(shortCode string) string {
	return "urlshortener:link:" + shortCode
}

// aliasReservationKey is used to atomically reserve a custom alias before
// it's committed to Postgres, so two concurrent requests can't both win
// the same alias.
func aliasReservationKey(alias string) string {
	return "urlshortener:alias_reserved:" + alias
}

// defaultCacheTTL bounds how long a link can live in cache without being
// refreshed. Links with an explicit expiry use that instead (capped to this).
const defaultCacheTTL = 24 * time.Hour

// cachedLink is the JSON-serialized value stored in Redis for each
// short code. It includes both the long URL and the link's database ID,
// so the redirect handler can fire click events with the correct link_id
// even on cache hits (without a Postgres round-trip).
type cachedLink struct {
	LongURL string `json:"url"`
	LinkID  uint64 `json:"id"`
}

// GetLongURL returns the cached long URL and link ID for a short code,
// or ErrCacheMiss if it's not cached (caller should query Postgres and
// call SetLongURL).
//
// Backward compatibility: if the cached value is a plain string (from
// before Phase 3, when only the URL was cached), it's returned with
// linkID=0. The caller can check for 0 and skip the click event.
func (c *Client) GetLongURL(ctx context.Context, shortCode string) (longURL string, linkID uint64, err error) {
	val, err := c.rdb.Get(ctx, cacheKey(shortCode)).Result()
	if errors.Is(err, redis.Nil) {
		return "", 0, ErrCacheMiss
	}
	if err != nil {
		return "", 0, err
	}

	// Try to parse as JSON (new format with link ID).
	var cl cachedLink
	if json.Unmarshal([]byte(val), &cl) == nil && cl.LongURL != "" {
		return cl.LongURL, cl.LinkID, nil
	}

	// Old format (plain URL string) — return with linkID=0.
	return val, 0, nil
}

// SetLongURL caches a short code -> long URL + link ID mapping. If
// expiresAt is nil, the default TTL is used; otherwise the cache entry
// expires alongside the link itself (capped at defaultCacheTTL so stale
// long-lived links still get periodically refreshed from Postgres).
func (c *Client) SetLongURL(ctx context.Context, shortCode, longURL string, linkID uint64, expiresAt *time.Time) error {
	ttl := defaultCacheTTL
	if expiresAt != nil {
		if until := time.Until(*expiresAt); until > 0 && until < ttl {
			ttl = until
		}
	}

	cl := cachedLink{LongURL: longURL, LinkID: linkID}
	data, err := json.Marshal(cl)
	if err != nil {
		// Fallback: store plain URL if JSON marshaling fails (shouldn't happen).
		return c.rdb.Set(ctx, cacheKey(shortCode), longURL, ttl).Err()
	}
	return c.rdb.Set(ctx, cacheKey(shortCode), string(data), ttl).Err()
}

// InvalidateLongURL removes a short code from the cache, e.g. after deletion.
func (c *Client) InvalidateLongURL(ctx context.Context, shortCode string) error {
	return c.rdb.Del(ctx, cacheKey(shortCode)).Err()
}

// ReserveAlias atomically claims a custom alias. Returns true if this call
// won the reservation, false if someone else already holds it. The
// reservation has a short TTL as a safety net in case the caller crashes
// between reserving and committing to Postgres.
func (c *Client) ReserveAlias(ctx context.Context, alias string) (bool, error) {
	ok, err := c.rdb.SetNX(ctx, aliasReservationKey(alias), "1", 30*time.Second).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ReleaseAlias frees a reservation, e.g. if the Postgres write after it fails.
func (c *Client) ReleaseAlias(ctx context.Context, alias string) error {
	return c.rdb.Del(ctx, aliasReservationKey(alias)).Err()
}
