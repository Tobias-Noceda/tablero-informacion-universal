package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	REDIS_URL       = "redis://redis:6379/"
	REQUEST_TIMEOUT = 5 * time.Second
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

func onlineBoardKey(board *models.Board) string {
	return "board:" + board.Id.String() + ":online"
}

func (db *RedisDB) ConnectClientToBoard(board *models.Board, id uuid.UUID) ([]string, error) {
	ctx, cancel := timeout()
	defer cancel()

	key := onlineBoardKey(board)

	var clients []string
	err := db.client.Watch(ctx, func(tx *redis.Tx) error {
		var err error

		clients, err = tx.SMembers(ctx, key).Result()
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.SAdd(ctx, key, id.String())
			return nil
		})

		return err
	}, key)

	return clients, err
}

func (db *RedisDB) DisconnectClientFromBoard(board *models.Board, id uuid.UUID) error {
	ctx, cancel := timeout()
	defer cancel()

	key := onlineBoardKey(board)

	return db.client.SRem(ctx, key, id.String()).Err()
}
