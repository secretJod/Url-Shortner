package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourorg/urlshortener/internal/redis"
)

// pinger is satisfied by store.Store (and PrismaStore), kept narrow here
// so the health handler doesn't need to import the store package for a
// single method.
type pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	Redis *redis.Client
	DB    pinger
}

func NewHealthHandler(r *redis.Client, db pinger) *HealthHandler {
	return &HealthHandler{Redis: r, DB: db}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := "ok"
	code := fiber.StatusOK
	redisStatus := "ok"
	dbStatus := "ok"

	if err := h.Redis.Ping(ctx); err != nil {
		redisStatus = "unreachable: " + err.Error()
		status = "degraded"
		code = fiber.StatusServiceUnavailable
	}

	if h.DB != nil {
		if err := h.DB.Ping(ctx); err != nil {
			dbStatus = "unreachable: " + err.Error()
			status = "degraded"
			code = fiber.StatusServiceUnavailable
		}
	}

	return c.Status(code).JSON(fiber.Map{
		"status":   status,
		"redis":    redisStatus,
		"database": dbStatus,
	})
}
