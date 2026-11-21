package redis

import "context"

func (db *RedisDB) Ping(ctx context.Context) error {
	return db.client.Ping(ctx).Err()
}
