package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// StreamKey is the Redis Stream that carries click events from the
// redirect hot path to the analytics worker.
const StreamKey = "urlshortener:click_events"

// ConsumerGroup is the Redis consumer group name used by the analytics
// worker. Using a consumer group (instead of plain XREAD) lets us:
//   - scale horizontally (multiple workers share the group)
//   - survive crashes (pending entries are re-delivered to another consumer)
//   - track which events have been processed (XACK)
const ConsumerGroup = "analytics-workers"

// ClickEvent is the payload pushed onto the Redis Stream by the redirect
// handler and consumed by the analytics worker. It mirrors the fields of
// the ClickEvent Prisma model, but is kept as a plain struct so the
// redirect handler doesn't need to import store types (keeping the hot
// path dependency-light).
type ClickEvent struct {
	LinkID     uint64 `json:"link_id"`
	Timestamp  int64  `json:"timestamp"` // Unix nanoseconds
	Referrer   string `json:"referrer,omitempty"`
	Country    string `json:"country,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	IPHash     string `json:"ip_hash,omitempty"`
}

// PushClickEvent enqueues a click event onto the Redis Stream. It is
// designed to be called in a fire-and-forget manner from the redirect
// handler — errors are returned but the caller is expected to ignore
// them (analytics is best-effort and must never block or fail a redirect).
func (c *Client) PushClickEvent(ctx context.Context, ev ClickEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	// XADD with MAXLEN ~ trims the stream so it doesn't grow unbounded
	// if the worker falls behind or is down for a long time. The exact
	// cap (100k) is a safety net, not a normal operating condition —
	// the worker should drain the stream far below this in practice.
	return c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		MaxLen: 100000,
		Approx: true, // use ~ for efficiency (trims in batches)
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}

// HashIP hashes a raw IP address for privacy. The hash is one-way
// (SHA-256) so the original IP can't be recovered from the stored
// analytics data. A salt could be added later for stronger protection.
func HashIP(ip string) string {
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])[:16] // first 16 hex chars (64 bits) is enough for dedup
}

// ReadClickEvents reads up to `count` click events from the stream using
// a consumer group. It blocks for up to `block` duration waiting for new
// events if none are immediately available. Returns the raw stream IDs
// (needed for XACK later) and the decoded events.
func (c *Client) ReadClickEvents(ctx context.Context, consumerName string, count int64, block time.Duration) (ids []string, events []ClickEvent, err error) {
	// Ensure the consumer group exists. XGroupCreateMkStream creates
	// the stream if it doesn't exist yet (first ever event). This
	// prevents the NOGROUP error that would otherwise occur on every
	// poll before the first click event is ever pushed.
	_ = c.rdb.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "$").Err()

	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{StreamKey, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, nil, err
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			raw, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}
			var ev ClickEvent
			if json.Unmarshal([]byte(raw), &ev) == nil {
				ids = append(ids, msg.ID)
				events = append(events, ev)
			}
		}
	}
	return ids, events, nil
}

// AckClickEvents acknowledges that the given stream IDs have been
// processed (written to Postgres). This removes them from the
// consumer group's pending entries list.
func (c *Client) AckClickEvents(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return c.rdb.XAck(ctx, StreamKey, ConsumerGroup, ids...).Err()
}

// StreamLen returns the current number of entries in the click events
// stream. Useful for monitoring / health checks.
func (c *Client) StreamLen(ctx context.Context) (int64, error) {
	return c.rdb.XLen(ctx, StreamKey).Result()
}