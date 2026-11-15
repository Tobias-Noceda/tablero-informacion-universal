package orgs

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func limited(err error, limit string) bool {
	var le *models.LimitError
	return errors.As(err, &le) && le.Limit == limit
}

func TestCreate_OwnedOrgsByPlan(t *testing.T) {
	for _, plan := range models.Plans {
		max := models.LimitsFor(plan).Orgs
		for _, owned := range []int{max - 1, max} {
			if owned < 0 {
				continue
			}
			srv, store := service()
			srv.plans = mocks.StaticEntitlements{Plan: plan}
			for range owned {
				store.Orgs = append(store.Orgs, models.Org{Id: uuid.New(), Owner: ana.ID, Members: []models.OrgMember{{User: ana.ID, Role: models.OrgRoleAdmin}}})
			}
			org, err := srv.Create(ana, "acme")
			if owned >= max {
				if !limited(err, models.LimitOrgs) {
					t.Errorf("%s owning %d: err = %v", plan, owned, err)
				}
			} else if err != nil || org.Owner != ana.ID {
				t.Errorf("%s owning %d: %v %+v", plan, owned, err, org)
			}
		}
	}
}

func TestSetMember_SeatsByPlan(t *testing.T) {
	for _, plan := range models.Plans {
		seats := models.LimitsFor(plan).Seats
		for _, size := range []int{seats - 1, seats} {
			if size < 1 {
				continue
			}
			srv, store := service()
			srv.plans = mocks.StaticEntitlements{Plan: plan}
			org := models.Org{Id: uuid.New(), Owner: ana.ID, Members: []models.OrgMember{{User: ana.ID, Role: models.OrgRoleAdmin}}}
			for len(org.Members) < size {
				org.Members = append(org.Members, models.OrgMember{User: uuid.New().String(), Role: models.OrgRoleMember})
			}
			store.Orgs = append(store.Orgs, org)

			_, err := srv.SetMember(ana, org.Id, bobUser.Email, models.OrgRoleMember)
			if size >= seats {
				if !limited(err, models.LimitSeats) {
					t.Errorf("%s with %d members: err = %v", plan, size, err)
				}
			} else if err != nil {
				t.Errorf("%s with %d members: %v", plan, size, err)
			}
		}
	}
}

func TestOrgResponses_CarryThePayersLimitsAndUsage(t *testing.T) {
	srv, store := service()
	srv.plans = mocks.StaticEntitlements{Plan: models.PlanPro}
	org := models.Org{Id: uuid.New(), Owner: bob.ID, Members: []models.OrgMember{{User: bob.ID, Role: models.OrgRoleAdmin}, {User: ana.ID, Role: models.OrgRoleMember}}}
	store.Orgs = append(store.Orgs, org)
	store.Boards[org.Id] = 3

	want := models.LimitsFor(models.PlanPro)
	list, err := srv.ListMine(ana)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	got := list[0]
	if got.Limits == nil || *got.Limits != want || got.SeatsUsed == nil || *got.SeatsUsed != 2 || got.BoardsUsed == nil || *got.BoardsUsed != 3 {
		t.Errorf("list entry = %+v", got)
	}

	detail, err := srv.Get(ana, org.Id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Limits == nil || *detail.Limits != want || *detail.SeatsUsed != 2 || *detail.BoardsUsed != 3 || detail.Owner != bob.ID {
		t.Errorf("detail = %+v", detail)
	}
}
