package redis

import (
	"context"
)

// counterKey is the single global counter used to mint new Link IDs.
// INCR is atomic in Redis, so concurrent requests never get the same ID
// without needing any database-side locking.
const counterKey = "urlshortener:link_id_counter"

// NextID atomically increments and returns the next unique ID to use
// as the base for a new short code.
func (c *Client) NextID(ctx context.Context) (uint64, error) {
	val, err := c.rdb.Incr(ctx, counterKey).Result()
	if err != nil {
		return 0, err
	}
	return uint64(val), nil
}
