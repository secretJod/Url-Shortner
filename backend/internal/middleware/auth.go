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
