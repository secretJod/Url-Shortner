package redis

import (
	"context"
	"testing"
	"time"
)

func TestPushAndReadClickEvent(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	// Clean up any leftover stream + consumer group from previous runs.
	c.rdb.Del(ctx, StreamKey)

	ev := ClickEvent{
		LinkID:    42,
		Timestamp: time.Now().UnixNano(),
		Referrer:  "https://google.com",
		IPHash:    "abc123def456",
	}

	if err := c.PushClickEvent(ctx, ev); err != nil {
		t.Fatalf("PushClickEvent() error: %v", err)
	}

	// Read it back — use a short block since the event is already in the stream.
	ids, events, err := c.ReadClickEvents(ctx, "test-consumer", 10, 1*time.Second)
	if err != nil {
		t.Fatalf("ReadClickEvents() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].LinkID != ev.LinkID {
		t.Errorf("LinkID mismatch: got %d, want %d", events[0].LinkID, ev.LinkID)
	}
	if events[0].Referrer != ev.Referrer {
		t.Errorf("Referrer mismatch: got %q, want %q", events[0].Referrer, ev.Referrer)
	}
	if events[0].IPHash != ev.IPHash {
		t.Errorf("IPHash mismatch: got %q, want %q", events[0].IPHash, ev.IPHash)
	}

	// Ack the event so it doesn't stay in the pending list.
	if err := c.AckClickEvents(ctx, ids...); err != nil {
		t.Fatalf("AckClickEvents() error: %v", err)
	}

	// Clean up.
	c.rdb.Del(ctx, StreamKey)
}

func TestHashIP(t *testing.T) {
	// Empty IP returns empty hash.
	if got := HashIP(""); got != "" {
		t.Errorf("HashIP(\"\") = %q, want \"\"", got)
	}

	// Same IP always produces the same hash (deterministic).
	ip := "192.168.1.1"
	h1 := HashIP(ip)
	h2 := HashIP(ip)
	if h1 != h2 {
		t.Errorf("HashIP is not deterministic: %q != %q", h1, h2)
	}

	// Different IPs produce different hashes.
	h3 := HashIP("10.0.0.1")
	if h1 == h3 {
		t.Errorf("different IPs produced the same hash: %q", h1)
	}

	// Hash is 16 hex characters (64 bits).
	if len(h1) != 16 {
		t.Errorf("hash length = %d, want 16", len(h1))
	}
}

func TestStreamLen(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	// Clean up.
	c.rdb.Del(ctx, StreamKey)

	// Empty stream has length 0.
	length, err := c.StreamLen(ctx)
	if err != nil {
		t.Fatalf("StreamLen() error: %v", err)
	}
	if length != 0 {
		t.Errorf("expected length 0, got %d", length)
	}

	// Push 3 events.
	for i := 0; i < 3; i++ {
		if err := c.PushClickEvent(ctx, ClickEvent{
			LinkID:    uint64(i + 1),
			Timestamp: time.Now().UnixNano(),
		}); err != nil {
			t.Fatalf("PushClickEvent() error: %v", err)
		}
	}

	length, err = c.StreamLen(ctx)
	if err != nil {
		t.Fatalf("StreamLen() error: %v", err)
	}
	if length != 3 {
		t.Errorf("expected length 3, got %d", length)
	}

	// Clean up.
	c.rdb.Del(ctx, StreamKey)
}