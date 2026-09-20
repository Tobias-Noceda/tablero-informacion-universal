package secrets

import (
	"errors"
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func put(t *testing.T, srv *SecretsService, scope models.SecretScope, principal models.Principal, name, value string) models.SecretRef {
	t.Helper()
	if err := srv.Put(scope, principal, name, models.SecretApiKey, value); err != nil {
		t.Fatalf("put %s/%s: %v", scope.Key(), name, err)
	}
	return models.SecretRef{Scope: scope, Name: name}
}

func TestResolveAs_FollowsRefsAcrossScopes(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	shared := put(t, srv, models.BoardScope(board), owner, "SHARED", "board-value")
	mine := put(t, srv, models.MemberScope(board, collaborator.ID), collaborator, "MINE", "member-value")
	profile := put(t, srv, models.UserScope(collaborator.ID), collaborator, "PROFILE", "user-value")

	resolved, err := srv.ResolveAs(collaborator, board, []models.SecretRef{shared, mine, profile})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	want := map[models.SecretRef]string{shared: "board-value", mine: "member-value", profile: "user-value"}
	for ref, value := range want {
		if resolved[ref] != value {
			t.Errorf("%s/%s = %q, want %q", ref.Scope.Key(), ref.Name, resolved[ref], value)
		}
	}
}

func TestResolveAs_ForbidsWhatThePrincipalMayNotUse(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	mine := put(t, srv, models.MemberScope(board, collaborator.ID), collaborator, "MINE", "member-value")

	if _, err := srv.ResolveAs(owner, board, []models.SecretRef{mine}); !errors.Is(err, ErrForbidden) {
		t.Errorf("the board owner resolved a member's private secret: %v", err)
	}
}

// A board token nobody stored is not an error: the card sends it verbatim,
// as it always did.
func TestResolveAs_SkipsMissingRefs(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	resolved, err := srv.ResolveAs(owner, board, []models.SecretRef{{Scope: models.BoardScope(board), Name: "NOPE"}})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("got %v, want nothing", resolved)
	}
}

func TestCanBind(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	mine := put(t, srv, models.MemberScope(board, collaborator.ID), collaborator, "MINE", "v")
	missing := models.SecretRef{Scope: models.BoardScope(board), Name: "NOPE"}

	if err := srv.CanBind(collaborator, board, []models.SecretRef{mine}); err != nil {
		t.Errorf("the member cannot bind their own secret: %v", err)
	}
	if err := srv.CanBind(owner, board, []models.SecretRef{mine}); !errors.Is(err, ErrForbidden) {
		t.Errorf("the board owner bound a member's private secret: %v", err)
	}
	if err := srv.CanBind(collaborator, board, []models.SecretRef{missing}); !errors.Is(err, ErrUnknownCredential) {
		t.Errorf("bound a secret that does not exist: %v", err)
	}
	if err := srv.CanBind(collaborator, board, nil); err != nil {
		t.Errorf("nothing to bind is not an error: %v", err)
	}
}

func TestListUsable_IsWhatThePrincipalCanBindOnThatBoard(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()
	otherBoard := uuid.New()

	put(t, srv, models.BoardScope(board), owner, "SHARED", "v")
	put(t, srv, models.MemberScope(board, collaborator.ID), collaborator, "MINE", "v")
	put(t, srv, models.UserScope(collaborator.ID), collaborator, "PROFILE", "v")
	put(t, srv, models.MemberScope(board, owner.ID), owner, "OWNERS", "v")
	put(t, srv, models.BoardScope(otherBoard), owner, "ELSEWHERE", "v")

	usable, err := srv.ListUsable(collaborator, board)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	got := map[string]models.ScopeKind{}
	for _, meta := range usable {
		got[meta.Name] = meta.Scope.Kind
	}
	want := map[string]models.ScopeKind{
		"SHARED":  models.ScopeBoard,
		"MINE":    models.ScopeMember,
		"PROFILE": models.ScopeUser,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("%s: kind %q, want %q", name, got[name], kind)
		}
	}
}

