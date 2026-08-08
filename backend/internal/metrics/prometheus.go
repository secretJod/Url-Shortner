package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the URL shortener.
// These are registered exactly once via promauto (safe for multiple calls).
var (
	// HTTP request counter: method, route path, status code.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "urlshortener_http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP request duration histogram: method, route path.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "urlshortener_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Redirects counter: result (success, not_found, error).
	RedirectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "urlshortener_redirects_total",
			Help: "Total number of redirect attempts by result.",
		},
		[]string{"result"},
	)

	// Rate limit decisions counter: result (allowed, limited).
	RateLimitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "urlshortener_rate_limit_total",
			Help: "Total number of rate-limit decisions by result.",
		},
		[]string{"result"},
	)

	// Analytics worker events processed counter.
	AnalyticsEventsProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "urlshortener_analytics_events_processed_total",
			Help: "Total number of analytics events successfully processed.",
		},
	)

	// Analytics worker events failed counter.
	AnalyticsEventsFailed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "urlshortener_analytics_events_failed_total",
			Help: "Total number of analytics events that failed to process.",
		},
	)

	// Analytics worker processing duration histogram.
	AnalyticsProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "urlshortener_analytics_processing_duration_seconds",
			Help:    "Time spent processing analytics events in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
)