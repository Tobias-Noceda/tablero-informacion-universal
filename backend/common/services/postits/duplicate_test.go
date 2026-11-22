package postits

import (
	"errors"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

// duplicateFixture is a one-off card on a board where member edits, with a
// title, a style and a position on the board.
type duplicateFixture struct {
	board, card uuid.UUID
	source      *models.PostIts
	entry       models.BoardPostIt
	db          *mocks.MockDB
	created     *models.PostIts
	createdAt   models.Position
	title       models.BoardPostItTitle
	ptype       string
	styled      *models.CardStylePatch
}

func newDuplicateFixture(cards int) *duplicateFixture {
	fixture := &duplicateFixture{board: uuid.New(), card: uuid.New()}
	fixture.source = &models.PostIts{
		Id: fixture.card, Board: fixture.board, RunAs: boardOwner.ID,
		Resource: &url.URL{Scheme: "https", Host: "example.com", Path: "/price"},
		Request:  models.Request{Method: "GET", Headers: map[string]string{"Accept": "application/json"}},
		Response: "json", Query: map[string]string{"price": ".price"}, Rate: 600,
		Params: map[string]string{"$coin": "bitcoin"},
	}
	width := float32(240)
	fixture.entry = models.BoardPostIt{
		Id: fixture.card, Type: "", Title: models.BoardPostItTitle{Text: "Price {{price}}", Vars: true},
		Position:  models.Position{X: 100, Y: 200},
		CardStyle: models.CardStyle{Color: "blue", Width: width},
	}

	fixture.db = boardOf(boardOwner, member)
	find := fixture.db.FindBoardFn
	fixture.db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
		b, _ := find(id)
		b.PostIts = append([]models.BoardPostIt{fixture.entry}, make([]models.BoardPostIt, cards-1)...)
		return b, nil
	}
	fixture.db.FindPostItFn = func(id uuid.UUID) (*models.PostIts, error) {
		if id != fixture.card {
			return nil, errors.New("not found")
		}
		copied := *fixture.source
		return &copied, nil
	}
	fixture.db.CreatePostItFn = func(p *models.PostIts, title models.BoardPostItTitle, ptype string, pos models.Position) (*models.PostIts, error) {
		p.Id = uuid.New()
		fixture.created, fixture.title, fixture.ptype, fixture.createdAt = p, title, ptype, pos
		return p, nil
	}
	fixture.db.StylePostItFn = func(_, _ uuid.UUID, patch models.CardStylePatch) (*models.CardStyle, error) {
		fixture.styled = &patch
		style := patch.Apply(models.CardStyle{})
		return &style, nil
	}
	return fixture
}

func (fixture *duplicateFixture) service(secrets *mocks.MockSecretResolver, plans mocks.StaticEntitlements) *PostItsService {
	if secrets == nil {
		secrets = &mocks.MockSecretResolver{}
	}
	return New(fixture.db, &mocks.MockCache{}, &mocks.MockExecuter{}, secrets, access.New(nil), builtins(), nil, plans, &mocks.CountingLimiter{})
}

func TestDuplicatePostIt_CopiesTheCardNextToIt(t *testing.T) {
	fixture := newDuplicateFixture(1)
	svc := fixture.service(nil, mocks.StaticEntitlements{})

	got, err := svc.DuplicatePostIt(member, fixture.card)
	if err != nil {
		t.Fatalf("DuplicatePostIt: %v", err)
	}
	if fixture.created == nil || fixture.created.Id == fixture.card {
		t.Fatalf("no new card was stored: %+v", fixture.created)
	}
	if fixture.created.Board != fixture.board || fixture.created.Query["price"] != ".price" || fixture.created.Params["$coin"] != "bitcoin" ||
		fixture.created.Resource.String() != "https://example.com/price" || fixture.created.Rate != 600 {
		t.Errorf("settings were not copied: %+v", fixture.created)
	}
	if fixture.created.RunAs != member.ID {
		t.Errorf("the copy runs as %q, want whoever duplicated it", fixture.created.RunAs)
	}
	if fixture.title != fixture.entry.Title || fixture.ptype != fixture.entry.Type {
		t.Errorf("title/type = %+v/%q, want %+v/%q", fixture.title, fixture.ptype, fixture.entry.Title, fixture.entry.Type)
	}
	if fixture.createdAt.X <= fixture.entry.Position.X || fixture.createdAt.Y <= fixture.entry.Position.Y {
		t.Errorf("the copy is at %+v, want beside %+v", fixture.createdAt, fixture.entry.Position)
	}
	if fixture.styled == nil || fixture.styled.Apply(models.CardStyle{}) != fixture.entry.CardStyle {
		t.Errorf("style patch = %+v, want the card's style", fixture.styled)
	}
	if got.Id != fixture.created.Id || got.Title != fixture.entry.Title || got.CardStyle != fixture.entry.CardStyle || got.Position != fixture.createdAt {
		t.Errorf("answered %+v", got)
	}

	fixture.created.Params["$coin"] = "ether"
	fixture.created.Request.Headers["Accept"] = "text/html"
	if fixture.source.Params["$coin"] != "bitcoin" || fixture.source.Request.Headers["Accept"] != "application/json" {
		t.Error("the copy shares its maps with the card it was made from")
	}
}

