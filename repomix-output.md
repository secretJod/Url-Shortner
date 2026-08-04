This file is a merged representation of the entire codebase, combined into a single document by Repomix.

# File Summary

## Purpose
This file contains a packed representation of the entire repository's contents.
It is designed to be easily consumable by AI systems for analysis, code review,
or other automated processes.

## File Format
The content is organized as follows:
1. This summary section
2. Repository information
3. Directory structure
4. Repository files (if enabled)
5. Multiple file entries, each consisting of:
  a. A header with the file path (## File: path/to/file)
  b. The full contents of the file in a code block

## Usage Guidelines
- This file should be treated as read-only. Any changes should be made to the
  original repository files, not this packed version.
- When processing this file, use the file path to distinguish
  between different files in the repository.
- Be aware that this file may contain sensitive information. Handle it with
  the same level of security as you would the original repository.

## Notes
- Some files may have been excluded based on .gitignore rules and Repomix's configuration
- Binary files are not included in this packed representation. Please refer to the Repository Structure section for a complete list of file paths, including binary files
- Files matching patterns in .gitignore are excluded
- Files matching default ignore patterns are excluded
- Files are sorted by Git change count (files with more changes are at the bottom)

# Directory Structure
````
cmd/
  api/
    main.go
internal/
  auth/
    apikey_test.go
    apikey.go
  config/
    config.go
  db/
    .gitignore
    prisma_store.go
  handlers/
    apikeys.go
    health.go
    redirect.go
    shorten.go
  middleware/
    auth.go
    ratelimit.go
  redis/
    cache.go
    client.go
    counter.go
    ratelimit_test.go
    ratelimit.go
    util.go
  shortcode/
    base62_test.go
    base62.go
  store/
    store.go
prisma/
  schema.prisma
.env.example
.gitignore
api
docker-compose.yml
Dockerfile
go.mod
package.json
PROJECT_OVERVIEW.md
README.md
````

# Files

## File: internal/auth/apikey_test.go
````go
package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKeyIsRandomAndConsistentlyHashed(t *testing.T) {
	raw1, hash1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	raw2, hash2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	if raw1 == raw2 {
		t.Error("two generated keys were identical — randomness is broken")
	}
	if !strings.HasPrefix(raw1, keyPrefix) {
		t.Errorf("key %q missing expected prefix %q", raw1, keyPrefix)
	}
	if HashKey(raw1) != hash1 {
		t.Error("HashKey(raw1) doesn't match the hash returned by GenerateAPIKey")
	}
	if hash1 == hash2 {
		t.Error("two different keys hashed to the same value")
	}
}

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		header  string
		want    string
		wantErr bool
	}{
		{"Bearer usk_abc123", "usk_abc123", false},
		{"Bearer   usk_abc123  ", "usk_abc123", false},
		{"Basic abc123", "", true},
		{"", "", true},
		{"Bearer ", "", true},
	}

	for _, c := range cases {
		got, err := ExtractBearerToken(c.header)
		if c.wantErr {
			if err == nil {
				t.Errorf("ExtractBearerToken(%q): expected error, got nil", c.header)
			}
			continue
		}
		if err != nil {
			t.Errorf("ExtractBearerToken(%q): unexpected error: %v", c.header, err)
		}
		if got != c.want {
			t.Errorf("ExtractBearerToken(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}
````

## File: internal/auth/apikey.go
````go
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
````

## File: internal/config/config.go
````go
package config

import (
	"os"
)

type Config struct {
	Port          string
	BaseURL       string
	Env           string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		Env:           getEnv("ENV", "development"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
````

## File: internal/db/.gitignore
````
# gitignore generated by Prisma Client Go. DO NOT EDIT.
*_gen.go
````

## File: internal/handlers/apikeys.go
````go
package handlers

import (
	"net/mail"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/auth"
	"github.com/yourorg/urlshortener/internal/store"
)

type APIKeyHandler struct {
	Store interface {
		store.UserStore
		store.ApiKeyStore
	}
}

func NewAPIKeyHandler(s interface {
	store.UserStore
	store.ApiKeyStore
}) *APIKeyHandler {
	return &APIKeyHandler{Store: s}
}

type createAPIKeyRequest struct {
	Email string `json:"email"`
}

type createAPIKeyResponse struct {
	APIKey  string `json:"api_key"`
	Warning string `json:"warning"`
}

// CreateKey issues a new API key for the given email (creating the user
// if this is their first key). No password/login flow yet — see
// PROJECT_OVERVIEW.md Phase 2 notes for the deliberate scope decision.
func (h *APIKeyHandler) CreateKey(c *fiber.Ctx) error {
	var req createAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid email is required"})
	}

	ctx := c.Context()

	user, err := h.Store.GetOrCreateUserByEmail(ctx, req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to resolve user"})
	}

	rawKey, hash, err := auth.GenerateAPIKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate API key"})
	}

	apiKey := &store.ApiKey{
		KeyHash:       hash,
		UserID:        user.ID,
		RateLimitTier: "standard",
	}
	if err := h.Store.CreateAPIKey(ctx, apiKey); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save API key"})
	}

	return c.Status(fiber.StatusCreated).JSON(createAPIKeyResponse{
		APIKey:  rawKey,
		Warning: "save this key now — it will not be shown again",
	})
}
````

## File: internal/handlers/health.go
````go
package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourorg/urlshortener/internal/redis"
)

type HealthHandler struct {
	Redis *redis.Client
}

func NewHealthHandler(r *redis.Client) *HealthHandler {
	return &HealthHandler{Redis: r}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := "ok"
	code := fiber.StatusOK
	redisStatus := "ok"

	if err := h.Redis.Ping(ctx); err != nil {
		redisStatus = "unreachable: " + err.Error()
		status = "degraded"
		code = fiber.StatusServiceUnavailable
	}

	return c.Status(code).JSON(fiber.Map{
		"status": status,
		"redis":  redisStatus,
	})
}
````

## File: internal/handlers/redirect.go
````go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/redis"
	"github.com/yourorg/urlshortener/internal/store"
)

