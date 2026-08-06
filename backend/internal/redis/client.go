package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func New(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		// Tuned for a high-throughput redirect hot path
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	return &Client{rdb: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Raw exposes the underlying redis client for use in other packages
// (cache, counter, rate-limiter) built in later phases.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) String() string {
	return fmt.Sprintf("redis-client(%s)", c.rdb.Options().Addr)
}