func TestDuplicatePostIt_LeavesAnUnstyledCardUnstyled(t *testing.T) {
	fixture := newDuplicateFixture(1)
	fixture.entry.CardStyle = models.CardStyle{}
	svc := fixture.service(nil, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, fixture.card); err != nil {
		t.Fatal(err)
	}
	if fixture.styled != nil {
		t.Errorf("styled an unstyled copy: %+v", fixture.styled)
	}
}

func TestDuplicatePostIt_OnlyEditors(t *testing.T) {
	viewer := models.Principal{ID: "viewer-id"}
	fixture := newDuplicateFixture(1)
	find := fixture.db.FindBoardFn
	fixture.db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
		b, err := find(id)
		b.Members = append(b.Members, models.BoardMember{User: viewer.ID, Role: models.BoardViewer})
		return b, err
	}
	svc := fixture.service(nil, mocks.StaticEntitlements{})

	for _, who := range []models.Principal{viewer, outsider} {
		if _, err := svc.DuplicatePostIt(who, fixture.card); !errors.Is(err, ErrNotAMember) {
			t.Errorf("%s: got %v, want ErrNotAMember", who.ID, err)
		}
	}
	if fixture.created != nil {
		t.Error("a card was stored")
	}
}

func TestDuplicatePostIt_CountsAgainstThePlan(t *testing.T) {
	max := models.LimitsFor(models.PlanFree).NodesPerBoard
	fixture := newDuplicateFixture(max)
	held := ""
	plans := mocks.StaticEntitlements{Plan: models.PlanFree, HoldFn: func(limit, holder string) (func(), error) {
		held = limit + ":" + holder
		return func() {}, nil
	}}
	svc := fixture.service(nil, plans)

	if _, err := svc.DuplicatePostIt(member, fixture.card); !limited(err, models.LimitNodes) {
		t.Errorf("got %v, want the nodes limit", err)
	}
	if held != models.LimitNodes+":board:"+fixture.board.String() {
		t.Errorf("held %q, want the board's nodes turn", held)
	}
	if fixture.created != nil {
		t.Error("a card was stored past the limit")
	}
}

func TestDuplicatePostIt_ChecksTheCredentialsAsWhoeverDuplicates(t *testing.T) {
	fixture := newDuplicateFixture(1)
	theirs := models.SecretRef{Scope: models.MemberScope(fixture.board, boardOwner.ID), Name: "KEY"}
	fixture.source.Bindings = map[string]models.SecretRef{"KEY": theirs}
	fixture.source.Request.Headers["Authorization"] = "$KEY"
	refused := errors.New("not yours")
	var asked models.Principal
	secrets := &mocks.MockSecretResolver{CanBindFn: func(p models.Principal, _ uuid.UUID, _ []models.SecretRef) error {
		asked = p
		return refused
	}}
	svc := fixture.service(secrets, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, fixture.card); !errors.Is(err, refused) {
		t.Errorf("got %v, want the binding refused", err)
	}
	if asked != member {
		t.Errorf("bindings checked as %+v, want the member duplicating", asked)
	}
	if fixture.created != nil {
		t.Error("a card was stored with a credential its new runner may not use")
	}
}

func TestDuplicatePostIt_RefusesACardTypeNoLongerOffered(t *testing.T) {
	fixture := newDuplicateFixture(1)
	fixture.source.WellKnown = "no_longer_there"
	svc := fixture.service(nil, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, fixture.card); !errors.Is(err, ErrUnknownWellKnown) {
		t.Errorf("got %v, want ErrUnknownWellKnown", err)
	}
}
