package redis

import (
	"context"
	"fmt"
	"time"
)

// Allow implements a sliding-window-log rate limiter using a Redis sorted
// set: each request is recorded as a member scored by its timestamp: old
// entries outside the window are trimmed, then the remaining count is
// compared against limit. This is more accurate than fixed-window counters
// (no burst-at-the-boundary problem) at the cost of O(log N) per request
// instead of O(1) — an acceptable tradeoff at the request rates a rate
// limiter itself needs to handle.
//
// key should already be scoped to the thing being limited, e.g.
// "ratelimit:apikey:123" or "ratelimit:ip:1.2.3.4".
func (c *Client) Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error) {
	now := time.Now()
	windowStart := now.Add(-window)
	member := fmt.Sprintf("%d-%s", now.UnixNano(), randSuffix())

	pipe := c.rdb.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redisZ(float64(now.UnixNano()), member))
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, err
	}

	count := countCmd.Val()
	if int(count) >= limit {
		// Over limit — remove the entry we just optimistically added, since
		// this request shouldn't count against the window.
		c.rdb.ZRem(ctx, key, member)

		// Estimate retry-after from the oldest entry still in the window.
		oldest, err := c.rdb.ZRangeWithScores(ctx, key, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			oldestTime := time.Unix(0, int64(oldest[0].Score))
			retryAfter = window - now.Sub(oldestTime)
			if retryAfter < 0 {
				retryAfter = 0
			}
		} else {
			retryAfter = window
		}
		return false, retryAfter, nil
	}

	return true, 0, nil
}
