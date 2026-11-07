package customwellknowns

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

var (
	ana   = models.Principal{ID: "ana"}   // owns the board
	bob   = models.Principal{ID: "bob"}   // board editor
	cy    = models.Principal{ID: "cy"}    // board viewer
	eve   = models.Principal{ID: "eve"}   // nobody
	admin = models.Principal{ID: "admin"} // org admin
	mem   = models.Principal{ID: "mem"}   // org member
	token = models.Principal{ID: "ana", Token: true}
)

type fixture struct {
	srv   *Service
	store *mocks.MemoryCustomWellKnownStore
	board *models.Board
	org   uuid.UUID
}

func setup(t *testing.T) *fixture {
	t.Helper()
	org := uuid.New()
	board := &models.Board{
		Id:    uuid.New(),
		Owner: ana.ID,
		Org:   &org,
		Members: []models.BoardMember{
			{User: bob.ID, Role: models.BoardEditor},
			{User: cy.ID, Role: models.BoardViewer},
		},
	}
	orgs := &mocks.MemoryOrgStore{Orgs: []models.Org{{Id: org, Members: []models.OrgMember{
		{User: admin.ID, Role: models.OrgRoleAdmin},
		{User: mem.ID, Role: models.OrgRoleMember},
	}}}}
	db := &mocks.MockDB{FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
		if id == board.Id {
			b := *board
			return &b, nil
		}
		return nil, errors.New("no board")
	}}
	store := mocks.NewMemoryCustomWellKnownStore()
	return &fixture{New(store, db, access.New(orgs)), store, board, org}
}

func (f *fixture) scope(kind models.CustomWellKnownScopeKind) models.CustomWellKnownScope {
	switch kind {
	case models.CustomScopeBoard:
		return models.CustomWellKnownScope{Kind: kind, Owner: f.board.Id.String()}
	case models.CustomScopeOrg:
		return models.CustomWellKnownScope{Kind: kind, Owner: f.org.String()}
	}
	return models.CustomWellKnownScope{Kind: kind, Owner: ana.ID}
}

func body() models.CustomWellKnown {
	return models.CustomWellKnown{
		Name:     "Cat fact",
		Title:    models.BoardPostItTitle{Text: "{{fact}}", Vars: true},
		Resource: "https://catfact.ninja/fact",
		Response: "json",
		Query:    map[string]string{"fact": ".fact"},
		Rate:     60,
		Outputs:  []models.CustomWellKnownOutput{{Key: "fact", Label: "Fact"}},
	}
}

func TestPermissions(t *testing.T) {
	cases := []struct {
		kind         models.CustomWellKnownScopeKind
		who          models.Principal
		read, manage bool
	}{
		{models.CustomScopeUser, ana, true, true},
		{models.CustomScopeUser, bob, false, false},
		{models.CustomScopeUser, token, true, false},
		{models.CustomScopeBoard, ana, true, true},
		{models.CustomScopeBoard, bob, true, true},
		{models.CustomScopeBoard, cy, true, false},
		{models.CustomScopeBoard, mem, true, false}, // an org member is a viewer
		{models.CustomScopeBoard, eve, false, false},
		{models.CustomScopeOrg, admin, true, true},
		{models.CustomScopeOrg, mem, true, false},
		{models.CustomScopeOrg, ana, false, false}, // on the board, not in the org
		{models.CustomScopeOrg, eve, false, false},
	}
	for _, c := range cases {
		f := setup(t)
		scope := f.scope(c.kind)

		_, err := f.srv.List(c.who, scope)
		if (err == nil) != c.read {
			t.Errorf("%s/%s list: err = %v, want allowed %v", c.kind, c.who.ID, err, c.read)
		}

		created, err := f.srv.Create(c.who, scope, body())
		if (err == nil) != c.manage {
			t.Errorf("%s/%s create: err = %v, want allowed %v", c.kind, c.who.ID, err, c.manage)
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			t.Errorf("%s/%s create: err = %v, want not_found", c.kind, c.who.ID, err)
		}

		// One someone allowed made, to edit, read and delete.
		if created == nil {
			owner := map[models.CustomWellKnownScopeKind]models.Principal{models.CustomScopeUser: ana, models.CustomScopeBoard: ana, models.CustomScopeOrg: admin}[c.kind]
			if created, err = f.srv.Create(owner, scope, body()); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := f.srv.Get(c.who, created.Key); (err == nil) != c.read {
			t.Errorf("%s/%s get: err = %v, want allowed %v", c.kind, c.who.ID, err, c.read)
		}
		if _, err := f.srv.Update(c.who, created.Key, body()); (err == nil) != c.manage {
			t.Errorf("%s/%s update: err = %v, want allowed %v", c.kind, c.who.ID, err, c.manage)
		}
		if err := f.srv.Delete(c.who, created.Key); (err == nil) != c.manage {
			t.Errorf("%s/%s delete: err = %v, want allowed %v", c.kind, c.who.ID, err, c.manage)
		}
	}
}

func TestCreate_StampsAndValidates(t *testing.T) {
	f := setup(t)
	scope := f.scope(models.CustomScopeUser)

	in := body()
	in.Key = "custom_mine"
	in.Scope = f.scope(models.CustomScopeBoard)
	c, err := f.srv.Create(ana, scope, in)
	if err != nil {
		t.Fatal(err)
	}
	if !models.IsCustomWellKnownKey(c.Key) || c.Key == "custom_mine" || c.Scope != scope {
		t.Errorf("key/scope come from the server: %+v", c)
	}
	if c.CreatedBy != ana.ID || c.UpdatedBy != ana.ID || c.Request.Method != "GET" {
		t.Errorf("stamps: %+v", c)
	}

	bad := body()
	bad.Resource = ""
	if _, err := f.srv.Create(ana, scope, bad); !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want invalid", err)
	}

	edited := body()
	edited.Name = "Renamed"
	edited.Scope = f.scope(models.CustomScopeBoard)
	updated, err := f.srv.Update(ana, c.Key, edited)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed" || updated.Scope != scope || updated.CreatedAt != c.CreatedAt {
		t.Errorf("update keeps key, scope and creation: %+v", updated)
	}
}

