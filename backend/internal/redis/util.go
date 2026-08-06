package redis

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/redis/go-redis/v9"
)

func redisZ(score float64, member string) redis.Z {
	return redis.Z{Score: score, Member: member}
}

// randSuffix guards against two requests landing on the exact same
// nanosecond timestamp (rare, but possible under high concurrency) and
// colliding as sorted-set members.
func randSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
