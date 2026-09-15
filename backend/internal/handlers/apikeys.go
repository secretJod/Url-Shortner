package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/auth"
	appmail "github.com/yourorg/urlshortener/internal/mail"
	"github.com/yourorg/urlshortener/internal/store"
)

// verificationTokenTTL bounds how long a magic link is valid for (SEC-01).
const verificationTokenTTL = 15 * time.Minute

type apiKeyStore interface {
	store.UserStore
	store.ApiKeyStore
	store.VerificationTokenStore
}

type APIKeyHandler struct {
	Store   apiKeyStore
	Mailer  appmail.Mailer
	BaseURL string
}

func NewAPIKeyHandler(s apiKeyStore, mailer appmail.Mailer, baseURL string) *APIKeyHandler {
	return &APIKeyHandler{Store: s, Mailer: mailer, BaseURL: baseURL}
}

type createAPIKeyRequest struct {
	Email string `json:"email"`
}

type createAPIKeyResponse struct {
	APIKey  string `json:"api_key"`
	Warning string `json:"warning"`
}

// CreateKey starts the email-verification flow (SEC-01): given an email,
// it resolves/creates the user, mints a one-time verification token, and
// emails a magic link. No API key is issued here anymore — that only
// happens once the link is clicked (see VerifyKey below).
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

	rawToken, hash, err := generateToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate verification token"})
	}

	vt := &store.VerificationToken{
		TokenHash: hash,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(verificationTokenTTL),
	}
	if err := h.Store.CreateVerificationToken(ctx, vt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create verification token"})
	}

	link := fmt.Sprintf("%s/api/keys/verify?token=%s", h.BaseURL, rawToken)
	body := fmt.Sprintf("Click the link below to verify your email and get your API key. This link expires in %d minutes.\n\n%s", int(verificationTokenTTL.Minutes()), link)

	// Best-effort send: analytics/mail failures shouldn't be silently
	// swallowed here since the user has no other way to get their key,
	// but they also shouldn't leak whether the email address exists.
	if err := h.Mailer.Send(req.Email, "Verify your email", body); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to send verification email, please try again"})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"message": "verification email sent"})
}

// VerifyKey consumes a verification token from the magic link, marks the
// user verified, and issues (and returns) their API key.
// GET /api/keys/verify?token=...
func (h *APIKeyHandler) VerifyKey(c *fiber.Ctx) error {
	rawToken := c.Query("token")
	if rawToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token is required"})
	}

	ctx := c.Context()
	hash := hashToken(rawToken)

	vt, err := h.Store.GetVerificationTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or already-used token"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to validate token"})
	}

	if vt.ExpiresAt.Before(time.Now()) {
		_ = h.Store.DeleteVerificationToken(ctx, vt.ID)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token expired, please request a new one"})
	}

	// Consume immediately so the token can't be replayed.
	if err := h.Store.DeleteVerificationToken(ctx, vt.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to consume token"})
	}

	if err := h.Store.MarkUserVerified(ctx, vt.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to verify user"})
	}

	rawKey, keyHash, err := auth.GenerateAPIKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate API key"})
	}

	apiKey := &store.ApiKey{
		KeyHash:       keyHash,
		UserID:        vt.UserID,
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

// generateToken creates a new random verification token. Returns the raw
// token (goes in the emailed link, never persisted) and its hash (stored).
func generateToken() (rawToken string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(buf)
	hash = hashToken(rawToken)
	return rawToken, hash, nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