func TestCreate_Limit(t *testing.T) {
	f := setup(t)
	scope := f.scope(models.CustomScopeUser)
	for range Limit {
		if _, err := f.srv.Create(ana, scope, body()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.srv.Create(ana, scope, body()); !errors.Is(err, ErrLimit) {
		t.Errorf("err = %v, want limit_reached", err)
	}
}

func TestForBoardAndUsable(t *testing.T) {
	f := setup(t)

	mine, _ := f.srv.Create(bob, models.CustomWellKnownScope{Kind: models.CustomScopeUser, Owner: bob.ID}, body())
	boards, _ := f.srv.Create(bob, f.scope(models.CustomScopeBoard), body())
	orgs, _ := f.srv.Create(admin, f.scope(models.CustomScopeOrg), body())
	anas, _ := f.srv.Create(ana, f.scope(models.CustomScopeUser), body())
	unused, _ := f.srv.Create(ana, f.scope(models.CustomScopeUser), body())

	// A card ana made from her own type is on the board.
	f.board.PostIts = []models.BoardPostIt{{Type: anas.Key}, {Type: "dog_facts"}}

	list, err := f.srv.ForBoard(bob, f.board.Id)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range list {
		got[c.Key] = c.Usable
	}
	want := map[string]bool{mine.Key: true, boards.Key: true, orgs.Key: true, anas.Key: false}
	if len(got) != len(want) {
		t.Errorf("listed %v, want %v", got, want)
	}
	for key, usable := range want {
		if u, ok := got[key]; !ok || u != usable {
			t.Errorf("%s: listed %v usable %v, want usable %v", key, ok, u, usable)
		}
	}
	if _, ok := got[unused.Key]; ok {
		t.Error("someone else's unused type leaked")
	}

	if _, err := f.srv.ForBoard(eve, f.board.Id); !errors.Is(err, ErrNotFound) {
		t.Errorf("stranger: err = %v", err)
	}

	for _, key := range []string{mine.Key, boards.Key, orgs.Key} {
		if wk, err := f.srv.Usable(bob, f.board.Id, key); err != nil || wk.Key != key {
			t.Errorf("bob uses %s: %v", key, err)
		}
	}
	if _, err := f.srv.Usable(bob, f.board.Id, anas.Key); !errors.Is(err, ErrNotFound) {
		t.Errorf("bob uses ana's own type: err = %v", err)
	}
	if _, err := f.srv.Usable(bob, uuid.New(), boards.Key); !errors.Is(err, ErrNotFound) {
		t.Errorf("board type on another board: err = %v", err)
	}
}
