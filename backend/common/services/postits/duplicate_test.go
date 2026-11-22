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
	f := &duplicateFixture{board: uuid.New(), card: uuid.New()}
	f.source = &models.PostIts{
		Id: f.card, Board: f.board, RunAs: boardOwner.ID,
		Resource: &url.URL{Scheme: "https", Host: "example.com", Path: "/price"},
		Request:  models.Request{Method: "GET", Headers: map[string]string{"Accept": "application/json"}},
		Response: "json", Query: map[string]string{"price": ".price"}, Rate: 600,
		Params: map[string]string{"$coin": "bitcoin"},
	}
	width := float32(240)
	f.entry = models.BoardPostIt{
		Id: f.card, Type: "", Title: models.BoardPostItTitle{Text: "Price {{price}}", Vars: true},
		Position:  models.Position{X: 100, Y: 200},
		CardStyle: models.CardStyle{Color: "blue", Width: width},
	}

	f.db = boardOf(boardOwner, member)
	find := f.db.FindBoardFn
	f.db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
		b, _ := find(id)
		b.PostIts = append([]models.BoardPostIt{f.entry}, make([]models.BoardPostIt, cards-1)...)
		return b, nil
	}
	f.db.FindPostItFn = func(id uuid.UUID) (*models.PostIts, error) {
		if id != f.card {
			return nil, errors.New("not found")
		}
		copied := *f.source
		return &copied, nil
	}
	f.db.CreatePostItFn = func(p *models.PostIts, title models.BoardPostItTitle, ptype string, pos models.Position) (*models.PostIts, error) {
		p.Id = uuid.New()
		f.created, f.title, f.ptype, f.createdAt = p, title, ptype, pos
		return p, nil
	}
	f.db.StylePostItFn = func(_, _ uuid.UUID, patch models.CardStylePatch) (*models.CardStyle, error) {
		f.styled = &patch
		style := patch.Apply(models.CardStyle{})
		return &style, nil
	}
	return f
}

func (f *duplicateFixture) service(secrets *mocks.MockSecretResolver, plans mocks.StaticEntitlements) *PostItsService {
	if secrets == nil {
		secrets = &mocks.MockSecretResolver{}
	}
	return New(f.db, &mocks.MockCache{}, &mocks.MockExecuter{}, secrets, access.New(nil), builtins(), nil, plans, &mocks.CountingLimiter{})
}

func TestDuplicatePostIt_CopiesTheCardNextToIt(t *testing.T) {
	f := newDuplicateFixture(1)
	svc := f.service(nil, mocks.StaticEntitlements{})

	got, err := svc.DuplicatePostIt(member, f.card)
	if err != nil {
		t.Fatalf("DuplicatePostIt: %v", err)
	}
	if f.created == nil || f.created.Id == f.card {
		t.Fatalf("no new card was stored: %+v", f.created)
	}
	if f.created.Board != f.board || f.created.Query["price"] != ".price" || f.created.Params["$coin"] != "bitcoin" ||
		f.created.Resource.String() != "https://example.com/price" || f.created.Rate != 600 {
		t.Errorf("settings were not copied: %+v", f.created)
	}
	if f.created.RunAs != member.ID {
		t.Errorf("the copy runs as %q, want whoever duplicated it", f.created.RunAs)
	}
	if f.title != f.entry.Title || f.ptype != f.entry.Type {
		t.Errorf("title/type = %+v/%q, want %+v/%q", f.title, f.ptype, f.entry.Title, f.entry.Type)
	}
	if f.createdAt.X <= f.entry.Position.X || f.createdAt.Y <= f.entry.Position.Y {
		t.Errorf("the copy is at %+v, want beside %+v", f.createdAt, f.entry.Position)
	}
	if f.styled == nil || f.styled.Apply(models.CardStyle{}) != f.entry.CardStyle {
		t.Errorf("style patch = %+v, want the card's style", f.styled)
	}
	if got.Id != f.created.Id || got.Title != f.entry.Title || got.CardStyle != f.entry.CardStyle || got.Position != f.createdAt {
		t.Errorf("answered %+v", got)
	}

	f.created.Params["$coin"] = "ether"
	f.created.Request.Headers["Accept"] = "text/html"
	if f.source.Params["$coin"] != "bitcoin" || f.source.Request.Headers["Accept"] != "application/json" {
		t.Error("the copy shares its maps with the card it was made from")
	}
}

