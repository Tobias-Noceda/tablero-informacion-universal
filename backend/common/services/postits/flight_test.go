package postits

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// joinGrace is how long a run waits once every request has missed the
// cache, so the last ones are inside the flight before it ends.
const joinGrace = 20 * time.Millisecond

// concurrently executes every card at once, after all of them missed the
// cache, and answers what each request got.
func concurrently(t *testing.T, svc *PostItsService, cards []*models.PostIts) ([]any, []error) {
	t.Helper()
	data := make([]any, len(cards))
	errs := make([]error, len(cards))

	var done sync.WaitGroup
	for i, card := range cards {
		done.Add(1)
		go func() {
			defer done.Done()
			data[i], errs[i] = svc.ExecutePostIt(card)
		}()
	}
	done.Wait()
	return data, errs
}

// missesThenWaits is a cache that always misses and counts who asked, and
// an executer that holds each run until every request has asked.
func missesThenWaits(requests int, result func(*models.PostIts) (any, error)) (*mocks.MockCache, *mocks.MockExecuter, *atomic.Int32) {
	var asked sync.WaitGroup
	asked.Add(requests)
	cache := &mocks.MockCache{
		FindPostItResultFn: func(uuid.UUID) (any, error) {
			asked.Done()
			return nil, mocks.ErrCacheMiss
		},
	}

	runs := &atomic.Int32{}
	run := &mocks.MockExecuter{
		ExecuteFn: func(card *models.PostIts) (any, error) {
			runs.Add(1)
			asked.Wait()
			time.Sleep(joinGrace)
			return result(card)
		},
	}
	return cache, run, runs
}

func TestExecutePostIt_ConcurrentMissesShareOneRun(t *testing.T) {
	const requests = 20
	cache, run, runs := missesThenWaits(requests, func(*models.PostIts) (any, error) { return "fresh", nil })
	svc := newService(nil, cache, run)

	card := models.PostIts{Id: uuid.New(), Params: map[string]string{"city": "Buenos Aires"}}
	cards := make([]*models.PostIts, requests)
	for i := range cards {
		copied := card
		cards[i] = &copied
	}

	data, errs := concurrently(t, svc, cards)

	if n := runs.Load(); n != 1 {
		t.Errorf("the card's API was asked %d times, want 1", n)
	}
	for i := range cards {
		if errs[i] != nil || data[i] != "fresh" {
			t.Errorf("request %d got (%v, %v)", i, data[i], errs[i])
		}
	}
}

// A request that loaded the card after an edit must not get the result of
// the settings it no longer has.
func TestExecutePostIt_EditedSettingsDoNotJoinTheOldRun(t *testing.T) {
	cache, run, runs := missesThenWaits(2, func(card *models.PostIts) (any, error) { return card.Params["city"], nil })
	svc := newService(nil, cache, run)

	id := uuid.New()
	before := &models.PostIts{Id: id, Params: map[string]string{"city": "Buenos Aires"}}
	after := &models.PostIts{Id: id, Params: map[string]string{"city": "Rosario"}}

	data, errs := concurrently(t, svc, []*models.PostIts{before, after})

	if n := runs.Load(); n != 2 {
		t.Errorf("runs = %d, want one per set of settings", n)
	}
	if errs[0] != nil || errs[1] != nil || data[0] != "Buenos Aires" || data[1] != "Rosario" {
		t.Errorf("got %v / %v (errors %v / %v)", data[0], data[1], errs[0], errs[1])
	}
}

func TestExecutePostIt_DifferentCardsRunApart(t *testing.T) {
	cache, run, runs := missesThenWaits(2, func(*models.PostIts) (any, error) { return "fresh", nil })
	svc := newService(nil, cache, run)

	concurrently(t, svc, []*models.PostIts{{Id: uuid.New()}, {Id: uuid.New()}})

	if n := runs.Load(); n != 2 {
		t.Errorf("runs = %d, want one per card", n)
	}
}

// Everyone waiting on a failed run hears the failure, but it is not kept:
// the next request tries the API again.
func TestExecutePostIt_AFailedRunIsSharedButNotKept(t *testing.T) {
	const requests = 5
	boom := errors.New("upstream down")
	cache, run, runs := missesThenWaits(requests, func(*models.PostIts) (any, error) { return nil, boom })
	svc := newService(nil, cache, run)

	card := models.PostIts{Id: uuid.New()}
	cards := make([]*models.PostIts, requests)
	for i := range cards {
		copied := card
		cards[i] = &copied
	}

	_, errs := concurrently(t, svc, cards)
	for i, err := range errs {
		if !errors.Is(err, boom) {
			t.Errorf("request %d: err = %v, want the run's failure", i, err)
		}
	}
	if n := runs.Load(); n != 1 {
		t.Fatalf("runs = %d, want 1", n)
	}

	svc.cache = &mocks.MockCache{}
	svc.exec = &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		runs.Add(1)
		return "recovered", nil
	}}
	if data, err := svc.ExecutePostIt(&card); err != nil || data != "recovered" {
		t.Errorf("after the failure got (%v, %v)", data, err)
	}
	if n := runs.Load(); n != 2 {
		t.Errorf("runs = %d, want the failure not to be reused", n)
	}
}
