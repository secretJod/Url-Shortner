package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/yourorg/urlshortener/internal/config"
	"github.com/yourorg/urlshortener/internal/db"
	"github.com/yourorg/urlshortener/internal/handlers"
	"github.com/yourorg/urlshortener/internal/metrics"
	"github.com/yourorg/urlshortener/internal/middleware"
	"github.com/yourorg/urlshortener/internal/redis"
	"github.com/yourorg/urlshortener/internal/worker"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	rdb := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer rdb.Close()

	linkStore, err := db.NewPrismaStore()
	if err != nil {
		log.Fatalf("failed to connect to Postgres via Prisma: %v", err)
	}
	defer linkStore.Close()

	// Start the analytics worker (Phase 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	analyticsWorker := worker.New(rdb, linkStore)
	analyticsWorker.Start(ctx)

	app := fiber.New(fiber.Config{
		AppName:      "urlshortener",
		ServerHeader: "urlshortener",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	// Phase 5: Metrics middleware
	appMetrics := metrics.New()
	app.Use(middleware.MetricsMiddleware(appMetrics))

	health := handlers.NewHealthHandler(rdb)
	app.Get("/health", health.Check)

	// Phase 5: Metrics endpoint
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.JSON(appMetrics.Snapshot())
	})

	// Auth + rate limit middleware
	authMW := middleware.OptionalAPIKeyAuth(linkStore)
	rateLimitMW := middleware.RateLimit(rdb)

	apiKeys := handlers.NewAPIKeyHandler(linkStore)
	app.Post("/api/keys", rateLimitMW, apiKeys.CreateKey)

	shorten := handlers.NewShortenHandler(linkStore, rdb, cfg.BaseURL)
	app.Post("/api/shorten", authMW, rateLimitMW, shorten.Shorten)

	// Phase 4: Admin/Stats API (registered BEFORE /:shortCode)
	stats := handlers.NewStatsHandler(linkStore)
	app.Get("/api/stats/top", rateLimitMW, stats.GetTopLinks)
	app.Get("/api/stats/:shortCode", rateLimitMW, stats.GetLinkStats)
	app.Get("/api/links", authMW, rateLimitMW, stats.GetUserLinks)
	app.Get("/api/links/:shortCode/clicks", rateLimitMW, stats.GetRecentClicks)

	// Redirect route (registered LAST so it doesn't catch /metrics, /api/*, etc.)
	redirect := handlers.NewRedirectHandler(linkStore, rdb)
	app.Get("/:shortCode", redirect.Redirect)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutdown signal received, stopping...")
		cancel()
		_ = app.Shutdown()
	}()

	log.Printf("starting urlshortener on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}