package jwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	gojwt "github.com/golang-jwt/jwt/v5"
)

func seed(t *testing.T) string {
	t.Helper()
	buf := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(buf); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func claims(now time.Time) infrastructure.Claims {
	return infrastructure.Claims{
		Subject:   "user-1",
		Admin:     true,
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
		ID:        "jti-1",
	}
}

func TestKeyring_RoundTrip(t *testing.T) {
	ring, err := NewFromKeys("k1:" + seed(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	token, err := ring.Sign(claims(now))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	got, err := ring.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Subject != "user-1" || !got.Admin || got.ID != "jti-1" || !got.IssuedAt.Equal(now) || !got.ExpiresAt.Equal(now.Add(15*time.Minute)) {
		t.Errorf("claims = %+v", got)
	}
}

func TestKeyring_RejectsExpiredToken(t *testing.T) {
	ring, _ := NewFromKeys("k1:" + seed(t))

	c := claims(time.Now().Add(-time.Hour))
	token, _ := ring.Sign(c)

	if _, err := ring.Verify(token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("expired: err = %v, want ErrInvalidToken", err)
	}
}

func TestKeyring_RejectsUnknownKeyAndForeignSignature(t *testing.T) {
	ring, _ := NewFromKeys("k1:" + seed(t))
	other, _ := NewFromKeys("k2:" + seed(t))

	token, _ := other.Sign(claims(time.Now()))
	if _, err := ring.Verify(token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("unknown kid: err = %v", err)
	}

	sameKid, _ := NewFromKeys("k1:" + seed(t))
	token, _ = sameKid.Sign(claims(time.Now()))
	if _, err := ring.Verify(token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("same kid, other key: err = %v", err)
	}
}

func TestKeyring_RejectsOtherAlgorithms(t *testing.T) {
	ring, _ := NewFromKeys("k1:" + seed(t))

	unsigned := gojwt.NewWithClaims(gojwt.SigningMethodNone, gojwt.MapClaims{"sub": "user-1", "exp": time.Now().Add(time.Hour).Unix()})
	unsigned.Header["kid"] = "k1"
	token, _ := unsigned.SignedString(gojwt.UnsafeAllowNoneSignatureType)
	if _, err := ring.Verify(token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("alg none: err = %v", err)
	}

	hmac := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{"sub": "user-1", "exp": time.Now().Add(time.Hour).Unix()})
	hmac.Header["kid"] = "k1"
	token, _ = hmac.SignedString([]byte("secret"))
	if _, err := ring.Verify(token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("alg HS256: err = %v", err)
	}

	if _, err := ring.Verify("not.a.token"); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("garbage: err = %v", err)
	}
}

func TestKeyring_FirstKeySignsEveryKeyVerifies(t *testing.T) {
	oldSeed := seed(t)
	oldRing, _ := NewFromKeys("k1:" + oldSeed)
	oldToken, _ := oldRing.Sign(claims(time.Now()))

	rotated, err := NewFromKeys("k2:" + seed(t) + ",k1:" + oldSeed)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if _, err := rotated.Verify(oldToken); err != nil {
		t.Errorf("token signed before rotation must still verify: %v", err)
	}

	newToken, _ := rotated.Sign(claims(time.Now()))
	if _, err := oldRing.Verify(newToken); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("new tokens must be signed with k2: err = %v", err)
	}
	if !strings.Contains(headerOf(t, newToken), `"kid":"k2"`) {
		t.Errorf("header = %s", headerOf(t, newToken))
	}
}

func TestNewFromKeys_RejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "k1", "k1:notbase64!", "k1:" + base64.StdEncoding.EncodeToString([]byte("short")), ":" + seed(t)} {
		if _, err := NewFromKeys(in); err == nil {
			t.Errorf("NewFromKeys(%q) accepted", in)
		}
	}
}

func headerOf(t *testing.T, token string) string {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