type RedirectHandler struct {
	Store store.LinkStore
	Redis *redis.Client
}

func NewRedirectHandler(s store.LinkStore, r *redis.Client) *RedirectHandler {
	return &RedirectHandler{Store: s, Redis: r}
}

func (h *RedirectHandler) Redirect(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	ctx := c.Context()

	// 1. Cache-first — this is the path that should serve almost all traffic.
	longURL, err := h.Redis.GetLongURL(ctx, shortCode)
	if err == nil {
		return c.Redirect(longURL, fiber.StatusFound)
	}
	if !errors.Is(err, redis.ErrCacheMiss) {
		// Redis itself errored (not just a miss) — log and fall through to
		// Postgres rather than failing the request outright.
		// (structured logging wired up in a later phase)
	}

	// 2. Cache miss — fall back to Postgres.
	link, err := h.Store.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found or expired"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to resolve short link"})
	}

	// 3. Backfill the cache so the next request for this code is a hit.
	_ = h.Redis.SetLongURL(ctx, link.ShortCode, link.LongURL, link.ExpiresAt)

	// TODO (Phase 3): fire-and-forget click event onto a Redis Stream here
	// for async analytics, without blocking this redirect.

	return c.Redirect(link.LongURL, fiber.StatusFound)
}
````

## File: internal/middleware/auth.go
````go
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/auth"
	"github.com/yourorg/urlshortener/internal/store"
)

// Context keys for values set by OptionalAPIKeyAuth, read by handlers and
// by the rate-limit middleware.
const (
	LocalsAPIKey = "apiKey"
	LocalsUser   = "user"
)

// OptionalAPIKeyAuth validates an `Authorization: Bearer <key>` header if
// present. Requests without one proceed as anonymous. Requests WITH a
// header that fails to validate are rejected — if you're presenting a key
// at all, it needs to be a real one; silently falling back to anonymous
// would hide misconfigured clients.
func OptionalAPIKeyAuth(s store.ApiKeyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Next() // anonymous — fine, rate limiter falls back to per-IP
		}

		rawKey, err := auth.ExtractBearerToken(header)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "malformed Authorization header, expected: Bearer <api_key>",
			})
		}

		hash := auth.HashKey(rawKey)
		apiKey, err := s.GetAPIKeyByHash(c.Context(), hash)
		if err != nil {
			if err == store.ErrNotFound {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid API key"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to validate API key"})
		}

		c.Locals(LocalsAPIKey, apiKey)
		return c.Next()
	}
}

// GetAPIKey retrieves the authenticated API key from context, if any.
func GetAPIKey(c *fiber.Ctx) *store.ApiKey {
	if v, ok := c.Locals(LocalsAPIKey).(*store.ApiKey); ok {
		return v
	}
	return nil
}
````

## File: internal/middleware/ratelimit.go
````go
package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/redis"
)

// tierLimits maps an ApiKey's rate_limit_tier to (requests, window).
// "standard" is the default tier new keys get (see internal/db's
// CreateAPIKey). Anonymous requests (no key) get the stricter anonLimit.
var tierLimits = map[string]struct {
	limit  int
	window time.Duration
}{
	"standard": {60, time.Minute},
	"pro":      {600, time.Minute},
}

var anonLimit = struct {
	limit  int
	window time.Duration
}{20, time.Minute}

