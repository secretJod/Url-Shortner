package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks application-level metrics using atomic counters.
// No external dependencies — simple and portable.
// Exposed via the /metrics endpoint as JSON.
type Metrics struct {
	TotalRequests    atomic.Int64
	RequestsByStatus map[int64]*atomic.Int64
	statusMu         sync.RWMutex
	totalLatencyUs   atomic.Int64
	maxLatencyUs     atomic.Int64
	CacheHits        atomic.Int64
	CacheMisses      atomic.Int64
	EventsProcessed  atomic.Int64
	EventsFailed     atomic.Int64
	StartTime        time.Time
}

func New() *Metrics {
	return &Metrics{
		StartTime:        time.Now(),
		RequestsByStatus: make(map[int64]*atomic.Int64),
	}
}

func (m *Metrics) RecordRequest(statusCode int, latency time.Duration) {
	m.TotalRequests.Add(1)
	m.statusMu.RLock()
	counter, ok := m.RequestsByStatus[int64(statusCode)]
	m.statusMu.RUnlock()
	if !ok {
		m.statusMu.Lock()
		counter, ok = m.RequestsByStatus[int64(statusCode)]
		if !ok {
			counter = &atomic.Int64{}
			m.RequestsByStatus[int64(statusCode)] = counter
		}
		m.statusMu.Unlock()
	}
	counter.Add(1)
	us := latency.Microseconds()
	m.totalLatencyUs.Add(us)
	for {
		old := m.maxLatencyUs.Load()
		if us <= old || m.maxLatencyUs.CompareAndSwap(old, us) {
			break
		}
	}
}

func (m *Metrics) RecordCacheHit()        { m.CacheHits.Add(1) }
func (m *Metrics) RecordCacheMiss()       { m.CacheMisses.Add(1) }
func (m *Metrics) RecordEventProcessed()  { m.EventsProcessed.Add(1) }
func (m *Metrics) RecordEventFailed()     { m.EventsFailed.Add(1) }

func (m *Metrics) Snapshot() map[string]interface{} {
	total := m.TotalRequests.Load()
	hits := m.CacheHits.Load()
	misses := m.CacheMisses.Load()
	var cacheHitRate float64
	if hits+misses > 0 {
		cacheHitRate = float64(hits) / float64(hits+misses) * 100
	}
	var avgLatencyMs float64
	if total > 0 {
		avgLatencyMs = float64(m.totalLatencyUs.Load()) / float64(total) / 1000.0
	}
	statusCounts := make(map[string]int64)
	m.statusMu.RLock()
	for code, counter := range m.RequestsByStatus {
		statusCounts[fmt.Sprintf("%d", code)] = counter.Load()
	}
	m.statusMu.RUnlock()
	return map[string]interface{}{
		"uptime_seconds":   int(time.Since(m.StartTime).Seconds()),
		"total_requests":   total,
		"status_codes":     statusCounts,
		"avg_latency_ms":   avgLatencyMs,
		"max_latency_ms":   float64(m.maxLatencyUs.Load()) / 1000.0,
		"cache_hits":       hits,
		"cache_misses":     misses,
		"cache_hit_rate":   cacheHitRate,
		"events_processed": m.EventsProcessed.Load(),
		"events_failed":    m.EventsFailed.Load(),
	}
}