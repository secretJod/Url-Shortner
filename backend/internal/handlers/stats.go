package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/middleware"
	"github.com/yourorg/urlshortener/internal/redis"
	"github.com/yourorg/urlshortener/internal/store"
)

// StatsHandler provides admin and analytics endpoints (Phase 4).
type StatsHandler struct {
	Store store.StatsStore
	Redis *redis.Client
}

func NewStatsHandler(s store.StatsStore, r *redis.Client) *StatsHandler {
	return &StatsHandler{Store: s, Redis: r}
}

// getLinkByShortCode is a helper that gets a link from the stats store.
// It uses type assertion to access LinkStore methods since PrismaStore
// implements both interfaces.
func getLinkByShortCode(s store.StatsStore, ctx context.Context, shortCode string) (*store.Link, error) {
	if ls, ok := s.(store.LinkStore); ok {
		return ls.GetLinkByShortCode(ctx, shortCode)
	}
	return nil, store.ErrNotFound
}

// ownsLink returns true if the authenticated API key's user owns the link.
// Links with no owner (anonymous-created) are never "owned" by anyone.
func ownsLink(link *store.Link, apiKey *store.ApiKey) bool {
	return link.UserID != nil && apiKey != nil && *link.UserID == apiKey.UserID
}

// GetLinkStats returns aggregated click analytics for a single link.
// Requires the authenticated caller to own the link (SEC-02) — an
// unauthenticated or non-owning request gets 404, same as a link that
// doesn't exist, to avoid confirming which short codes exist.
// GET /api/stats/:shortCode
func (h *StatsHandler) GetLinkStats(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	link, err := getLinkByShortCode(h.Store, c.Context(), shortCode)
	if err != nil {
		if err == store.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to find link"})
	}

	apiKey := middleware.GetAPIKey(c)
	if !ownsLink(link, apiKey) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
	}

	stats, err := h.Store.GetLinkStats(c.Context(), shortCode)
	if err != nil {
		if err == store.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch link stats"})
	}

	// Convert DailyClicks to JSON-friendly format.
	daily := make([]fiber.Map, 0, len(stats.DailyClicks))
	for _, dc := range stats.DailyClicks {
		daily = append(daily, fiber.Map{
			"date":  dc.Date.Format("2006-01-02"),
			"count": dc.Count,
		})
	}

	return c.JSON(fiber.Map{
		"short_code":    stats.Link.ShortCode,
		"long_url":      stats.Link.LongURL,
		"total_clicks":  stats.TotalClicks,
		"unique_ips":    stats.UniqueIPs,
		"top_referrers": stats.ReferrerTop,
		"daily_clicks":  daily,
	})
}

// GetTopLinks returns the most-clicked links. Public: aggregate-only data,
// no per-user detail, so it stays open (SEC-02 doesn't apply here).
// GET /api/stats/top?limit=N
func (h *StatsHandler) GetTopLinks(c *fiber.Ctx) error {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	topLinks, err := h.Store.GetTopLinks(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch top links"})
	}

	results := make([]fiber.Map, 0, len(topLinks))
	for _, ls := range topLinks {
		results = append(results, fiber.Map{
			"short_code":   ls.Link.ShortCode,
			"long_url":     ls.Link.LongURL,
			"total_clicks": ls.TotalClicks,
		})
	}

	return c.JSON(fiber.Map{"top_links": results})
}

// GetUserLinks returns all links created by the authenticated user.
// GET /api/links
func (h *StatsHandler) GetUserLinks(c *fiber.Ctx) error {
	apiKey := middleware.GetAPIKey(c)
	if apiKey == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "API key required"})
	}

	links, err := h.Store.GetUserLinks(c.Context(), apiKey.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch user links"})
	}

	results := make([]fiber.Map, 0, len(links))
	for _, link := range links {
		results = append(results, fiber.Map{
			"short_code":   link.ShortCode,
			"long_url":     link.LongURL,
			"custom_alias": link.CustomAlias,
			"created_at":   link.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"links": results})
}

// GetRecentClicks returns the most recent click events for a link.
// Requires the authenticated caller to own the link (SEC-02).
// GET /api/links/:shortCode/clicks?limit=N
func (h *StatsHandler) GetRecentClicks(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	// First get the link to find its ID.
	link, err := getLinkByShortCode(h.Store, c.Context(), shortCode)
	if err != nil {
		if err == store.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to find link"})
	}

	apiKey := middleware.GetAPIKey(c)
	if !ownsLink(link, apiKey) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := h.Store.GetRecentClickEvents(c.Context(), link.ID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch click events"})
	}

	results := make([]fiber.Map, 0, len(events))
	for _, ev := range events {
		results = append(results, fiber.Map{
			"timestamp":   ev.Timestamp,
			"referrer":    ev.Referrer,
			"country":     ev.Country,
			"device_type": ev.DeviceType,
			"ip_hash":     ev.IPHash,
		})
	}

	return c.JSON(fiber.Map{"clicks": results})
}

// DeleteLink deletes a link owned by the authenticated caller and
// invalidates its cache entry.
// DELETE /api/links/:shortCode
func (h *StatsHandler) DeleteLink(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	link, err := getLinkByShortCode(h.Store, c.Context(), shortCode)
	if err != nil {
		if err == store.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to find link"})
	}

	apiKey := middleware.GetAPIKey(c)
	if !ownsLink(link, apiKey) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
	}

	ls, ok := h.Store.(store.LinkStore)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete not supported"})
	}
	if err := ls.DeleteLink(c.Context(), shortCode); err != nil {
		if err == store.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete link"})
	}

	if h.Redis != nil {
		_ = h.Redis.InvalidateLongURL(c.Context(), shortCode)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
