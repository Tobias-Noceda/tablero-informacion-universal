package boards

import (
	"errors"
	"slices"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden   = infrastructure.ErrForbidden
	ErrOrgNotFound = infrastructure.ErrOrgNotFound
	ErrInvalidRole = errors.New("A member is an editor or a viewer")
	ErrOwnerRole   = errors.New("The owner's role cannot change")
	ErrRateLimited = infrastructure.ErrRateLimited

	ErrStrandNotFound = infrastructure.ErrStrandNotFound
	ErrInvalidStrand  = models.ErrInvalidStrand
)

// Adding someone by email tells the caller whether they have an account, so
// each user gets a bounded number of lookups (shared with organizations).
const (
	lookupLimit  = 30
	lookupWindow = 10 * time.Minute
)

type BoardService struct {
	db      infrastructure.Database
	secrets infrastructure.ScopePurger
	access  infrastructure.Access
	users   infrastructure.UserReader
	limiter infrastructure.RateLimiter
	// orgs is where an organization board's members are read from, to count
	// who reaches it; plans is what its payer's plan allows.
	orgs  infrastructure.OrgReader
	plans infrastructure.Entitlements
}

func New(db infrastructure.Database, secrets infrastructure.ScopePurger, access infrastructure.Access, users infrastructure.UserReader, limiter infrastructure.RateLimiter, orgs infrastructure.OrgReader, plans infrastructure.Entitlements) *BoardService {
	return &BoardService{db, secrets, access, users, limiter, orgs, plans}
}

// payer is the plan the board runs on and the organization it belongs to,
// nil for a personal board or when the organization is gone (the owner pays
// then).
func (srv *BoardService) payer(board *models.Board) (models.Entitlement, *models.Org, error) {
	if board.Org != nil && srv.orgs != nil {
		org, err := srv.orgs.FindOrg(*board.Org)
		if err == nil {
			plan, err := srv.plans.ForOrg(org)
			return plan, org, err
		}
		if !errors.Is(err, infrastructure.ErrOrgNotFound) {
			return models.Entitlement{}, nil, err
		}
	}

	plan, err := srv.plans.ForUser(board.Owner)
	return plan, nil, err
}

// describe fills in what the board's payer's plan allows and how much of it
// is taken, for the one board a response is about.
func (srv *BoardService) describe(board *models.Board) error {
	plan, org, err := srv.payer(board)
	if err != nil {
		return err
	}
	left, err := srv.plans.SecretsLeft(plan)
	if err != nil {
		return err
	}

	board.Limits = &plan.Limits
	board.SeatsUsed = len(board.People(org))
	board.SecretsLeft = &left
	return nil
}

// require loads a board on which the principal holds at least min, with the
// principal's role filled in. A board that cannot be found is
// indistinguishable from one the principal may not see.
func (srv *BoardService) require(principal models.Principal, id uuid.UUID, min models.BoardRole) (*models.Board, error) {
	board, err := srv.db.FindBoard(id)
	if err != nil || board == nil {
		return nil, ErrForbidden
	}

	board.Role = srv.access.BoardRole(principal, board)
	if !board.Role.AtLeast(min) {
		return nil, ErrForbidden
	}
	return board, nil
}

func (srv *BoardService) GetUserBoards(principal models.Principal) ([]models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	orgs, err := srv.access.Orgs(principal)
	if err != nil {
		return nil, err
	}
	boards, err := srv.db.FindUserBoards(principal.ID, orgs)
	if err != nil {
		return nil, err
	}
	for i := range boards {
		boards[i].Role = srv.access.BoardRole(principal, &boards[i])
	}
	return boards, nil
}

// GetBoard is the board with the caller's role and what its payer's plan
// allows on it.
func (srv *BoardService) GetBoard(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	board, err := srv.require(principal, id, models.BoardViewer)
	if err != nil {
		return nil, err
	}
	if err := srv.describe(board); err != nil {
		return nil, err
	}
	return board, nil
}