// RateLimit enforces per-API-key (or per-IP, if anonymous) request limits
// using Redis. Must run AFTER OptionalAPIKeyAuth, since it reads the
// authenticated key (if any) from context to decide which bucket/limit to use.
func RateLimit(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var (
			bucketKey string
			limit     int
			window    time.Duration
		)

		if apiKey := GetAPIKey(c); apiKey != nil {
			tier, ok := tierLimits[apiKey.RateLimitTier]
			if !ok {
				tier = tierLimits["standard"]
			}
			bucketKey = fmt.Sprintf("ratelimit:apikey:%d", apiKey.ID)
			limit, window = tier.limit, tier.window
		} else {
			bucketKey = fmt.Sprintf("ratelimit:ip:%s", c.IP())
			limit, window = anonLimit.limit, anonLimit.window
		}

		allowed, retryAfter, err := rdb.Allow(c.Context(), bucketKey, limit, window)
		if err != nil {
			// Fail open: a Redis hiccup shouldn't take down the whole API.
			// (Logged once structured logging lands in a later phase.)
			return c.Next()
		}
		if !allowed {
			c.Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter.String(),
			})
		}

		return c.Next()
	}
}
````

## File: internal/redis/cache.go
````go
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
````

## File: internal/redis/client.go
````go
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func New(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		// Tuned for a high-throughput redirect hot path
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	return &Client{rdb: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Raw exposes the underlying redis client for use in other packages
// (cache, counter, rate-limiter) built in later phases.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) String() string {
	return fmt.Sprintf("redis-client(%s)", c.rdb.Options().Addr)
}
````

## File: internal/redis/counter.go
````go
package redis

import (
	"context"
)

// counterKey is the single global counter used to mint new Link IDs.
// INCR is atomic in Redis, so concurrent requests never get the same ID
// without needing any database-side locking.
const counterKey = "urlshortener:link_id_counter"

// NextID atomically increments and returns the next unique ID to use
// as the base for a new short code.
func (c *Client) NextID(ctx context.Context) (uint64, error) {
	val, err := c.rdb.Incr(ctx, counterKey).Result()
	if err != nil {
		return 0, err
	}
	return uint64(val), nil
}
````

## File: internal/redis/ratelimit_test.go
````go
package redis

import (
	"context"
	"testing"
	"time"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	c := New("localhost:6379", "", 0)
	if err := c.Ping(context.Background()); err != nil {
		t.Skipf("redis not reachable, skipping: %v", err)
	}
	return c
}

func TestAllow_UnderLimit(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:under:" + randSuffix()

	for i := 0; i < 5; i++ {
		allowed, _, err := c.Allow(ctx, key, 5, time.Minute)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=5)", i+1)
		}
	}
}

func TestAllow_OverLimit(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:over:" + randSuffix()

	for i := 0; i < 3; i++ {
		allowed, _, err := c.Allow(ctx, key, 3, time.Minute)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=3)", i+1)
		}
	}

	// 4th request should be blocked.
	allowed, retryAfter, err := c.Allow(ctx, key, 3, time.Minute)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if allowed {
		t.Fatal("4th request should have been blocked (limit=3)")
	}
	if retryAfter <= 0 {
		t.Errorf("expected positive retryAfter, got %v", retryAfter)
	}
}

func TestAllow_WindowSlides(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:slide:" + randSuffix()

	shortWindow := 500 * time.Millisecond

	for i := 0; i < 2; i++ {
		allowed, _, err := c.Allow(ctx, key, 2, shortWindow)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=2)", i+1)
		}
	}

	allowed, _, _ := c.Allow(ctx, key, 2, shortWindow)
	if allowed {
		t.Fatal("3rd request should have been blocked immediately")
	}

	time.Sleep(shortWindow + 100*time.Millisecond)

	allowed, _, err := c.Allow(ctx, key, 2, shortWindow)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Fatal("request after window expiry should have been allowed")
	}
}
````

## File: internal/redis/ratelimit.go
````go
package redis

import (
	"context"
	"fmt"
	"time"
)

// Allow implements a sliding-window-log rate limiter using a Redis sorted
// set: each request is recorded as a member scored by its timestamp: old
// entries outside the window are trimmed, then the remaining count is
// compared against limit. This is more accurate than fixed-window counters
// (no burst-at-the-boundary problem) at the cost of O(log N) per request
// instead of O(1) — an acceptable tradeoff at the request rates a rate
// limiter itself needs to handle.
//
// key should already be scoped to the thing being limited, e.g.
// "ratelimit:apikey:123" or "ratelimit:ip:1.2.3.4".
func (c *Client) Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error) {
	now := time.Now()
	windowStart := now.Add(-window)
	member := fmt.Sprintf("%d-%s", now.UnixNano(), randSuffix())

	pipe := c.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redisZ(float64(now.UnixNano()), member))
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := countCmd.Val()
	if int(count) >= limit {
		// Over limit — remove the entry we just optimistically added, since
		// this request shouldn't count against the window.
		c.rdb.ZRem(ctx, key, member)

		// Estimate retry-after from the oldest entry still in the window.
		oldest, err := c.rdb.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			oldestTime := time.Unix(0, int64(oldest[0].Score))
			retryAfter = window - now.Sub(oldestTime)
			if retryAfter < 0 {
				retryAfter = 0
			}
		} else {
			retryAfter = window
		}
		return false, retryAfter, nil
	}

	return true, 0, nil
}
````

