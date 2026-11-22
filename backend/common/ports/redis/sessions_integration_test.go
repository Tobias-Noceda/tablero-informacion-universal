//go:build integration

package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/google/uuid"
)

func integrationCache(t *testing.T) *RedisDB {
	t.Helper()

	db, err := New()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := timeout()
		defer cancel()
		_ = db.client.FlushDB(ctx).Err()
		_ = db.Close()
	})

	return db
}

func session(user, family string) infrastructure.Session {
	return infrastructure.Session{TokenHash: uuid.NewString(), Family: family, UserID: user, IssuedAt: time.Now().UTC().Truncate(time.Second)}
}

func TestRedis_SessionsTakeIsSingleUse(t *testing.T) {
	db := integrationCache(t)

	s := session("u1", "f1")
	if err := db.PutSession(s, time.Minute); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := db.TakeSession(s.TokenHash)
	if err != nil || got.UserID != "u1" || got.Family != "f1" || !got.IssuedAt.Equal(s.IssuedAt) {
		t.Fatalf("take = %+v, %v", got, err)
	}
	if _, err := db.TakeSession(s.TokenHash); !errors.Is(err, infrastructure.ErrSessionNotFound) {
		t.Errorf("second take: err = %v, want ErrSessionNotFound", err)
	}
}

func TestRedis_SessionsSpentMarker(t *testing.T) {
	db := integrationCache(t)

	if _, _, found, err := db.Spent("nothing"); err != nil || found {
		t.Fatalf("unknown spent = found %v, %v", found, err)
	}

	marked := time.Now().UTC().Truncate(time.Millisecond)
	if err := db.MarkSpent("h1", "f1", marked, time.Minute); err != nil {
		t.Fatalf("mark: %v", err)
	}
	family, at, found, err := db.Spent("h1")
	if err != nil || !found || family != "f1" || !at.Equal(marked) {
		t.Errorf("spent = %q, %v, %v, %v", family, at, found, err)
	}
}

func TestRedis_SessionsRevokeFamilyAndUser(t *testing.T) {
	db := integrationCache(t)

	a1, a2 := session("ana", "fa"), session("ana", "fa")
	b1 := session("ana", "fb")
	c1 := session("carl", "fc")
	for _, s := range []infrastructure.Session{a1, a2, b1, c1} {
		if err := db.PutSession(s, time.Minute); err != nil {
			t.Fatalf("put: %v", err)
		}
	}

	if err := db.RevokeFamily("fa"); err != nil {
		t.Fatalf("revoke family: %v", err)
	}
	for _, hash := range []string{a1.TokenHash, a2.TokenHash} {
		if _, err := db.TakeSession(hash); !errors.Is(err, infrastructure.ErrSessionNotFound) {
			t.Errorf("family token still redeemable: %v", err)
		}
	}
	if _, err := db.TakeSession(b1.TokenHash); err != nil {
		t.Errorf("other family must survive: %v", err)
	}
	if err := db.PutSession(b1, time.Minute); err != nil {
		t.Fatalf("put back: %v", err)
	}

	if err := db.RevokeUser("ana"); err != nil {
		t.Fatalf("revoke user: %v", err)
	}
	if _, err := db.TakeSession(b1.TokenHash); !errors.Is(err, infrastructure.ErrSessionNotFound) {
		t.Errorf("user token still redeemable: %v", err)
	}
	if _, err := db.TakeSession(c1.TokenHash); err != nil {
		t.Errorf("other user must survive: %v", err)
	}
}

func TestRedis_RateLimiterCountsWithinWindow(t *testing.T) {
	db := integrationCache(t)

	key := "login:" + uuid.NewString()
	for i := 0; i < 3; i++ {
		if ok, err := db.Allow(key, 3, time.Minute); err != nil || !ok {
			t.Fatalf("attempt %d: allowed = %v, %v", i+1, ok, err)
		}
	}
	if ok, err := db.Allow(key, 3, time.Minute); err != nil || ok {
		t.Errorf("fourth attempt: allowed = %v, %v; want refused", ok, err)
	}

	ctx, cancel := timeout()
	defer cancel()
	if ttl := db.client.TTL(ctx, "rate:"+key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("window ttl = %v", ttl)
	}
}

// A counter left without a window (the old limiter set it in a second call
// that could fail) would refuse that key forever. The next attempt gives it
// a window again, so it empties as any other.
func TestRedis_RateLimiterHealsACounterWithoutAWindow(t *testing.T) {
	db := integrationCache(t)
	key := "login:" + uuid.NewString()

	ctx, cancel := timeout()
	defer cancel()
	if err := db.client.Set(ctx, "rate:"+key, 50, 0).Err(); err != nil {
		t.Fatal(err)
	}

	if ok, err := db.Allow(key, 3, time.Minute); err != nil || ok {
		t.Fatalf("allowed = %v, %v; the stuck counter is still over the limit", ok, err)
	}
	if ttl := db.client.TTL(ctx, "rate:"+key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("window ttl = %v, want the counter to expire within the window", ttl)
	}
}

// The window starts with the first attempt and later ones do not stretch it.
func TestRedis_RateLimiterKeepsTheFirstWindow(t *testing.T) {
	db := integrationCache(t)
	key := "login:" + uuid.NewString()

	if _, err := db.Allow(key, 10, time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := timeout()
	defer cancel()
	if err := db.client.PExpire(ctx, "rate:"+key, 30*time.Second).Err(); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Allow(key, 10, time.Minute); err != nil {
		t.Fatal(err)
	}
	if ttl := db.client.TTL(ctx, "rate:"+key).Val(); ttl > 30*time.Second {
		t.Errorf("window ttl = %v, want the second attempt to leave it alone", ttl)
	}
}
