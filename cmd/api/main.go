package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/yourorg/urlshortener/internal/config"
	"github.com/yourorg/urlshortener/internal/handlers"
	"github.com/yourorg/urlshortener/internal/redis"
)

func main() {
	cfg := config.Load()

	rdb := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer rdb.Close()

	app := fiber.New(fiber.Config{
		AppName:      "urlshortener",
		ServerHeader: "urlshortener",
	})

	app.Use(recover.New())
	app.Use(logger.New())

	health := handlers.NewHealthHandler(rdb)
	app.Get("/health", health.Check)

	// Placeholder routes — wired up properly in Phase 1
	app.Post("/api/shorten", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"message": "coming in Phase 1",
		})
	})
	app.Get("/:shortCode", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"message": "coming in Phase 1",
		})
	})

	log.Printf("starting urlshortener on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
