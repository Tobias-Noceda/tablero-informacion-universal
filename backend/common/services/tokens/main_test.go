package tokens

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ana = models.Principal{ID: uuid.NewString()}
	bob = models.Principal{ID: uuid.NewString()}
)

type fixture struct {
	svc     *TokenService
	store   *mocks.MemoryAccessTokenStore
	limiter *mocks.CountingLimiter
	clock   *time.Time
}

func setup() fixture {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	f := fixture{store: &mocks.MemoryAccessTokenStore{}, limiter: &mocks.CountingLimiter{}, clock: &now}
	f.svc = New(f.store, f.limiter, func() time.Time { return *f.clock })
	return f
}

func TestCreate_ReturnsThePlaintextOnceAndStoresOnlyItsHash(t *testing.T) {
	f := setup()

	created, err := f.svc.Create(ana, "  laptop  ", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(created.Token, PREFIX) || len(created.Token) != len(PREFIX)+43 {
		t.Errorf("token = %q", created.Token)
	}
	if created.Name != "laptop" || created.Prefix != created.Token[:displayed] {
		t.Errorf("created = %+v", created.AccessToken)
	}
	if got := created.ExpiresAt.Sub(created.CreatedAt); got != DEFAULT_DAYS*24*time.Hour {
		t.Errorf("lifetime = %v", got)
	}

	stored := f.store.Tokens[0]
	if stored.Hash == created.Token || strings.Contains(stored.Hash, created.Token[len(PREFIX):]) || stored.Hash != hash(created.Token) {
		t.Errorf("stored hash = %q", stored.Hash)
	}

	list, _ := f.svc.List(ana)
	if len(list) != 1 || list[0].Id != created.Id {
		t.Errorf("list = %+v", list)
	}
}

func TestCreate_Validates(t *testing.T) {
	f := setup()

	cases := map[string]struct {
		principal models.Principal
		name      string
		days      int
		want      error
	}{
		"anonymous":     {models.Principal{}, "x", 15, ErrForbidden},
		"from a token":  {models.Principal{ID: ana.ID, Token: true}, "x", 15, ErrForbidden},
		"blank name":    {ana, "   ", 15, ErrInvalidName},
		"long name":     {ana, strings.Repeat("a", MAX_NAME+1), 15, ErrInvalidName},
		"odd lifetime":  {ana, "x", 365, ErrInvalidExpiry},
		"negative days": {ana, "x", -1, ErrInvalidExpiry},
	}
	for name, c := range cases {
		if _, err := f.svc.Create(c.principal, c.name, c.days); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	for _, days := range []int{7, 15, 30} {
		if _, err := f.svc.Create(ana, "ok", days); err != nil {
			t.Errorf("%d days: %v", days, err)
		}
	}
}

func TestCreate_Limits(t *testing.T) {
	f := setup()

	for range MAX_TOKENS {
		if _, err := f.svc.Create(ana, "t", 7); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if _, err := f.svc.Create(ana, "one too many", 7); !errors.Is(err, ErrTooManyTokens) {
		t.Errorf("over the cap: err = %v", err)
	}

	// Expired tokens no longer count.
	*f.clock = f.clock.Add(8 * 24 * time.Hour)
	if _, err := f.svc.Create(ana, "after expiry", 7); err != nil {
		t.Errorf("after expiry: %v", err)
	}

	f.limiter.Refuse = map[string]bool{"pat-create:" + bob.ID: true}
	if _, err := f.svc.Create(bob, "t", 7); !errors.Is(err, infrastructure.ErrRateLimited) {
		t.Errorf("rate limited: err = %v", err)
	}
}

func TestResolve(t *testing.T) {
	f := setup()
	created, _ := f.svc.Create(ana, "laptop", 7)

	claims, err := f.svc.Resolve(created.Token)
	if err != nil || claims.Subject != ana.ID || !claims.AccessToken || claims.Admin {
		t.Fatalf("claims = %+v, err = %v", claims, err)
	}
	if f.store.Tokens[0].LastUsedAt == nil {
		t.Error("a use must be recorded")
	}

	for name, raw := range map[string]string{"unknown": PREFIX + "nope", "empty": "", "tampered": created.Token + "x"} {
		if _, err := f.svc.Resolve(raw); !errors.Is(err, infrastructure.ErrInvalidToken) {
			t.Errorf("%s: err = %v", name, err)
		}
	}

	*f.clock = created.ExpiresAt
	if _, err := f.svc.Resolve(created.Token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("expired: err = %v", err)
	}
}

func TestRevoke_OnlyOwnTokens(t *testing.T) {
	f := setup()
	created, _ := f.svc.Create(ana, "laptop", 7)

	if err := f.svc.Revoke(bob, created.Id); !errors.Is(err, infrastructure.ErrAccessTokenNotFound) {
		t.Errorf("someone else's: err = %v", err)
	}
	if err := f.svc.Revoke(ana, created.Id); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := f.svc.Resolve(created.Token); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("after revoke: err = %v", err)
	}
}

func TestVerifier_RoutesByPrefix(t *testing.T) {
	f := setup()
	created, _ := f.svc.Create(ana, "laptop", 7)
	v := f.svc.Verifier(mocks.StaticVerifier{"jwt": {Subject: bob.ID, Admin: true}})

	if claims, err := v.Verify(created.Token); err != nil || claims.Subject != ana.ID || !claims.AccessToken {
		t.Errorf("token: %+v, %v", claims, err)
	}
	if claims, err := v.Verify("jwt"); err != nil || claims.Subject != bob.ID || claims.AccessToken {
		t.Errorf("jwt: %+v, %v", claims, err)
	}
	if _, err := v.Verify(PREFIX + "forged"); !errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("forged: %v", err)
	}
}
