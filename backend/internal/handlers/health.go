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