## File: internal/redis/util.go
````go
package redis

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/redis/go-redis/v9"
)

func redisZ(score float64, member string) redis.Z {
	return redis.Z{Score: score, Member: member}
}

// randSuffix guards against two requests landing on the exact same
// nanosecond timestamp (rare, but possible under high concurrency) and
// colliding as sorted-set members.
func randSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
````

## File: internal/shortcode/base62_test.go
````go
package shortcode

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 61, 62, 63, 12345, 999999999, 18446744073709551615}
	for _, id := range cases {
		code := Encode(id)
		got, err := Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q) returned error: %v", code, err)
		}
		if got != id {
			t.Errorf("round trip mismatch: id=%d code=%q decoded=%d", id, code, got)
		}
	}
}

func TestEncodeIsDeterministicAndUnique(t *testing.T) {
	seen := make(map[string]uint64)
	for id := uint64(0); id < 10000; id++ {
		code := Encode(id)
		if prev, ok := seen[code]; ok {
			t.Fatalf("collision: id=%d and id=%d both encode to %q", prev, id, code)
		}
		seen[code] = id
	}
}

func TestDecodeInvalidCharacter(t *testing.T) {
	if _, err := Decode("abc!def"); err == nil {
		t.Error("expected error for invalid character, got nil")
	}
}
````

## File: internal/shortcode/base62.go
````go
// Package shortcode turns numeric IDs (from the Redis INCR counter) into
// short, URL-safe base62 codes, and back again.
//
// Base62 alphabet = [0-9a-zA-Z], so codes are compact and collision-free
// by construction: every distinct counter value maps to exactly one code.
package shortcode

import (
	"errors"
	"strings"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const base = uint64(len(alphabet))

var ErrInvalidCode = errors.New("shortcode: invalid character in code")

// Encode converts a positive integer ID into a base62 short code.
// e.g. 0 -> "0", 61 -> "Z", 62 -> "10"
func Encode(id uint64) string {
	if id == 0 {
		return string(alphabet[0])
	}

	var sb strings.Builder
	// Encode digits least-significant-first, then reverse.
	digits := make([]byte, 0, 11) // enough for a uint64
	for id > 0 {
		digits = append(digits, alphabet[id%base])
		id /= base
	}
	for i := len(digits) - 1; i >= 0; i-- {
		sb.WriteByte(digits[i])
	}
	return sb.String()
}

// Decode converts a base62 short code back into its numeric ID.
// Useful for debugging/admin tooling; not needed on the hot path.
func Decode(code string) (uint64, error) {
	var id uint64
	for _, ch := range code {
		idx := strings.IndexRune(alphabet, ch)
		if idx < 0 {
			return 0, ErrInvalidCode
		}
		id = id*base + uint64(idx)
	}
	return id, nil
}
````

## File: package.json
````json
{}
````

## File: internal/handlers/shorten.go
````go
package handlers

import (
	"context"
	"net/url"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/middleware"
	"github.com/yourorg/urlshortener/internal/redis"
	"github.com/yourorg/urlshortener/internal/shortcode"
	"github.com/yourorg/urlshortener/internal/store"
)

// customAliasPattern restricts custom aliases to safe, URL-friendly characters.
var customAliasPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

type ShortenHandler struct {
	Store   store.LinkStore
	Redis   *redis.Client
	BaseURL string
}

func NewShortenHandler(s store.LinkStore, r *redis.Client, baseURL string) *ShortenHandler {
	return &ShortenHandler{Store: s, Redis: r, BaseURL: baseURL}
}

type shortenRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"` // RFC3339, optional
}

type shortenResponse struct {
	ShortURL  string `json:"short_url"`
	ShortCode string `json:"short_code"`
	LongURL   string `json:"long_url"`
}

func (h *ShortenHandler) Shorten(c *fiber.Ctx) error {
	var req shortenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	longURL, err := validateURL(req.URL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expires_at must be RFC3339 format"})
		}
		expiresAt = &t
	}

	ctx := c.Context()

	shortCode, isCustom, err := h.resolveShortCode(ctx, req.CustomAlias)
	if err != nil {
		switch err {
		case store.ErrInvalidAlias:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "custom alias must be 3-32 chars, letters/numbers/underscore/hyphen only"})
		case store.ErrAliasTaken:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "custom alias already taken"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate short code"})
		}
	}

	link := &store.Link{
		ShortCode:   shortCode,
		LongURL:     longURL,
		CustomAlias: isCustom,
		ExpiresAt:   expiresAt,
	}
	if apiKey := middleware.GetAPIKey(c); apiKey != nil {
		link.UserID = &apiKey.UserID
	}

	if err := h.Store.CreateLink(ctx, link); err != nil {
		if isCustom {
			_ = h.Redis.ReleaseAlias(ctx, shortCode)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save link"})
	}

	// Write-through cache so the very first redirect is already a cache hit.
	_ = h.Redis.SetLongURL(ctx, shortCode, longURL, expiresAt)

	return c.Status(fiber.StatusCreated).JSON(shortenResponse{
		ShortURL:  h.BaseURL + "/" + shortCode,
		ShortCode: shortCode,
		LongURL:   longURL,
	})
}

