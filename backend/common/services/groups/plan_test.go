package groups

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
)

func limited(err error, limit string) bool {
	var le *models.LimitError
	return errors.As(err, &le) && le.Limit == limit
}

func TestCreate_GroupsByPlan(t *testing.T) {
	for _, plan := range models.Plans {
		max := models.LimitsFor(plan).Groups
		// Fill to the plan's max, then one more is refused.
		store := &mocks.MemoryGroupStore{}
		srv := New(store, &purgeRecorder{}, access.New(nil), mocks.StaticEntitlements{Plan: plan})
		for i := range max {
			if _, err := srv.Create(owner, "g", nil); err != nil {
				t.Fatalf("%s group %d of %d: %v", plan, i+1, max, err)
			}
		}
		if _, err := srv.Create(owner, "over", nil); !limited(err, models.LimitGroups) {
			t.Errorf("%s over %d: %v", plan, max, err)
		}
	}
}

func TestAddMember_SeatsByPlan(t *testing.T) {
	srv, store, _ := service()
	srv.plans = mocks.StaticEntitlements{Plan: models.PlanPro}
	group, err := srv.Create(owner, "g", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = store
	// Pro: 5 seats, the owner takes one.
	for _, id := range []string{"a", "b", "c", "d"} {
		if err := srv.AddMember(owner, group.Id, id); err != nil {
			t.Fatalf("add %s: %v", id, err)
		}
	}
	if err := srv.AddMember(owner, group.Id, "e"); !limited(err, models.LimitSeats) {
		t.Errorf("fifth member: %v", err)
	}
	if err := srv.AddMember(owner, group.Id, "a"); err != nil {
		t.Errorf("a member already there: %v", err)
	}
}
