package customwellknowns

import (
	"errors"
	"fmt"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	// ErrNotFound also answers whoever may not see a scope or a card type:
	// to a stranger they do not exist.
	ErrNotFound = errors.New("not_found")
	ErrInvalid  = errors.New("invalid_well_known")
	ErrLimit    = errors.New("limit_reached")
)

// Limit is how many card types one scope may hold.
const Limit = 50

type boardFinder interface {
	FindBoard(id uuid.UUID) (*models.Board, error)
}

// BoardWellKnown is a custom card type as a board sees it. Usable ones are
// offered for new cards; the rest are only there because a card on the
// board was made from them (someone else's own type), so it can be shown
// and its settings edited.
type BoardWellKnown struct {
	models.CustomWellKnown
	Usable bool `json:"usable"`
}

// Service keeps the card types users define for themselves (user scope),
// for a board (its editors manage them) or for an organization (its admins
// manage them). Every role comes from infrastructure.Access.
type Service struct {
	store  infrastructure.CustomWellKnownStore
	boards boardFinder
	access infrastructure.Access
	now    func() time.Time
}

func New(store infrastructure.CustomWellKnownStore, boards boardFinder, access infrastructure.Access) *Service {
	return &Service{store: store, boards: boards, access: access, now: time.Now}
}

func (srv *Service) List(principal models.Principal, scope models.CustomWellKnownScope) ([]models.CustomWellKnown, error) {
	if !srv.allowed(principal, scope, false) {
		return nil, ErrNotFound
	}
	return srv.store.ListCustomWellKnowns(scope)
}

func (srv *Service) Get(principal models.Principal, key string) (*models.CustomWellKnown, error) {
	c, err := srv.find(key)
	if err != nil {
		return nil, err
	}
	if !srv.allowed(principal, c.Scope, false) {
		return nil, ErrNotFound
	}
	return c, nil
}

// Create stores a new card type in scope; the key is generated here.
func (srv *Service) Create(principal models.Principal, scope models.CustomWellKnownScope, body models.CustomWellKnown) (*models.CustomWellKnown, error) {
	if !srv.allowed(principal, scope, true) {
		return nil, ErrNotFound
	}

	count, err := srv.store.CountCustomWellKnowns(scope)
	if err != nil {
		return nil, err
	}
	if count >= Limit {
		return nil, ErrLimit
	}

	now := srv.now().UTC()
	body.Key = models.NewCustomWellKnownKey()
	body.Scope = scope
	body.CreatedBy, body.CreatedAt = principal.ID, now
	return srv.save(principal, body, now)
}

// Update replaces a card type's definition; its key and scope stay. Cards
// already made keep the copy they were created with.
func (srv *Service) Update(principal models.Principal, key string, body models.CustomWellKnown) (*models.CustomWellKnown, error) {
	existing, err := srv.find(key)
	if err != nil {
		return nil, err
	}
	if !srv.allowed(principal, existing.Scope, true) {
		return nil, ErrNotFound
	}

	body.Key = existing.Key
	body.Scope = existing.Scope
	body.CreatedBy, body.CreatedAt = existing.CreatedBy, existing.CreatedAt
	return srv.save(principal, body, srv.now().UTC())
}

