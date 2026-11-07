package postittemplates

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
	ErrInvalid  = errors.New("invalid_postit_template")
	ErrLimit    = errors.New("limit_reached")
)

// Limit is how many card types one scope may hold.
const Limit = 50

type boardFinder interface {
	FindBoard(id uuid.UUID) (*models.Board, error)
}

// BoardPostitTemplate is a post-it template as a board sees it. Usable ones are
// offered for new cards; the rest are only there because a card on the
// board was made from them (someone else's own type), so it can be shown
// and its settings edited.
type BoardPostitTemplate struct {
	models.PostitTemplate
	Usable bool `json:"usable"`
}

// Service keeps the card types users define for themselves (user scope),
// for a board (its editors manage them) or for an organization (its admins
// manage them). Every role comes from infrastructure.Access.
type Service struct {
	store  infrastructure.PostitTemplateStore
	boards boardFinder
	access infrastructure.Access
	now    func() time.Time
}

func New(store infrastructure.PostitTemplateStore, boards boardFinder, access infrastructure.Access) *Service {
	return &Service{store: store, boards: boards, access: access, now: time.Now}
}

func (srv *Service) List(principal models.Principal, scope models.PostitTemplateScope) ([]models.PostitTemplate, error) {
	if !srv.allowed(principal, scope, false) {
		return nil, ErrNotFound
	}
	return srv.store.ListPostitTemplates(scope)
}

func (srv *Service) Get(principal models.Principal, key string) (*models.PostitTemplate, error) {
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
func (srv *Service) Create(principal models.Principal, scope models.PostitTemplateScope, body models.PostitTemplate) (*models.PostitTemplate, error) {
	if !srv.allowed(principal, scope, true) {
		return nil, ErrNotFound
	}

	count, err := srv.store.CountPostitTemplates(scope)
	if err != nil {
		return nil, err
	}
	if count >= Limit {
		return nil, ErrLimit
	}

	now := srv.now().UTC()
	body.Key = models.NewPostitTemplateKey()
	body.Scope = scope
	body.CreatedBy, body.CreatedAt = principal.ID, now
	return srv.save(principal, body, now)
}

// Update replaces a card type's definition; its key and scope stay. Cards
// already made keep the copy they were created with.
func (srv *Service) Update(principal models.Principal, key string, body models.PostitTemplate) (*models.PostitTemplate, error) {
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

	if err := srv.store.DeletePostitTemplate(key); err != nil {
		if errors.Is(err, infrastructure.ErrPostitTemplateNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// ForBoard is every post-it template a board needs: the ones the caller may
// drop on it (theirs, the board's, the board's organization's) and the ones
// its cards were made from.
func (srv *Service) ForBoard(principal models.Principal, boardID uuid.UUID) ([]BoardPostitTemplate, error) {
	board, err := srv.board(principal, boardID, models.BoardViewer)
	if err != nil {
		return nil, err
	}

	usable, err := srv.store.ListPostitTemplates(usableScopes(principal, board)...)
	if err != nil {
		return nil, err
	}

	result := make([]BoardPostitTemplate, 0, len(usable))
	seen := make(map[string]bool, len(usable))
	for _, c := range usable {
		result = append(result, BoardPostitTemplate{c, true})
		seen[c.Key] = true
	}

	var referenced []string
	for _, postit := range board.PostIts {
		if models.IsPostitTemplateKey(postit.Type) && !seen[postit.Type] {
			referenced = append(referenced, postit.Type)
			seen[postit.Type] = true
		}
	}
	others, err := srv.store.FindPostitTemplates(referenced)
	if err != nil {
		return nil, err
	}
	for _, c := range others {
		result = append(result, BoardPostitTemplate{c, false})
	}

	return result, nil
}

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

func usableScopes(principal models.Principal, board *models.Board) []models.PostitTemplateScope {
	scopes := []models.PostitTemplateScope{
		{Kind: models.TemplateScopeUser, Owner: principal.ID},
		{Kind: models.TemplateScopeBoard, Owner: board.Id.String()},
	}
	if board.Org != nil {
		scopes = append(scopes, models.PostitTemplateScope{Kind: models.TemplateScopeOrg, Owner: board.Org.String()})
	}
	return scopes
}

func (srv *Service) save(principal models.Principal, body models.PostitTemplate, now time.Time) (*models.PostitTemplate, error) {
	if body.Request.Method == "" {
		body.Request.Method = "GET"
	}
	body.UpdatedBy, body.UpdatedAt = principal.ID, now

	if err := body.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := srv.store.PutPostitTemplate(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

func (srv *Service) find(key string) (*models.PostitTemplate, error) {
	if !models.IsPostitTemplateKey(key) {
		return nil, ErrNotFound
	}
	c, err := srv.store.FindPostitTemplate(key)
	if errors.Is(err, infrastructure.ErrPostitTemplateNotFound) {
		return nil, ErrNotFound
	}
	return c, err
}

// allowed: a user's own scope is theirs alone; a board's is seen by anyone
// on it and managed by its editors; an organization's is seen by its
// members and managed by its admins. Access tokens only read.
func (srv *Service) allowed(principal models.Principal, scope models.PostitTemplateScope, manage bool) bool {
	if principal.Anonymous() || (manage && principal.Token) {
		return false
	}

	switch scope.Kind {
	case models.TemplateScopeUser:
		return scope.Owner == principal.ID
	case models.TemplateScopeBoard:
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
	case models.TemplateScopeOrg:
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
