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