// Delete removes a card type for good. Cards made from it keep running on
// their own copy, they just cannot be edited any more.
func (srv *Service) Delete(principal models.Principal, key string) error {
	existing, err := srv.find(key)
	if err != nil {
		return err
	}
	if !srv.allowed(principal, existing.Scope, true) {
		return ErrNotFound
	}

	if err := srv.store.DeleteCustomWellKnown(key); err != nil {
		if errors.Is(err, infrastructure.ErrCustomWellKnownNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// ForBoard is every custom card type a board needs: the ones the caller may
// drop on it (theirs, the board's, the board's organization's) and the ones
// its cards were made from.
func (srv *Service) ForBoard(principal models.Principal, boardID uuid.UUID) ([]BoardWellKnown, error) {
	board, err := srv.board(principal, boardID, models.BoardViewer)
	if err != nil {
		return nil, err
	}

	usable, err := srv.store.ListCustomWellKnowns(usableScopes(principal, board)...)
	if err != nil {
		return nil, err
	}

	result := make([]BoardWellKnown, 0, len(usable))
	seen := make(map[string]bool, len(usable))
	for _, c := range usable {
		result = append(result, BoardWellKnown{c, true})
		seen[c.Key] = true
	}

	var referenced []string
	for _, postit := range board.PostIts {
		if models.IsCustomWellKnownKey(postit.Type) && !seen[postit.Type] {
			referenced = append(referenced, postit.Type)
			seen[postit.Type] = true
		}
	}
	others, err := srv.store.FindCustomWellKnowns(referenced)
	if err != nil {
		return nil, err
	}
	for _, c := range others {
		result = append(result, BoardWellKnown{c, false})
	}

	return result, nil
}

// Usable is the definition a new card on boardID is made from, if the
// principal may use it there. Whether they may edit the board is the
// post-its service's call.
func (srv *Service) Usable(principal models.Principal, boardID uuid.UUID, key string) (*models.WellKnown, error) {
	c, err := srv.find(key)
	if err != nil {
		return nil, err
	}

	board, err := srv.boards.FindBoard(boardID)
	if err != nil || board == nil {
		return nil, ErrNotFound
	}
	for _, scope := range usableScopes(principal, board) {
		if c.Scope == scope {
			wk := c.AsWellKnown()
			return &wk, nil
		}
	}
	return nil, ErrNotFound
}

func usableScopes(principal models.Principal, board *models.Board) []models.CustomWellKnownScope {
	scopes := []models.CustomWellKnownScope{
		{Kind: models.CustomScopeUser, Owner: principal.ID},
		{Kind: models.CustomScopeBoard, Owner: board.Id.String()},
	}
	if board.Org != nil {
		scopes = append(scopes, models.CustomWellKnownScope{Kind: models.CustomScopeOrg, Owner: board.Org.String()})
	}
	return scopes
}

func (srv *Service) save(principal models.Principal, body models.CustomWellKnown, now time.Time) (*models.CustomWellKnown, error) {
	if body.Request.Method == "" {
		body.Request.Method = "GET"
	}
	body.UpdatedBy, body.UpdatedAt = principal.ID, now

	if err := body.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := srv.store.PutCustomWellKnown(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (srv *Service) find(key string) (*models.CustomWellKnown, error) {
	if !models.IsCustomWellKnownKey(key) {
		return nil, ErrNotFound
	}
	c, err := srv.store.FindCustomWellKnown(key)
	if errors.Is(err, infrastructure.ErrCustomWellKnownNotFound) {
		return nil, ErrNotFound
	}
	return c, err
}

// allowed: a user's own scope is theirs alone; a board's is seen by anyone
// on it and managed by its editors; an organization's is seen by its
// members and managed by its admins. Access tokens only read.
func (srv *Service) allowed(principal models.Principal, scope models.CustomWellKnownScope, manage bool) bool {
	if principal.Anonymous() || (manage && principal.Token) {
		return false
	}

	switch scope.Kind {
	case models.CustomScopeUser:
		return scope.Owner == principal.ID
	case models.CustomScopeBoard:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return false
		}
		min := models.BoardViewer
		if manage {
			min = models.BoardEditor
		}
		_, err = srv.board(principal, id, min)
		return err == nil
	case models.CustomScopeOrg:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return false
		}
		role := srv.access.OrgRole(principal, id)
		if manage {
			return role == models.OrgRoleAdmin
		}
		return role != ""
	}
	return false
}

func (srv *Service) board(principal models.Principal, id uuid.UUID, min models.BoardRole) (*models.Board, error) {
	board, err := srv.boards.FindBoard(id)
	if err != nil || board == nil {
		return nil, ErrNotFound
	}
	if !srv.access.BoardRole(principal, board).AtLeast(min) {
		return nil, ErrNotFound
	}
	return board, nil
}
