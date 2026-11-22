package entitlements

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

func withLocks(locks infrastructure.Locker) *Service {
	return New(&mocks.MemoryUserStore{}, &mocks.MemoryOrgStore{}, &mocks.MockDB{}, &mocks.MemoryGroupStore{}, nil, locks, time.Now)
}

// What counts against one limit of one payer happens one request at a
// time, so two of them cannot both see room for the last one.
func TestHold_SerialisesTheSameLimitOfTheSamePayer(t *testing.T) {
	svc := withLocks(mocks.NewMemoryLocker())

	var inside, most atomic.Int32
	var done sync.WaitGroup
	for range 10 {
		done.Add(1)
		go func() {
			defer done.Done()
			release, err := svc.Hold(models.LimitBoards, "ana")
			if err != nil {
				t.Error(err)
				return
			}
			now := inside.Add(1)
			for {
				seen := most.Load()
				if now <= seen || most.CompareAndSwap(seen, now) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			inside.Add(-1)
			release()
		}()
	}
	done.Wait()

	if n := most.Load(); n != 1 {
		t.Errorf("%d requests held the same limit at once, want 1", n)
	}
}

func TestHold_OtherPayersAndLimitsDoNotWait(t *testing.T) {
	svc := withLocks(mocks.NewMemoryLocker())
	release, err := svc.Hold(models.LimitBoards, "ana")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	for _, c := range []struct{ limit, holder string }{
		{models.LimitBoards, "bob"},
		{models.LimitGroups, "ana"},
	} {
		other, err := svc.Hold(c.limit, c.holder)
		if err != nil {
			t.Fatalf("%s of %s: %v", c.limit, c.holder, err)
		}
		other()
	}
}

func TestHold_GivesUpOnALockThatNeverFrees(t *testing.T) {
	never := &mocks.MockLocker{AcquireFn: func(string, time.Duration) (string, bool, error) { return "", false, nil }}
	svc := withLocks(never)
	svc.wait = 30 * time.Millisecond

	if _, err := svc.Hold(models.LimitBoards, "ana"); !errors.Is(err, infrastructure.ErrRateLimited) {
		t.Errorf("err = %v, want to be told to try again", err)
	}
}

func TestHold_WithoutALockerHoldsNothing(t *testing.T) {
	release, err := withLocks(nil).Hold(models.LimitBoards, "ana")
	if err != nil {
		t.Fatal(err)
	}
	release()
}
