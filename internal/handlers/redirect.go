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