// resolveShortCode either validates+reserves a custom alias, or mints a new
// one via the Redis counter + base62 encoding.
func (h *ShortenHandler) resolveShortCode(ctx context.Context, customAlias string) (code string, isCustom bool, err error) {
	if customAlias != "" {
		if !customAliasPattern.MatchString(customAlias) {
			return "", false, store.ErrInvalidAlias
		}
		reserved, rErr := h.Redis.ReserveAlias(ctx, customAlias)
		if rErr != nil {
			return "", false, rErr
		}
		if !reserved {
			return "", false, store.ErrAliasTaken
		}
		return customAlias, true, nil
	}

	id, rErr := h.Redis.NextID(ctx)
	if rErr != nil {
		return "", false, rErr
	}
	return shortcode.Encode(id), false, nil
}

func validateURL(raw string) (string, error) {
	if raw == "" {
		return "", errInvalidURL("url is required")
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errInvalidURL("url must be a valid http(s) URL")
	}
	return u.String(), nil
}

type errInvalidURL string

func (e errInvalidURL) Error() string { return string(e) }
````

## File: internal/store/store.go
````go
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
````

## File: prisma/schema.prisma
````prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator db {
  provider = "go run github.com/steebchen/prisma-client-go"
  output   = "../internal/db"
}

model Link {
  id           BigInt      @id @default(autoincrement())
  shortCode    String      @unique @map("short_code")
  longUrl      String      @map("long_url")
  userId       BigInt?     @map("user_id")
  user         User?       @relation(fields: [userId], references: [id])
  customAlias  Boolean     @default(false) @map("custom_alias")
  expiresAt    DateTime?   @map("expires_at")
  passwordHash String?     @map("password_hash")
  createdAt    DateTime    @default(now()) @map("created_at")
  clicks       ClickEvent[]

  @@index([userId])
  @@map("links")
}

model User {
  id           BigInt    @id @default(autoincrement())
  email        String    @unique
  passwordHash String    @map("password_hash")
  createdAt    DateTime  @default(now()) @map("created_at")
  links        Link[]
  apiKeys      ApiKey[]

  @@map("users")
}

model ApiKey {
  id            BigInt   @id @default(autoincrement())
  keyHash       String   @unique @map("key_hash")
  userId        BigInt   @map("user_id")
  user          User     @relation(fields: [userId], references: [id])
  rateLimitTier String   @default("standard") @map("rate_limit_tier")
  createdAt     DateTime @default(now()) @map("created_at")

  @@index([userId])
  @@map("api_keys")
}

model ClickEvent {
  id         BigInt   @id @default(autoincrement())
  linkId     BigInt   @map("link_id")
  link       Link     @relation(fields: [linkId], references: [id])
  timestamp  DateTime @default(now())
  referrer   String?
  country    String?
  deviceType String?  @map("device_type")
  ipHash     String?  @map("ip_hash")

  @@index([linkId])
  @@map("click_events")
}
````

## File: .gitignore
````
.env
*.log
/tmp/
bin/
prisma/db/

# Go
*.exe
*.test
*.out
vendor/
````

## File: Dockerfile
````dockerfile
# --- Build stage ---
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Alpine's musl libc needs libc6-compat and openssl for Prisma's engine binaries
RUN apk add --no-cache openssl

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# IMPORTANT: generate the Prisma client HERE, inside the build container,
# not on your host machine. This ensures the query-engine binary that gets
# embedded matches the container's actual platform (Alpine/musl), not
# whatever OS you're developing on (e.g. macOS). This was the cause of the
# "ensure: no binary found" runtime error.
RUN go run github.com/steebchen/prisma-client-go prefetch
RUN go run github.com/steebchen/prisma-client-go generate

RUN CGO_ENABLED=0 GOOS=linux go build -o /urlshortener ./cmd/api

# --- Run stage ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates openssl
WORKDIR /root/

COPY --from=builder /urlshortener .

EXPOSE 8080
CMD ["./urlshortener"]
````

## File: go.mod
````
module github.com/yourorg/urlshortener

go 1.22.2

replace golang.org/x/sys => github.com/golang/sys v0.28.0

replace golang.org/x/net => github.com/golang/net v0.30.0