func TestListUsable_RequiresMembership(t *testing.T) {
	srv := service(t, newStore())

	if _, err := srv.ListUsable(stranger, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestListUsable_IncludesTheGroupsThePrincipalBelongsTo(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	ops := models.Group{Id: uuid.New(), Name: "ops", Owner: owner.ID, Members: []string{collaborator.ID}}
	private := models.Group{Id: uuid.New(), Name: "private", Owner: owner.ID}
	store.groups.Groups = []models.Group{ops, private}

	put(t, srv, models.GroupScope(ops.Id), owner, "OPS_KEY", "v")
	put(t, srv, models.GroupScope(private.Id), owner, "PRIVATE_KEY", "v")

	usable, err := srv.ListUsable(collaborator, board)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(usable) != 1 || usable[0].Name != "OPS_KEY" || usable[0].Scope != models.GroupScope(ops.Id) {
		t.Errorf("usable = %+v, want only the ops key", usable)
	}
}

func TestSetGrants(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	alice := models.Principal{ID: "alice"}
	mine := put(t, srv, models.UserScope(alice.ID), alice, "KEY", "v")
	share := []models.Grant{{To: models.Audience{Kind: models.AudienceUser, ID: collaborator.ID}}}

	if err := srv.SetGrants(mine.Scope, alice, "KEY", share); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := store.rows[0].Grants; len(got) != 1 || got[0] != share[0] {
		t.Errorf("stored grants = %v, want %v", got, share)
	}

	if err := srv.SetGrants(mine.Scope, collaborator, "KEY", nil); !errors.Is(err, ErrForbidden) {
		t.Errorf("someone else changed the grants: %v", err)
	}
	if err := srv.SetGrants(mine.Scope, alice, "NOPE", share); !errors.Is(err, ErrUnknownCredential) {
		t.Errorf("granting a missing secret: %v, want ErrUnknownCredential", err)
	}
	if err := srv.SetGrants(mine.Scope, alice, "KEY", []models.Grant{{To: models.Audience{Kind: "org", ID: "x"}}}); err == nil {
		t.Error("an invalid grant was accepted")
	}

	put(t, srv, models.SystemScope, models.Principal{}, "PLATFORM", "v")
	if err := srv.SetGrants(models.SystemScope, models.Principal{}, "PLATFORM", share); err == nil {
		t.Error("a system secret was shared")
	}

	if err := srv.SetGrants(mine.Scope, alice, "KEY", nil); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if got := store.rows[0].Grants; len(got) != 0 {
		t.Errorf("grants after revoke = %v", got)
	}
}

func TestListUsable_IncludesWhatWasSharedWithThePrincipal(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()
	otherBoard := uuid.New()
	alice := models.Principal{ID: "alice"}
	ops := models.Group{Id: uuid.New(), Name: "ops", Owner: owner.ID, Members: []string{collaborator.ID}}
	store.groups.Groups = []models.Group{ops}

	anywhere := put(t, srv, models.UserScope(alice.ID), alice, "ANYWHERE", "v")
	here := put(t, srv, models.UserScope(alice.ID), alice, "HERE_ONLY", "v")
	viaGroup := put(t, srv, models.UserScope(alice.ID), alice, "VIA_GROUP", "v")
	viaBoard := put(t, srv, models.UserScope(alice.ID), alice, "VIA_BOARD", "v")
	put(t, srv, models.UserScope(alice.ID), alice, "NOT_SHARED", "v")

	user := models.Audience{Kind: models.AudienceUser, ID: collaborator.ID}
	_ = srv.SetGrants(anywhere.Scope, alice, anywhere.Name, []models.Grant{{To: user}})
	_ = srv.SetGrants(here.Scope, alice, here.Name, []models.Grant{{To: user, Board: board.String()}})
	_ = srv.SetGrants(viaGroup.Scope, alice, viaGroup.Name, []models.Grant{{To: models.Audience{Kind: models.AudienceGroup, ID: ops.Id.String()}}})
	_ = srv.SetGrants(viaBoard.Scope, alice, viaBoard.Name, []models.Grant{{To: models.Audience{Kind: models.AudienceBoard, ID: board.String()}}})

	names := func(board uuid.UUID) []string {
		t.Helper()
		usable, err := srv.ListUsable(collaborator, board)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		var out []string
		for _, meta := range usable {
			out = append(out, meta.Name)
		}
		slices.Sort(out)
		return out
	}

	if got := names(board); !slices.Equal(got, []string{"ANYWHERE", "HERE_ONLY", "VIA_BOARD", "VIA_GROUP"}) {
		t.Errorf("on the board: %v", got)
	}
	if got := names(otherBoard); !slices.Equal(got, []string{"ANYWHERE", "VIA_GROUP"}) {
		t.Errorf("on another board: %v", got)
	}
}

// A secret the principal already reaches through its scope is listed once,
// grant or no grant.
func TestListUsable_DoesNotRepeatASecretGrantedToItsOwnAudience(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	shared := put(t, srv, models.BoardScope(board), owner, "SHARED", "v")
	_ = srv.SetGrants(shared.Scope, owner, shared.Name, []models.Grant{{To: models.Audience{Kind: models.AudienceUser, ID: collaborator.ID}}})

	usable, err := srv.ListUsable(collaborator, board)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(usable) != 1 {
		t.Errorf("usable = %+v, want SHARED once", usable)
	}
}
