package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yourorg/urlshortener/internal/metrics"
)

// PrometheusMetricsMiddleware records HTTP request count and duration
// using the registered route path (e.g. "/:shortCode") as the path label,
// avoiding high-cardinality from raw request URLs.
// This is additive — it does not change request behavior.
func PrometheusMetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Resolve the registered route path (not the raw URL)
		routePath := c.Route().Path
		if routePath == "" {
			routePath = c.Path()
		}
		method := c.Method()
		status := strconv.Itoa(c.Response().StatusCode())

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(method, routePath, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, routePath).Observe(time.Since(start).Seconds())

		return err
	}
}