func TestDuplicatePostIt_LeavesAnUnstyledCardUnstyled(t *testing.T) {
	f := newDuplicateFixture(1)
	f.entry.CardStyle = models.CardStyle{}
	svc := f.service(nil, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, f.card); err != nil {
		t.Fatal(err)
	}
	if f.styled != nil {
		t.Errorf("styled an unstyled copy: %+v", f.styled)
	}
}

func TestDuplicatePostIt_OnlyEditors(t *testing.T) {
	viewer := models.Principal{ID: "viewer-id"}
	f := newDuplicateFixture(1)
	find := f.db.FindBoardFn
	f.db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
		b, err := find(id)
		b.Members = append(b.Members, models.BoardMember{User: viewer.ID, Role: models.BoardViewer})
		return b, err
	}
	svc := f.service(nil, mocks.StaticEntitlements{})

	for _, who := range []models.Principal{viewer, outsider} {
		if _, err := svc.DuplicatePostIt(who, f.card); !errors.Is(err, ErrNotAMember) {
			t.Errorf("%s: got %v, want ErrNotAMember", who.ID, err)
		}
	}
	if f.created != nil {
		t.Error("a card was stored")
	}
}

func TestDuplicatePostIt_CountsAgainstThePlan(t *testing.T) {
	max := models.LimitsFor(models.PlanFree).NodesPerBoard
	f := newDuplicateFixture(max)
	held := ""
	plans := mocks.StaticEntitlements{Plan: models.PlanFree, HoldFn: func(limit, holder string) (func(), error) {
		held = limit + ":" + holder
		return func() {}, nil
	}}
	svc := f.service(nil, plans)

	if _, err := svc.DuplicatePostIt(member, f.card); !limited(err, models.LimitNodes) {
		t.Errorf("got %v, want the nodes limit", err)
	}
	if held != models.LimitNodes+":board:"+f.board.String() {
		t.Errorf("held %q, want the board's nodes turn", held)
	}
	if f.created != nil {
		t.Error("a card was stored past the limit")
	}
}

func TestDuplicatePostIt_ChecksTheCredentialsAsWhoeverDuplicates(t *testing.T) {
	f := newDuplicateFixture(1)
	theirs := models.SecretRef{Scope: models.MemberScope(f.board, boardOwner.ID), Name: "KEY"}
	f.source.Bindings = map[string]models.SecretRef{"KEY": theirs}
	f.source.Request.Headers["Authorization"] = "$KEY"
	refused := errors.New("not yours")
	var asked models.Principal
	secrets := &mocks.MockSecretResolver{CanBindFn: func(p models.Principal, _ uuid.UUID, _ []models.SecretRef) error {
		asked = p
		return refused
	}}
	svc := f.service(secrets, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, f.card); !errors.Is(err, refused) {
		t.Errorf("got %v, want the binding refused", err)
	}
	if asked != member {
		t.Errorf("bindings checked as %+v, want the member duplicating", asked)
	}
	if f.created != nil {
		t.Error("a card was stored with a credential its new runner may not use")
	}
}

func TestDuplicatePostIt_RefusesACardTypeNoLongerOffered(t *testing.T) {
	f := newDuplicateFixture(1)
	f.source.WellKnown = "no_longer_there"
	svc := f.service(nil, mocks.StaticEntitlements{})

	if _, err := svc.DuplicatePostIt(member, f.card); !errors.Is(err, ErrUnknownWellKnown) {
		t.Errorf("got %v, want ErrUnknownWellKnown", err)
	}
}
