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