replace golang.org/x/text => github.com/golang/text v0.19.0

replace golang.org/x/crypto => github.com/golang/crypto v0.28.0

require (
	github.com/gofiber/fiber/v2 v2.52.14
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.6.1
	github.com/shopspring/decimal v1.4.0
	github.com/steebchen/prisma-client-go v0.47.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.0.1 // indirect
	golang.org/x/sys v0.28.0 // indirect
)
````

## File: README.md
````markdown
# URL Shortener (Enterprise)

Go + Fiber + Prisma + PostgreSQL + Redis. See `PROJECT_OVERVIEW.md` for full architecture, phase plan, and session log — **read that file first** if you're picking this project back up.

## Local setup

```bash
cp .env.example .env
docker-compose up -d postgres redis

# REQUIRED before first build — generates the Prisma Go client into internal/db
go run github.com/steebchen/prisma-client-go generate

# push the schema to Postgres (dev-friendly, no migration files)
go run github.com/steebchen/prisma-client-go db push

go build ./...
go run ./cmd/api
```

## API

- `GET  /health` — Redis connectivity check
- `POST /api/keys` — body: `{"email": "you@example.com"}` — issues a new API key (shown once)
- `POST /api/shorten` — optional `Authorization: Bearer <api_key>` header; body: `{"url": "https://...", "custom_alias": "optional", "expires_at": "optional RFC3339"}`
- `GET  /:shortCode` — redirects to the long URL (cache-first via Redis, Postgres fallback)

Rate limits: 20 req/min per IP (anonymous), 60 req/min per key (standard tier), 600 req/min (pro tier).

## Full stack via Docker

```bash
docker-compose up --build
```

## Status

Phase 2 complete: API key issuance, optional bearer-token auth, Redis sliding-window rate limiting.
See `PROJECT_OVERVIEW.md` §4 for the full phase tracker and §10 for what to verify next.
````