// CreateBoard makes the caller the board's owner. A board created in an
// organization the caller belongs to is also reachable by its members. A
// space (the caller's own, or each organization) holds as many boards as its
// payer's plan allows.
func (srv *BoardService) CreateBoard(principal models.Principal, name string, org *uuid.UUID) (*models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}
	if org != nil && srv.access.OrgRole(principal, *org) == "" {
		return nil, ErrOrgNotFound
	}

	space := &models.Board{Owner: principal.ID, Org: org}
	plan, _, err := srv.payer(space)
	if err != nil {
		return nil, err
	}
	// Count and create in the space's turn, or two requests could both take
	// the last board the plan allows.
	holder := "user:" + principal.ID
	if org != nil {
		holder = "org:" + org.String()
	}
	release, err := srv.plans.Hold(models.LimitBoards, holder)
	if err != nil {
		return nil, err
	}
	defer release()
	count := srv.db.CountPersonalBoards
	if org != nil {
		count = func(string) (int64, error) { return srv.db.CountOrgBoards(*org) }
	}
	boards, err := count(principal.ID)
	if err != nil {
		return nil, err
	}
	if err := plan.Admits(models.LimitBoards, int(boards), plan.Limits.BoardsPerSpace); err != nil {
		return nil, err
	}

	board, err := srv.db.CreateBoard(name, principal.ID, org)
	if err != nil {
		return nil, err
	}
	board.Role = models.BoardOwner
	if err := srv.describe(board); err != nil {
		return nil, err
	}
	return board, nil
}

func (srv *BoardService) DeleteBoard(principal models.Principal, id uuid.UUID) error {
	board, err := srv.require(principal, id, models.BoardOwner)
	if err != nil {
		return err
	}

	if err := srv.db.DeleteBoard(id); err != nil {
		return err
	}

	if err := srv.secrets.Purge(models.BoardScope(id)); err != nil {
		return err
	}

	for _, user := range board.Everyone() {
		if err := srv.secrets.Purge(models.MemberScope(id, user)); err != nil {
			return err
		}
	}

	return nil
}

func (srv *BoardService) GetBoardPostIts(principal models.Principal, id uuid.UUID) ([]models.PostIts, error) {
	if _, err := srv.require(principal, id, models.BoardViewer); err != nil {
		return nil, err
	}
	return srv.db.FindBoardPostIts(id)
}

// Members lists everyone on the board, the owner first.
func (srv *BoardService) Members(principal models.Principal, id uuid.UUID) ([]models.BoardMemberSummary, error) {
	board, err := srv.require(principal, id, models.BoardViewer)
	if err != nil {
		return nil, err
	}

	ids := board.Everyone()
	found, err := srv.users.FindUsers(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.UserSummary, len(found))
	for _, user := range found {
		byID[user.Id.String()] = user.Summary()
	}

	members := make([]models.BoardMemberSummary, 0, len(ids))
	for _, user := range ids {
		summary, ok := byID[user]
		if !ok {
			// An account that no longer exists still shows up, by id, so the
			// owner can remove it.
			summary.Id, _ = uuid.Parse(user)
		}
		members = append(members, models.BoardMemberSummary{User: summary, Role: board.ExplicitRole(user)})
	}
	return members, nil
}

