package boards

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

func isLimit(err error, limit string) bool {
	var le *models.LimitError
	return errors.As(err, &le) && le.Limit == limit
}

func TestCreateBoard_BoardsPerSpaceByPlan(t *testing.T) {
	for _, plan := range models.Plans {
		max := models.LimitsFor(plan).BoardsPerSpace
		for count, refused := range map[int]bool{max - 1: false, max: true} {
			db := &mocks.MockDB{
				CountPersonalBoardsFn: func(string) (int64, error) { return int64(count), nil },
				CreateBoardFn: func(name, owner string, org *uuid.UUID) (*models.Board, error) {
					return &models.Board{Id: uuid.New(), Owner: owner}, nil
				},
			}
			svc := New(db, &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, mocks.StaticEntitlements{Plan: plan})
			_, err := svc.CreateBoard(owner, "b", nil)
			if refused != isLimit(err, models.LimitBoards) || (!refused && err != nil) {
				t.Errorf("%s with %d boards: err = %v, refused want %v", plan, count, err, refused)
			}
		}
	}
}

func TestCreateBoard_OrgSpaceIsBilledToTheOrgPayer(t *testing.T) {
	payer := uuid.NewString()
	orgID := uuid.New()
	acme := models.Org{Id: orgID, Owner: payer, Members: []models.OrgMember{{User: owner.ID, Role: models.OrgRoleMember}}}
	orgs := &mocks.MemoryOrgStore{Orgs: []models.Org{acme}}
	db := &mocks.MockDB{CountOrgBoardsFn: func(uuid.UUID) (int64, error) { return 5, nil }}
	plans := mocks.StaticEntitlements{Plans: map[string]models.Plan{payer: models.PlanPro, owner.ID: models.PlanFree}}

	svc := New(db, &purgeRecorder{}, access.New(orgs), users(), &mocks.CountingLimiter{}, orgs, plans)
	db.CreateBoardFn = func(name, o string, org *uuid.UUID) (*models.Board, error) {
		return &models.Board{Id: uuid.New(), Owner: o, Org: org}, nil
	}
	if _, err := svc.CreateBoard(owner, "b", &orgID); err != nil {
		t.Errorf("5 boards under a pro payer: %v", err)
	}
	db.CountOrgBoardsFn = func(uuid.UUID) (int64, error) { return 10, nil }
	if _, err := svc.CreateBoard(owner, "b", &orgID); !isLimit(err, models.LimitBoards) {
		t.Errorf("10 boards under a pro payer: %v", err)
	}
}

func seatsDB(id uuid.UUID, members int) *mocks.MockDB {
	return &mocks.MockDB{FindBoardFn: func(uuid.UUID) (*models.Board, error) {
		board := &models.Board{Id: id, Owner: owner.ID}
		for range members {
			board.Members = append(board.Members, models.BoardMember{User: uuid.NewString(), Role: models.BoardViewer})
		}
		return board, nil
	}}
}

func TestSetMember_SeatsByPlan(t *testing.T) {
	id := uuid.New()
	for _, plan := range models.Plans {
		seats := models.LimitsFor(plan).Seats
		// seats-1 people already (owner included): room for one more.
		svc := New(seatsDB(id, seats-2), &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, mocks.StaticEntitlements{Plan: plan})
		if seats >= 2 {
			if _, err := svc.SetMember(owner, id, eveUser.Email, models.BoardViewer); err != nil {
				t.Errorf("%s at seats-1: %v", plan, err)
			}
		}
		// Full: the next one is refused.
		svc = New(seatsDB(id, seats-1), &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, mocks.StaticEntitlements{Plan: plan})
		if _, err := svc.SetMember(owner, id, eveUser.Email, models.BoardViewer); !isLimit(err, models.LimitSeats) {
			t.Errorf("%s at seats: %v", plan, err)
		}
	}
}

func TestSetMember_ExistingMemberKeepsTheirSeat(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{FindBoardFn: func(uuid.UUID) (*models.Board, error) {
		return &models.Board{Id: id, Owner: owner.ID, Members: []models.BoardMember{{User: eveUser.Id.String(), Role: models.BoardViewer}}}, nil
	}}
	svc := New(db, &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, mocks.StaticEntitlements{Plan: models.PlanFree})
	if _, err := svc.SetMember(owner, id, eveUser.Email, models.BoardEditor); err != nil {
		t.Errorf("a role change on a full board: %v", err)
	}
}

func TestSetMember_SeatsCountOrgMembersToo(t *testing.T) {
	id, orgID := uuid.New(), uuid.New()
	org := models.Org{Id: orgID, Owner: owner.ID}
	// Pro: 5 seats. Owner + 2 board members + 2 org-only people = 5.
	org.Members = []models.OrgMember{{User: owner.ID, Role: models.OrgRoleAdmin}, {User: "x", Role: models.OrgRoleMember}, {User: "y", Role: models.OrgRoleMember}, {User: "m1", Role: models.OrgRoleMember}}
	orgs := &mocks.MemoryOrgStore{Orgs: []models.Org{org}}
	db := &mocks.MockDB{FindBoardFn: func(uuid.UUID) (*models.Board, error) {
		return &models.Board{Id: id, Owner: owner.ID, Org: &orgID, Members: []models.BoardMember{{User: "m1", Role: models.BoardViewer}, {User: "m2", Role: models.BoardViewer}}}, nil
	}}
	svc := New(db, &purgeRecorder{}, access.New(orgs), users(), &mocks.CountingLimiter{}, orgs, mocks.StaticEntitlements{Plan: models.PlanPro})
	// distinct: owner, x, y, m1, m2 = 5 -> full.
	if _, err := svc.SetMember(owner, id, eveUser.Email, models.BoardViewer); !isLimit(err, models.LimitSeats) {
		t.Errorf("err = %v, want seats", err)
	}
}

func TestGetBoard_CarriesLimitsSeatsAndSecretsLeft(t *testing.T) {
	id := uuid.New()
	plans := mocks.StaticEntitlements{Plan: models.PlanPro, Secrets: map[string]int{owner.ID: 20}}
	svc := New(boardDB(id), &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, plans)
	board, err := svc.GetBoard(bob, id)
	if err != nil {
		t.Fatal(err)
	}
	if board.Limits == nil || *board.Limits != models.LimitsFor(models.PlanPro) || board.SeatsUsed != 3 || board.SecretsLeft == nil || *board.SecretsLeft != 5 {
		t.Errorf("board = limits %+v seats %d left %v", board.Limits, board.SeatsUsed, board.SecretsLeft)
	}
}
