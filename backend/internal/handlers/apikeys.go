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
