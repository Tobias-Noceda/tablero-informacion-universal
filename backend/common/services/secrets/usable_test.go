package secrets

import (
	"errors"
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
