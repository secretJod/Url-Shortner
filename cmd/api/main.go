package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/yourorg/urlshortener/internal/config"
	"github.com/yourorg/urlshortener/internal/db"
	"github.com/yourorg/urlshortener/internal/handlers"
	"github.com/yourorg/urlshortener/internal/middleware"
	"github.com/yourorg/urlshortener/internal/redis"
)

func main() {
	cfg := config.Load()

	rdb := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer rdb.Close()

	linkStore, err := db.NewPrismaStore()
	if err != nil {
		log.Fatalf("failed to connect to Postgres via Prisma: %v", err)
	}
	defer linkStore.Close()

	app := fiber.New(fiber.Config{
		AppName:      "urlshortener",
		ServerHeader: "urlshortener",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	health := handlers.NewHealthHandler(rdb)
	app.Get("/health", health.Check)

	// Auth is optional (anonymous requests still work) but must run before
	// rate limiting, since the limiter needs to know whether a request is
	// authenticated to pick the right bucket/limit.
	authMW := middleware.OptionalAPIKeyAuth(linkStore)
	rateLimitMW := middleware.RateLimit(rdb)

	apiKeys := handlers.NewAPIKeyHandler(linkStore)
	app.Post("/api/keys", rateLimitMW, apiKeys.CreateKey)

	shorten := handlers.NewShortenHandler(linkStore, rdb, cfg.BaseURL)
	app.Post("/api/shorten", authMW, rateLimitMW, shorten.Shorten)

	redirect := handlers.NewRedirectHandler(linkStore, rdb)
	app.Get("/:shortCode", redirect.Redirect)

	log.Printf("starting urlshortener on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
