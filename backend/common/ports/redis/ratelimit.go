package redis

import (
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/redis/go-redis/v9"
)

var _ infrastructure.RateLimiter = (*RedisDB)(nil)

func rateKey(key string) string { return "rate:" + key }

// countInWindow counts one attempt and gives the counter its window in the
// same step: as two calls, a failed EXPIRE left a counter that never
// emptied. A counter found without a window (PTTL -1) gets one too, so one
// left behind that way heals on its next attempt; a running window is never
// stretched.
var countInWindow = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) == -1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// Allow counts one attempt in a fixed window and refuses once the bucket
// holds more than limit.
func (db *RedisDB) Allow(key string, limit int, window time.Duration) (bool, error) {
	ctx, cancel := timeout()
	defer cancel()

	count, err := countInWindow.Run(ctx, db.client, []string{rateKey(key)}, window.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}

	return count <= int64(limit), nil
}