// SetMember adds the account registered under email, or changes its role.
// Only someone who does not reach the board yet takes a seat: a new role for
// a member, or listing someone the organization already brings, is free.
func (srv *BoardService) SetMember(principal models.Principal, id uuid.UUID, email string, role models.BoardRole) (*models.BoardMemberSummary, error) {
	board, err := srv.require(principal, id, models.BoardOwner)
	if err != nil {
		return nil, err
	}
	if !role.Assignable() {
		return nil, ErrInvalidRole
	}
	if err := srv.allowLookup(principal); err != nil {
		return nil, err
	}

	user, err := srv.users.FindUserByEmail(models.NormalizeEmail(email))
	if err != nil {
		return nil, err
	}
	if user.Id.String() == board.Owner {
		return nil, ErrOwnerRole
	}

	// The seats are counted on the board as it is in this board's turn: the
	// one read above may miss someone added meanwhile.
	release, err := srv.plans.Hold(models.LimitSeats, "board:"+id.String())
	if err != nil {
		return nil, err
	}
	defer release()
	if board, err = srv.db.FindBoard(id); err != nil || board == nil {
		return nil, ErrForbidden
	}
	plan, org, err := srv.payer(board)
	if err != nil {
		return nil, err
	}
	if people := board.People(org); !slices.Contains(people, user.Id.String()) {
		if err := plan.Admits(models.LimitSeats, len(people), plan.Limits.Seats); err != nil {
			return nil, err
		}
	}
	// On a plan that shares read-only, nobody new becomes an editor; someone
	// who already edits keeps it until the owner changes it.
	if plan.Limits.ViewersOnly && role != models.BoardViewer && board.ExplicitRole(user.Id.String()) != role {
		return nil, plan.Refuse(models.LimitEditors, 0)
	}

	if err := srv.db.SetBoardMember(id, user.Id.String(), role); err != nil {
		return nil, err
	}
	return &models.BoardMemberSummary{User: user.Summary(), Role: role}, nil
}

func (srv *BoardService) allowLookup(principal models.Principal) error {
	allowed, err := srv.limiter.Allow("member-lookup:"+principal.ID, lookupLimit, lookupWindow)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrRateLimited
	}
	return nil
}

// RemoveMember is the owner's to decide, except that anyone may leave a board
// on their own. What the member kept on the board goes with them.
func (srv *BoardService) RemoveMember(principal models.Principal, id uuid.UUID, user string) error {
	board, err := srv.require(principal, id, models.BoardViewer)
	if err != nil {
		return err
	}
	if board.Role != models.BoardOwner && user != principal.ID {
		return ErrForbidden
	}
	if user == board.Owner {
		return ErrOwnerRole
	}

	if err := srv.db.RemoveBoardMember(id, user); err != nil {
		return err
	}

	return srv.secrets.Purge(models.MemberScope(id, user))
}

func (srv *BoardService) UpdateBoardName(principal models.Principal, id uuid.UUID, name string) error {
	if _, err := srv.require(principal, id, models.BoardOwner); err != nil {
		return err
	}
	return srv.db.UpdateBoardName(id, name)
}

// ConnectPostIts draws a strand from source to target with the given style.
func (srv *BoardService) ConnectPostIts(principal models.Principal, boardID, source, target uuid.UUID, style models.StrandStyle) (*models.Strand, error) {
	if _, err := srv.require(principal, boardID, models.BoardEditor); err != nil {
		return nil, err
	}
	style, err := style.Normalized()
	if err != nil {
		return nil, err
	}
	return srv.db.ConnectPostIts(boardID, models.Strand{Source: source, Target: target, StrandStyle: style})
}

// UpdateStrand changes part of a strand's style. A patch that changes
// nothing answers the strand as it is.
func (srv *BoardService) UpdateStrand(principal models.Principal, boardID, strandID uuid.UUID, patch models.StrandPatch) (*models.Strand, error) {
	board, err := srv.require(principal, boardID, models.BoardEditor)
	if err != nil {
		return nil, err
	}
	patch, err = patch.Normalized()
	if err != nil {
		return nil, err
	}
	if !patch.Empty() {
		return srv.db.UpdateStrand(boardID, strandID, patch)
	}
	for _, strand := range board.Strands {
		if strand.Id == strandID {
			return &strand, nil
		}
	}
	return nil, ErrStrandNotFound
}

func (srv *BoardService) DisconnectPostIts(principal models.Principal, boardID, strandID uuid.UUID) error {
	if _, err := srv.require(principal, boardID, models.BoardEditor); err != nil {
		return err
	}
	return srv.db.DisconnectPostIts(boardID, strandID)
}
