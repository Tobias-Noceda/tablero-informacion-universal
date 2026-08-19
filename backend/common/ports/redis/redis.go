package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Secreto31126/tesis/common/env"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	REQUEST_TIMEOUT = 5 * time.Second
)

var (
	REDIS_URL = env.Get("REDIS_URL", "redis://redis:6379/")
)

type RedisDB struct {
	client *redis.Client
}

func New() (*RedisDB, error) {
	db := &RedisDB{}

	opt, err := redis.ParseURL(REDIS_URL)
	if err != nil {
		return nil, err
	}

	db.client = redis.NewClient(opt)

	return db, nil
}

func (db *RedisDB) Close() error {
	return db.client.Close()
}

func timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
}

func (db *RedisDB) FindPostItResult(id uuid.UUID) (any, error) {
	ctx, cancel := timeout()
	defer cancel()

	val, err := db.client.Get(ctx, id.String()).Bytes()
	if err != nil {
		return nil, err
	}

	var data any
	if err := json.Unmarshal(val, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (db *RedisDB) AddPostItResult(postit *models.PostIts, data any) error {
	ctx, cancel := timeout()
	defer cancel()
	return db.client.Set(ctx, postit.Id.String(), data, time.Duration(postit.Rate)*time.Minute).Err()
}
