package worker

import (
	"context"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/yourorg/urlshortener/internal/metrics"
	"github.com/yourorg/urlshortener/internal/redis"
	"github.com/yourorg/urlshortener/internal/store"
)

// AnalyticsWorker consumes click events from a Redis Stream and writes
// them to Postgres. It runs as a background goroutine alongside the API
// server, keeping the redirect hot path free of DB writes.
type AnalyticsWorker struct {
	Redis        *redis.Client
	Store        store.ClickEventStore
	ConsumerName string
	BatchSize    int64
	BlockTime    time.Duration
}

// New creates an AnalyticsWorker with sensible defaults.
func New(rdb *redis.Client, s store.ClickEventStore) *AnalyticsWorker {
	return &AnalyticsWorker{
		Redis:        rdb,
		Store:        s,
		ConsumerName: "worker-1",
		BatchSize:    100,
		BlockTime:    5 * time.Second,
	}
}

// Start launches the worker loop in a background goroutine. It runs
// until the provided context is cancelled (graceful shutdown).
func (w *AnalyticsWorker) Start(ctx context.Context) {
	log.Printf("analytics worker started (consumer=%s, batch=%d, block=%s)",
		w.ConsumerName, w.BatchSize, w.BlockTime)

	go w.run(ctx)
}

func (w *AnalyticsWorker) run(ctx context.Context) {
	for {
		// Check if we should stop.
		if ctx.Err() != nil {
			log.Println("analytics worker stopping: context cancelled")
			return
		}

		// Read a batch of click events from the Redis Stream.
		// This blocks for up to BlockTime waiting for new events.
		ids, events, err := w.Redis.ReadClickEvents(ctx, w.ConsumerName, w.BatchSize, w.BlockTime)
		if err != nil {
			if ctx.Err() != nil {
				// Context was cancelled while blocking — shut down.
				return
			}
			// redis.Nil is returned by XReadGroup when the block timeout
			// expires with no new events — this is normal, not an error.
			if err == goredis.Nil {
				continue
			}
			// Actual error — log and back off briefly.
			log.Printf("analytics worker: ReadClickEvents error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if len(events) == 0 {
			// No events in this batch — loop back and block again.
			continue
		}

		// Write each event to Postgres. We write individually (not in
		// a single bulk INSERT) because Prisma's Go client doesn't
		// expose a bulk-create API. This is fine for analytics — the
		// worker is off the hot path and can take its time.
		var ackIDs []string
		for i, ev := range events {
			clickEvent := &store.ClickEvent{
				LinkID:     ev.LinkID,
				Timestamp:  time.Unix(0, ev.Timestamp),
				Referrer:   ev.Referrer,
				Country:    ev.Country,
				DeviceType: ev.DeviceType,
				IPHash:     ev.IPHash,
			}

			start := time.Now()
			if err := w.Store.CreateClickEvent(ctx, clickEvent); err != nil {
				// If a single event fails to write, we skip it and
				// don't ACK it — it stays in the pending entries list
				// and can be re-processed later. We DO continue
				// processing the rest of the batch.
				metrics.AnalyticsEventsFailed.Inc()
				log.Printf("analytics worker: failed to write click event (stream id=%s): %v", ids[i], err)
				continue
			}
			metrics.AnalyticsEventsProcessed.Inc()
			metrics.AnalyticsProcessingDuration.Observe(time.Since(start).Seconds())
			ackIDs = append(ackIDs, ids[i])
		}

		// Acknowledge all successfully written events so they're
		// removed from the consumer group's pending list.
		if len(ackIDs) > 0 {
			if err := w.Redis.AckClickEvents(ctx, ackIDs...); err != nil {
				log.Printf("analytics worker: failed to ACK %d events: %v", len(ackIDs), err)
			}
		}
	}
}