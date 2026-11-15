package postits

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

func limited(err error, limit string) bool {
	var le *models.LimitError
	return errors.As(err, &le) && le.Limit == limit
}

func TestCreatePostIt_NodesPerBoardByPlan(t *testing.T) {
	for _, plan := range models.Plans {
		max := models.LimitsFor(plan).NodesPerBoard
		for _, have := range []int{max - 1, max} {
			board := uuid.New()
			db := boardOf(boardOwner)
			find := db.FindBoardFn
			db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
				b, _ := find(id)
				b.PostIts = make([]models.BoardPostIt, have)
				return b, nil
			}
			svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil, mocks.StaticEntitlements{Plan: plan}, &mocks.CountingLimiter{})
			_, err := svc.CreatePostIt(boardOwner, &models.BoardPostItTitle{Text: "x"}, oneOff(board, nil))
			if have >= max && !limited(err, models.LimitNodes) {
				t.Errorf("%s with %d nodes: %v", plan, have, err)
			}
			if have < max && err != nil {
				t.Errorf("%s with %d nodes: %v", plan, have, err)
			}
		}
	}
}

func cardOf(rate int) *models.PostIts {
	return &models.PostIts{Id: uuid.New(), Board: uuid.New(), RunAs: boardOwner.ID, Rate: rate,
		Resource: &url.URL{Scheme: "https", Host: "example.com"}, Request: models.Request{Method: "GET"}}
}

// cachedRate runs the card and reports the rate its result was cached with.
func cachedRate(t *testing.T, svc *PostItsService, cache chan int, run func(*models.PostIts) (any, error)) int {
	t.Helper()
	if _, err := run(cardOf(5)); err != nil {
		t.Fatal(err)
	}
	select {
	case rate := <-cache:
		return rate
	case <-time.After(time.Second):
		t.Fatal("nothing was cached")
		return 0
	}
}

func serviceOn(plan models.Plan, limiter *mocks.CountingLimiter) (*PostItsService, chan int) {
	cache := make(chan int, 4)
	c := &mocks.MockCache{AddPostItResultFn: func(p *models.PostIts, _ any) error { cache <- p.Rate; return nil }}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) { return "ok", nil }}
	return New(boardOf(boardOwner), c, run, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil, mocks.StaticEntitlements{Plan: plan}, limiter), cache
}

func TestExecutePostIt_TTLIsTheLargerOfRateAndMinRate(t *testing.T) {
	for _, plan := range models.Plans {
		svc, cache := serviceOn(plan, &mocks.CountingLimiter{})
		want := models.LimitsFor(plan).MinRate
		if got := cachedRate(t, svc, cache, svc.ExecutePostIt); got != want {
			t.Errorf("%s: rate 5 cached for %d, want %d", plan, got, want)
		}

		svc, cache = serviceOn(plan, &mocks.CountingLimiter{})
		slow := cardOf(900)
		if _, err := svc.ExecutePostIt(slow); err != nil {
			t.Fatal(err)
		}
		if got := <-cache; got != 900 {
			t.Errorf("%s: rate 900 cached for %d", plan, got)
		}
	}
}

func TestExecutePostIt_CardWithoutResourceIsNotFloored(t *testing.T) {
	svc, cache := serviceOn(models.PlanFree, &mocks.CountingLimiter{})
	card := &models.PostIts{Id: uuid.New(), Board: uuid.New(), Rate: 5}
	if _, err := svc.ExecutePostIt(card); err != nil {
		t.Fatal(err)
	}
	if got := <-cache; got != 5 {
		t.Errorf("cached for %d", got)
	}
}

func TestRefreshPostIt_OnlyMaxAndRateLimited(t *testing.T) {
	for _, plan := range []models.Plan{models.PlanFree, models.PlanPro} {
		svc, _ := serviceOn(plan, &mocks.CountingLimiter{})
		if _, err := svc.RefreshPostIt(cardOf(60)); !limited(err, models.LimitForceRefresh) {
			t.Errorf("%s: %v", plan, err)
		}
	}

	card := cardOf(60)
	limiter := &mocks.CountingLimiter{}
	svc, cache := serviceOn(models.PlanMax, limiter)
	if _, err := svc.RefreshPostIt(card); err != nil {
		t.Fatalf("max: %v", err)
	}
	if got := <-cache; got != models.LimitsFor(models.PlanMax).MinRate {
		t.Errorf("refreshed result cached for %d", got)
	}
	limiter.Refuse = map[string]bool{"fresh:" + card.Id.String(): true}
	if _, err := svc.RefreshPostIt(card); !errors.Is(err, ErrRateLimited) {
		t.Errorf("second refresh: %v", err)
	}
}

func TestExecutePostIt_FailuresAreNotCached(t *testing.T) {
	cached := false
	c := &mocks.MockCache{AddPostItResultFn: func(*models.PostIts, any) error { cached = true; return nil }}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) { return nil, errors.New("boom") }}
	svc := New(boardOf(boardOwner), c, run, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil, mocks.StaticEntitlements{}, &mocks.CountingLimiter{})
	if _, err := svc.ExecutePostIt(cardOf(60)); err == nil {
		t.Fatal("expected the failure")
	}
	time.Sleep(20 * time.Millisecond)
	if cached {
		t.Error("a failure was cached")
	}
}

func TestDeletePostIt_DropsTheCachedResult(t *testing.T) {
	var dropped uuid.UUID
	db := boardOf(boardOwner)
	card := cardOf(60)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) { return card, nil }
	db.DeletePostItFn = func(uuid.UUID) ([]models.Strand, error) { return nil, nil }
	c := &mocks.MockCache{DropPostItResultFn: func(id uuid.UUID) error { dropped = id; return nil }}
	svc := New(db, c, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil, mocks.StaticEntitlements{}, &mocks.CountingLimiter{})
	if _, err := svc.DeletePostIt(boardOwner, card.Id); err != nil {
		t.Fatal(err)
	}
	if dropped != card.Id {
		t.Errorf("dropped %v", dropped)
	}
}
