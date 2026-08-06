package redis

import (
	"context"
	"testing"
	"time"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	c := New("localhost:6379", "", 0)
	if err := c.Ping(context.Background()); err != nil {
		t.Skipf("redis not reachable, skipping: %v", err)
	}
	return c
}

func TestAllow_UnderLimit(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:under:" + randSuffix()

	for i := 0; i < 5; i++ {
		allowed, _, err := c.Allow(ctx, key, 5, time.Minute)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=5)", i+1)
		}
	}
}

func TestAllow_OverLimit(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:over:" + randSuffix()

	for i := 0; i < 3; i++ {
		allowed, _, err := c.Allow(ctx, key, 3, time.Minute)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=3)", i+1)
		}
	}

	// 4th request should be blocked.
	allowed, retryAfter, err := c.Allow(ctx, key, 3, time.Minute)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if allowed {
		t.Fatal("4th request should have been blocked (limit=3)")
	}
	if retryAfter <= 0 {
		t.Errorf("expected positive retryAfter, got %v", retryAfter)
	}
}

func TestAllow_WindowSlides(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test:ratelimit:slide:" + randSuffix()

	shortWindow := 500 * time.Millisecond

	for i := 0; i < 2; i++ {
		allowed, _, err := c.Allow(ctx, key, 2, shortWindow)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed (limit=2)", i+1)
		}
	}

	allowed, _, _ := c.Allow(ctx, key, 2, shortWindow)
	if allowed {
		t.Fatal("3rd request should have been blocked immediately")
	}

	time.Sleep(shortWindow + 100*time.Millisecond)

	allowed, _, err := c.Allow(ctx, key, 2, shortWindow)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Fatal("request after window expiry should have been allowed")
	}
}
