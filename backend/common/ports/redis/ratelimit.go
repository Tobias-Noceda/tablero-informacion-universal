package redis

import (
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
)

var _ infrastructure.RateLimiter = (*RedisDB)(nil)

func rateKey(key string) string { return "rate:" + key }

// Allow counts one attempt in a fixed window and refuses once the bucket
// holds more than limit.
func (db *RedisDB) Allow(key string, limit int, window time.Duration) (bool, error) {
	ctx, cancel := timeout()
	defer cancel()

	count, err := db.client.Incr(ctx, rateKey(key)).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := db.client.Expire(ctx, rateKey(key), window).Err(); err != nil {
			return false, err
		}
	}

	return count <= int64(limit), nil
}
