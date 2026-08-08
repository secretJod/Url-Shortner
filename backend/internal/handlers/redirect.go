package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/metrics"
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
	longURL, linkID, err := h.Redis.GetLongURL(ctx, shortCode)
	if err == nil {
		// Fire-and-forget click event for analytics (Phase 3).
		// linkID may be 0 for old cache entries — skip if so.
		if linkID > 0 {
			h.fireClickEvent(ctx, linkID, c)
		}
		metrics.RedirectsTotal.WithLabelValues("success").Inc()
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
			metrics.RedirectsTotal.WithLabelValues("not_found").Inc()
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found or expired"})
		}
		metrics.RedirectsTotal.WithLabelValues("error").Inc()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to resolve short link"})
	}

	// 3. Backfill the cache so the next request for this code is a hit.
	_ = h.Redis.SetLongURL(ctx, link.ShortCode, link.LongURL, link.ID, link.ExpiresAt)

	// 4. Fire-and-forget click event for analytics (Phase 3).
	h.fireClickEvent(ctx, link.ID, c)

	metrics.RedirectsTotal.WithLabelValues("success").Inc()
	return c.Redirect(link.LongURL, fiber.StatusFound)
}

// fireClickEvent pushes a click event onto the Redis Stream for the
// analytics worker to consume. It is strictly fire-and-forget:
// errors are ignored, and the redirect is never blocked or failed
// due to analytics issues.
func (h *RedirectHandler) fireClickEvent(ctx context.Context, linkID uint64, c *fiber.Ctx) {
	ev := redis.ClickEvent{
		LinkID:    linkID,
		Timestamp: time.Now().UnixNano(),
		Referrer:  c.Get("Referer"),
		IPHash:    redis.HashIP(c.IP()),
	}
	// Ignore error — analytics is best-effort.
	_ = h.Redis.PushClickEvent(ctx, ev)
}