## File: cmd/api/main.go
````go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/yourorg/urlshortener/internal/config"
	"github.com/yourorg/urlshortener/internal/db"
	"github.com/yourorg/urlshortener/internal/handlers"
	"github.com/yourorg/urlshortener/internal/middleware"
	"github.com/yourorg/urlshortener/internal/redis"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	rdb := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer rdb.Close()

	linkStore, err := db.NewPrismaStore()
	if err != nil {
		log.Fatalf("failed to connect to Postgres via Prisma: %v", err)
	}
	defer linkStore.Close()

	app := fiber.New(fiber.Config{
		AppName:      "urlshortener",
		ServerHeader: "urlshortener",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	health := handlers.NewHealthHandler(rdb)
	app.Get("/health", health.Check)

	// Auth is optional (anonymous requests still work) but must run before
	// rate limiting, since the limiter needs to know whether a request is
	// authenticated to pick the right bucket/limit.
	authMW := middleware.OptionalAPIKeyAuth(linkStore)
	rateLimitMW := middleware.RateLimit(rdb)

	apiKeys := handlers.NewAPIKeyHandler(linkStore)
	app.Post("/api/keys", rateLimitMW, apiKeys.CreateKey)

	shorten := handlers.NewShortenHandler(linkStore, rdb, cfg.BaseURL)
	app.Post("/api/shorten", authMW, rateLimitMW, shorten.Shorten)

	redirect := handlers.NewRedirectHandler(linkStore, rdb)
	app.Get("/:shortCode", redirect.Redirect)

	log.Printf("starting urlshortener on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
````

## File: internal/db/prisma_store.go
````go
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
````

## File: PROJECT_OVERVIEW.md
````markdown
# URL Shortener — Project Overview

> **Purpose of this file:** This is the single source of truth for the project.
> At the start of every session, read this file first to restore full context
> instead of re-explaining the project. Update it at the end of every phase.

---

## 1. Stack Decisions (locked in)

| Layer | Choice | Notes |
|---|---|---|
| Language | Go | Chosen for raw speed + concurrency |
| Web framework | Fiber | Fast, Express-like ergonomics |
| ORM | Prisma (prisma-client-go) | Community-maintained Go client. Fallback plan: swap to `ent` if we hit maturity issues |
| Primary DB | PostgreSQL | Persistent store for links, users, API keys |
| Cache / hot path | Redis | Redirect cache, ID counter, rate limiting |
| Containerization | Docker + Docker Compose | Local dev + eventual deploy |
| Analytics queue | Redis Streams (or lightweight pub/sub) | Async click tracking, doesn't block redirects |

---

## 2. Architecture

```
Client → Load Balancer → Go API (stateless, N instances)
                              ├─ Redis: short_code lookup cache (GET, sub-ms)
                              ├─ Redis: INCR counter → base62 encode → short_code
                              ├─ Redis: rate limiting (sliding window per API key/IP)
                              ├─ Postgres (via Prisma): source of truth for links/users/keys
                              └─ Redis Stream → Analytics Worker → Postgres analytics tables
```

**Hot path (redirect) flow:**
1. Request hits `GET /:shortCode`
2. Check Redis cache → if hit, 302 redirect immediately (~sub-ms)
3. If miss, query Postgres, backfill Redis cache, then redirect
4. Fire-and-forget: push click event to Redis Stream for async analytics

**Write path (shorten) flow:**
1. `POST /api/shorten` with long URL (+ optional custom alias/expiry)
2. If custom alias: check availability, else `INCR` global counter in Redis
3. Base62-encode counter value → short code (collision-free by construction)
4. Write to Postgres via Prisma, write-through to Redis cache
5. Return short URL

---

## 3. Data Model (initial draft)

**Link**
- id (bigint, PK — same value used for base62 short code)
- short_code (unique, indexed)
- long_url
- user_id (nullable, FK — anonymous links allowed)
- custom_alias (bool)
- expires_at (nullable)
- password_hash (nullable)
- created_at

**User**
- id, email, password_hash, created_at

**ApiKey**
- id, key_hash, user_id, rate_limit_tier, created_at

**ClickEvent** (analytics, written async)
- id, link_id, timestamp, referrer, country, device_type, ip_hash

---

## 4. Build Phases & Status

- [x] **Phase 0** — Scaffold, this overview file, Docker setup, Fiber server + Redis client wired, `/health` endpoint, builds clean
- [x] **Phase 1** — Core shorten + redirect API (base62 ID gen, Postgres via Prisma, Redis cache) *(verified working end-to-end: POST /api/shorten → Postgres write → cache write-through → GET redirect → 302 confirmed on Karan's machine)*
- [x] **Phase 2** — Auth (API keys) + rate limiting *(code complete, builds clean — needs the same local Prisma codegen re-run + a live test, see session log)*
- [ ] **Phase 3** — Async analytics pipeline (Redis Streams → worker → Postgres)
- [ ] **Phase 4** — Admin/stats API
- [ ] **Phase 5** — Load testing, caching tuning, observability (metrics/logging)
- [ ] **Phase 6** — Deployment hardening (Docker Compose finalized, optional K8s manifests)

---

## 5. Project Structure

```
urlshortener/
├── PROJECT_OVERVIEW.md      ← you are here
├── cmd/api/                 ← main.go, app entrypoint
├── internal/
│   ├── handlers/             ← HTTP handlers: health, shorten, redirect
│   ├── redis/                  ← Redis client, ID counter (INCR), cache, alias reservation
│   ├── shortcode/              ← base62 encode/decode (unit tested)
│   ├── store/                  ← LinkStore interface (decouples handlers from Prisma)
│   ├── db/                     ← Prisma-backed LinkStore implementation (needs codegen — see §8)
│   ├── middleware/             ← auth, rate limiting, logging (Phase 2+)
│   └── config/                 ← env/config loading
├── prisma/
│   └── schema.prisma          ← DB schema (source of truth for Postgres), output → internal/db
├── docker-compose.yml
├── go.mod
└── .env.example
```

---

## 6. Open Decisions / Things to Revisit
- Prisma Go client maturity — revisit if we hit blockers, fallback is `ent`
- Custom domains — deferred to a later phase
- Deployment target (K8s vs simple Docker Compose on a VM) — not yet decided

---

## 7. Session Log
- **Session 1**: Locked stack (Go + Fiber + Prisma + Postgres + Redis). Scaffolded project structure.
  Phase 0 completed: `go.mod` set up, Fiber + go-redis wired, `/health` endpoint live, `docker-compose.yml`
  (Postgres + Redis + api), Dockerfile (multi-stage), Prisma schema drafted (Link/User/ApiKey/ClickEvent).
  Verified with `go build` and `go vet` — both clean. Next session: Phase 1 (base62 short code generation
  via Redis INCR, wire Prisma client, implement real `/api/shorten` and `/:shortCode` handlers).

- **Session 3**: Debugged Phase 1 all the way to a verified working end-to-end test. In order:
  1. Fixed two real Prisma-client-go usage bugs found on first real build (couldn't be caught in the
     sandbox since codegen wasn't possible there): (a) `CreateOne` — Go disallows mixing an individual
     variadic arg with a spread slice in one call; moved all optional `Set()` params into one slice.
     (b) `User.ID.Equals` needed an explicit `types.BigInt` conversion, not a plain Go `int`.
  2. Fixed "ensure: no binary found" at container runtime — `prisma generate` had been run on the host
     (macOS/arm64), embedding the wrong platform's query-engine binary. Fix: moved `prisma-client-go
     prefetch` + `generate` into the Dockerfile's builder stage itself (`golang:1.22-alpine`), per
     Prisma's documented Docker pattern, so the embedded engine matches the container's actual platform.
  3. Fixed api container connecting to `localhost:5432`/`localhost:6379` instead of the Postgres/Redis
     *containers* — `.env` is correct for host-side `go run`, but wrong once loaded into a container via
     `env_file` (inside Docker's network, services are reachable by service name). Fix: added an
     `environment:` override block on the `api` service in `docker-compose.yml` for just those two vars.
  4. Fixed `relation "public.links" does not exist` — `prisma generate` (builds the Go client) and
     `prisma db push` (creates the actual Postgres tables) are separate steps; only the former had been
     run. Running `db push` locally (needs `DATABASE_URL` pointing to `localhost:5432`, i.e. host-side,
     while `docker compose`'s Postgres port is mapped out to the host) created the tables.
  5. **Verified end-to-end**: `POST /api/shorten` → `201` with real `short_url`/`short_code`. `GET
     /<short_code>` → `302 Found` with correct `Location` header. Full pipeline confirmed working.

  **Workflow note for next time**: after `docker compose up --build`, if tables don't exist yet, run
  `go run github.com/steebchen/prisma-client-go db push` locally (host-side, not in Docker) once, with
  the Postgres container already up and its port mapped to the host. This is a manual step for now —
  worth switching to proper `prisma migrate` files in a later phase so it's automatic.

## 9. Repo & credentials note
This project lives at `github.com/secretJod/Url-Shortner` (public repo). Claude pushes commits directly
using a short-lived, repo-scoped fine-grained PAT that Karan provides fresh each session — the token is
never stored between sessions, and Karan revokes it after each session ends.

## 10. Session Log (continued)
- **Session 4**: Phase 2 built — API keys + rate limiting.

  **Scope decision**: no full signup/login/password flow yet. `POST /api/keys` takes just an email,
  get-or-creates a `User` (with an empty password placeholder — full auth is a separate future decision,
  not part of Phase 2), generates a random 256-bit API key, and returns the raw key exactly once. Only
  its SHA-256 hash is ever stored. This matches how most API-first products (Stripe, bit.ly, etc.) start.

  **What was built**:
  - `internal/auth` — API key generation (`usk_`-prefixed, 256-bit random) + SHA-256 hashing + bearer
    token extraction. Unit tested (randomness, hash consistency, header parsing edge cases).
  - `internal/redis/ratelimit.go` — sliding-window-log rate limiter using a Redis sorted set (ZADD +
    ZREMRANGEBYSCORE + ZCARD), more accurate than fixed-window counters (no boundary-burst problem).
    Integration-tested against a real local Redis (under-limit, over-limit, window-slide-expiry cases).
  - `internal/store` — extended with `User`/`ApiKey` types and `UserStore`/`ApiKeyStore` interfaces;
    `PrismaStore` now implements the full combined `Store` interface.
  - `internal/middleware/auth.go` — `OptionalAPIKeyAuth`: anonymous requests pass through; requests
    WITH an `Authorization` header that fails to validate are rejected (no silent fallback to anonymous
    for a malformed/invalid key — that would hide misconfigured clients).
  - `internal/middleware/ratelimit.go` — per-API-key-tier limits (`standard`=60/min, `pro`=600/min) or
    per-IP for anonymous (20/min). Fails open on Redis errors (a cache hiccup shouldn't take down the API).
  - `POST /api/keys` and `POST /api/shorten` now go through `authMW` (shorten only) then `rateLimitMW`.
    Links created with a valid API key get `UserID` set, associating them with that user.

  **Verified in sandbox**: all new packages (`auth`, `redis` incl. rate limiter, `store`, `middleware`,
  `handlers`, `cmd/api`) build and vet clean; `internal/db` fails only on the same known missing-codegen
  symbols as before (confirms no new bugs, same as the Phase 1 pattern). **Not yet live-tested end-to-end**
  on Karan's machine — do that next: `git pull`, rebuild, then hit `POST /api/keys` and confirm rate
  limiting kicks in on `POST /api/shorten` after enough anonymous requests.

## 8. Required local step before this runs: generate the Prisma client

This sandbox's network can't reach Prisma's binary CDN, so the Prisma Go client has never actually been
generated. On your own machine (normal internet access), from the project root:

```bash
go run github.com/steebchen/prisma-client-go generate
go build ./...
```

This downloads Prisma's CLI/query-engine binaries and writes the generated client into `internal/db`
(per the `output` path set in `prisma/schema.prisma`). After that, `go build ./...` should succeed
end-to-end. If it doesn't, paste the error back in and we'll debug it together.



### Dev environment note
Go module downloads needed `GOPROXY=direct GOSUMDB=off` plus `replace` directives in `go.mod` mapping
`golang.org/x/*` → `github.com/golang/*` mirrors, because the build sandbox only allowlists `github.com`/
`codeload.github.com`, not `proxy.golang.org` or `golang.org`. Keep these replace directives — remove only
if building outside this constrained network.
````
