package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/metrics"
)

// MetricsMiddleware records request count, status code, and latency
// for every request. Must be registered AFTER recover and logger.
func MetricsMiddleware(m *metrics.Metrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Record metrics
		latency := time.Since(start)
		m.RecordRequest(c.Response().StatusCode(), latency)

		return err
	}
}