package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/redis/go-redis/v9"
)

var _ infrastructure.SessionStore = (*RedisDB)(nil)

func sessionKey(tokenHash string) string { return "session:" + tokenHash }
func spentKey(tokenHash string) string   { return "spent:" + tokenHash }
func familyKey(family string) string     { return "family:" + family }
func userFamiliesKey(user string) string { return "user-families:" + user }

type spentMarker struct {
	Family string    `json:"family"`
	At     time.Time `json:"at"`
}

func (db *RedisDB) PutSession(session infrastructure.Session, ttl time.Duration) error {
	ctx, cancel := timeout()
	defer cancel()

	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	pipe := db.client.TxPipeline()
	pipe.Set(ctx, sessionKey(session.TokenHash), payload, ttl)
	pipe.SAdd(ctx, familyKey(session.Family), session.TokenHash)
	pipe.Expire(ctx, familyKey(session.Family), ttl)
	pipe.SAdd(ctx, userFamiliesKey(session.UserID), session.Family)
	pipe.Expire(ctx, userFamiliesKey(session.UserID), ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (db *RedisDB) TakeSession(tokenHash string) (*infrastructure.Session, error) {
	ctx, cancel := timeout()
	defer cancel()

	payload, err := db.client.GetDel(ctx, sessionKey(tokenHash)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, infrastructure.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	session := &infrastructure.Session{}
	if err := json.Unmarshal(payload, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (db *RedisDB) MarkSpent(tokenHash, family string, at time.Time, ttl time.Duration) error {
	ctx, cancel := timeout()
	defer cancel()

	payload, err := json.Marshal(spentMarker{Family: family, At: at.UTC()})
	if err != nil {
		return err
	}

	return db.client.Set(ctx, spentKey(tokenHash), payload, ttl).Err()
}

func (db *RedisDB) Spent(tokenHash string) (string, time.Time, bool, error) {
	ctx, cancel := timeout()
	defer cancel()

	payload, err := db.client.Get(ctx, spentKey(tokenHash)).Bytes()
	if errors.Is(err, redis.Nil) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}

	marker := spentMarker{}
	if err := json.Unmarshal(payload, &marker); err != nil {
		return "", time.Time{}, false, err
	}

	return marker.Family, marker.At, true, nil
}

func (db *RedisDB) RevokeFamily(family string) error {
	ctx, cancel := timeout()
	defer cancel()

	return db.revokeFamilies(ctx, []string{family})
}

func (db *RedisDB) RevokeUser(userID string) error {
	ctx, cancel := timeout()
	defer cancel()

	families, err := db.client.SMembers(ctx, userFamiliesKey(userID)).Result()
	if err != nil {
		return err
	}
	if err := db.revokeFamilies(ctx, families); err != nil {
		return err
	}

	return db.client.Del(ctx, userFamiliesKey(userID)).Err()
}

func (db *RedisDB) revokeFamilies(ctx context.Context, families []string) error {
	for _, family := range families {
		hashes, err := db.client.SMembers(ctx, familyKey(family)).Result()
		if err != nil {
			return err
		}

		keys := make([]string, 0, len(hashes)+1)
		for _, hash := range hashes {
			keys = append(keys, sessionKey(hash))
		}
		keys = append(keys, familyKey(family))

		if err := db.client.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}

	return nil
}
