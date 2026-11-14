package models

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLimitsFor_Table(t *testing.T) {
	want := map[Plan]Limits{
		PlanFree: {0, 5, 5, true, 25, 5, 0, 0, 5, 0, 300, false, false, 0},
		PlanPro:  {3, 10, 5, false, 100, 10, 5, 3, 25, 5, 150, false, true, 9},
		PlanMax:  {10, 50, 25, false, 300, 50, 50, 50, 200, 25, 60, true, true, 29},
	}
	for plan, limits := range want {
		if got := LimitsFor(plan); got != limits {
			t.Errorf("LimitsFor(%s) = %+v, want %+v", plan, got, limits)
		}
	}
	if LimitsFor("gold") != LimitsFor(PlanFree) || LimitsFor("") != LimitsFor(PlanFree) {
		t.Error("an unknown plan must get the free limits")
	}
}

func TestPlan_ValidAndRank(t *testing.T) {
	if !PlanPro.Valid() || Plan("gold").Valid() || Plan("").Valid() {
		t.Error("Valid")
	}
	if !(PlanFree.Rank() < PlanPro.Rank() && PlanPro.Rank() < PlanMax.Rank() && Plan("x").Rank() < PlanFree.Rank()) {
		t.Error("Rank order")
	}
}

func TestUser_EffectivePlan(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	for name, c := range map[string]struct {
		user User
		want Plan
	}{
		"empty":             {User{}, PlanFree},
		"paid, no expiry":   {User{Plan: PlanPro}, PlanPro},
		"paid, running":     {User{Plan: PlanMax, PlanExpires: &future}, PlanMax},
		"paid, lapsed":      {User{Plan: PlanMax, PlanExpires: &past}, PlanFree},
		"lapsing right now": {User{Plan: PlanPro, PlanExpires: &now}, PlanFree},
		"unknown stored":    {User{Plan: "gold"}, PlanFree},
		"admin":             {User{Admin: true}, PlanMax},
	} {
		if got := c.user.EffectivePlan(now); got != c.want {
			t.Errorf("%s: %s, want %s", name, got, c.want)
		}
	}
}

func TestUser_ProfileCarriesPlanAndLimits(t *testing.T) {
	future := time.Now().Add(time.Hour)
	user := User{Id: uuid.New(), Plan: PlanPro, PlanExpires: &future}
	profile := user.Profile()
	if profile.Plan != PlanPro || profile.Limits != LimitsFor(PlanPro) || profile.PlanExpires == nil {
		t.Errorf("profile = %+v", profile)
	}
	if free := (&User{Id: uuid.New()}).Profile(); free.Plan != PlanFree || free.PlanExpires != nil {
		t.Errorf("free profile = %+v", free)
	}
}

func TestEntitlement_Admits(t *testing.T) {
	e := EntitlementOf("u", PlanFree)
	if err := e.Admits(LimitBoards, 4, 5); err != nil {
		t.Errorf("limit-1: %v", err)
	}
	err := e.Admits(LimitBoards, 5, 5)
	var limit *LimitError
	if !errors.As(err, &limit) || limit.Limit != LimitBoards || limit.Max != 5 || limit.Plan != PlanFree {
		t.Errorf("limit: %v", err)
	}
}

func TestBoard_PeopleCountsEachPersonOnce(t *testing.T) {
	board := &Board{Owner: "o", Members: []BoardMember{{User: "a", Role: BoardEditor}}}
	org := &Org{Members: []OrgMember{{User: "o"}, {User: "a"}, {User: "b"}}}
	if got := len(board.People(nil)); got != 2 {
		t.Errorf("no org: %d", got)
	}
	if got := len(board.People(org)); got != 3 {
		t.Errorf("org union: %d, want 3", got)
	}
}
