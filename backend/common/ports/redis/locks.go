package redis

import (
	"time"
)

func (db *RedisDB) Acquire(key string, ttl time.Duration) (bool, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.client.SetNX(ctx, "lock:"+key, "1", ttl).Result()
}

func (db *RedisDB) Release(key string) error {
	ctx, cancel := timeout()
	defer cancel()

	return db.client.Del(ctx, "lock:"+key).Err()
}
