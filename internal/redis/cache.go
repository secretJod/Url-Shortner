package redis

import (
	"context"
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

// GetLongURL returns the cached long URL for a short code, or ErrCacheMiss
// if it's not cached (caller should query Postgres and call SetLongURL).
func (c *Client) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	val, err := c.rdb.Get(ctx, cacheKey(shortCode)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetLongURL caches a short code -> long URL mapping. If expiresAt is nil,
// the default TTL is used; otherwise the cache entry expires alongside the
// link itself (capped at defaultCacheTTL so stale long-lived links still
// get periodically refreshed from Postgres).
func (c *Client) SetLongURL(ctx context.Context, shortCode, longURL string, expiresAt *time.Time) error {
	ttl := defaultCacheTTL
	if expiresAt != nil {
		if until := time.Until(*expiresAt); until > 0 && until < ttl {
			ttl = until
		}
	}
	return c.rdb.Set(ctx, cacheKey(shortCode), longURL, ttl).Err()
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
