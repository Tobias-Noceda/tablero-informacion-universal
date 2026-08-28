package redis

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/redis/go-redis/v9"
)

var releaseScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	end
	return 0
`)

func lockKey(key string) string {
	return "lock:" + key
}

func (db *RedisDB) Acquire(key string, ttl time.Duration) (string, bool, error) {
	ctx, cancel := timeout()
	defer cancel()

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", false, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	held, err := db.client.SetNX(ctx, lockKey(key), token, ttl).Result()
	if err != nil || !held {
		return "", false, err
	}

	return token, true, nil
}

func (db *RedisDB) Release(key, token string) error {
	ctx, cancel := timeout()
	defer cancel()

	return releaseScript.Run(ctx, db.client, []string{lockKey(key)}, token).Err()
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
