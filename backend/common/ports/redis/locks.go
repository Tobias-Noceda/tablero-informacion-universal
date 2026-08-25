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

// Put stores an in-flight handshake under a TTL.
func (db *RedisDB) Put(key string, value []byte, ttl time.Duration) error {
	ctx, cancel := timeout()
	defer cancel()

	return db.client.Set(ctx, key, value, ttl).Err()
}

// Take reads and removes an entry atomically, so a replayed state finds
// nothing the second time.
func (db *RedisDB) Take(key string) ([]byte, error) {
	ctx, cancel := timeout()
	defer cancel()

	return db.client.GetDel(ctx, key).Bytes()
}
