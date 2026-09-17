package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yourorg/urlshortener/internal/config"
	"github.com/yourorg/urlshortener/internal/db"
	"github.com/yourorg/urlshortener/internal/handlers"
	"github.com/yourorg/urlshortener/internal/mail"
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

	// CORS — origins are env-driven (CORS_ALLOWED_ORIGINS) rather than
	// hardcoded, so prod deployments can lock this down without a code change.
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Phase 5: Metrics middleware (atomic counters — kept for backward compat)
	appMetrics := metrics.New()
	app.Use(middleware.MetricsMiddleware(appMetrics))

	// Prometheus metrics middleware (additive — records HTTP metrics with route paths)
	app.Use(middleware.PrometheusMetricsMiddleware())

	health := handlers.NewHealthHandler(rdb, linkStore)
	app.Get("/health", health.Check)

	// Prometheus /metrics endpoint — standard Prometheus exposition format.
	// Uses the well-tested promhttp.Handler() (adapted for Fiber) instead of a
	// hand-rolled encoder — this ensures a correct Content-Type
	// ("text/plain; version=0.0.4; charset=utf-8") that Prometheus requires.
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// JSON metrics endpoint for the admin dashboard (Prometheus /metrics above
	// stays in the standard exposition format for scrapers).
	app.Get("/api/metrics", func(c *fiber.Ctx) error {
		return c.JSON(appMetrics.Snapshot())
	})

	// Auth + rate limit middleware
	authMW := middleware.OptionalAPIKeyAuth(linkStore)
	requireAuthMW := middleware.RequireAPIKeyAuth(linkStore)
	rateLimitMW := middleware.RateLimit(rdb)

	mailer := mail.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.MailFrom)
	apiKeys := handlers.NewAPIKeyHandler(linkStore, mailer, cfg.BaseURL)
	app.Post("/api/keys", rateLimitMW, apiKeys.CreateKey)
	app.Get("/api/keys/verify", rateLimitMW, apiKeys.VerifyKey)

	shorten := handlers.NewShortenHandler(linkStore, rdb, cfg.BaseURL)
	app.Post("/api/shorten", authMW, rateLimitMW, shorten.Shorten)

	// Phase 4: Admin/Stats API (registered BEFORE /:shortCode)
	// SEC-02: /api/stats/:shortCode and /api/links/:shortCode/clicks require
	// an authenticated key AND ownership of the link (enforced in the
	// handlers) — /api/stats/top stays public since it's aggregate-only.
	stats := handlers.NewStatsHandler(linkStore, rdb)
	app.Get("/api/stats/top", rateLimitMW, stats.GetTopLinks)
	app.Get("/api/stats/:shortCode", requireAuthMW, rateLimitMW, stats.GetLinkStats)
	app.Get("/api/links", authMW, rateLimitMW, stats.GetUserLinks)
	app.Get("/api/links/:shortCode/clicks", requireAuthMW, rateLimitMW, stats.GetRecentClicks)
	app.Delete("/api/links/:shortCode", requireAuthMW, rateLimitMW, stats.DeleteLink)

	// Phase 6: Serve frontend static files (if frontend/dist exists)
	app.Static("/", "./frontend/dist")

	// Phase 6: SPA routes — serve index.html for these frontend paths
	serveIndex := func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/dist/index.html")
	}
	app.Get("/", serveIndex)
	app.Get("/login", serveIndex)
	app.Get("/dashboard", serveIndex)
	app.Get("/stats/*", serveIndex)
	app.Get("/top", serveIndex)
	app.Get("/admin", serveIndex)

	// Redirect route for short codes — fallback to SPA index.html if not a valid short code.
	// SEC-06: rate-limited per-IP (anonymous, since redirects carry no API key)
	// to blunt scraping/enumeration of short codes.
	// NOTE: RedirectHandler.Redirect handles the not-found case internally
	// (it writes a JSON 404 and returns nil), and the frontend has no
	// dedicated "not found" route/page to serve instead — so there is no
	// SPA page worth falling back to here. We just return the handler's
	// result (redirect on success, JSON 404/500 on failure) as-is.
	redirect := handlers.NewRedirectHandler(linkStore, rdb, cfg.IPHashSecret)
	app.Get("/:shortCode", rateLimitMW, redirect.Redirect)

	// Catch-all 404 — registered after every other route (including
	// /:shortCode) so that any request that reaches here matches a real,
	// bounded Fiber route ("/*") instead of Fiber's synthetic not-found
	// route, whose Path is the raw request URL. Without this, the
	// Prometheus middleware's `path` label would be unbounded for
	// probes/scanners hitting arbitrary paths.
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	})

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
