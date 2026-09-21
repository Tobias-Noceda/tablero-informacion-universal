package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestScopeKey(t *testing.T) {
	board := uuid.MustParse("6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d")

	cases := []struct {
		scope SecretScope
		want  string
	}{
		{BoardScope(board), "board:6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"},
		{UserScope("cognito-123"), "user:cognito-123"},
		{MemberScope(board, "cognito-123"), "member:6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d:cognito-123"},
		{GroupScope(board), "group:6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"},
		{SystemScope, "system"},
	}

	for _, c := range cases {
		if got := c.scope.Key(); got != c.want {
			t.Errorf("%+v.Key() = %q, want %q", c.scope, got, c.want)
		}
	}
}

// The same owner string under two kinds must never share a key, or a user
// whose id happens to equal a board id could reach the board's secrets.
func TestScopeKey_KindsDoNotCollide(t *testing.T) {
	owner := uuid.New().String()

	if BoardScope(uuid.MustParse(owner)).Key() == UserScope(owner).Key() {
		t.Error("board and user scopes with the same owner produced the same key")
	}
}

func TestScopeValid(t *testing.T) {
	cases := []struct {
		label string
		scope SecretScope
		want  bool
	}{
		{"board", BoardScope(uuid.New()), true},
		{"user", UserScope("cognito-123"), true},
		{"system", SystemScope, true},
		{"member", MemberScope(uuid.New(), "cognito-123"), true},
		{"group", GroupScope(uuid.New()), true},
		{"group with non uuid owner", SecretScope{Kind: ScopeGroup, Owner: "nope"}, false},
		{"member without user", SecretScope{Kind: ScopeMember, Owner: uuid.New().String() + ":"}, false},
		{"member with non uuid board", SecretScope{Kind: ScopeMember, Owner: "nope:cognito-123"}, false},
		{"member without separator", SecretScope{Kind: ScopeMember, Owner: uuid.New().String()}, false},
		{"board without owner", SecretScope{Kind: ScopeBoard}, false},
		{"board with non uuid owner", SecretScope{Kind: ScopeBoard, Owner: "nope"}, false},
		{"user without owner", SecretScope{Kind: ScopeUser}, false},
		{"system with owner", SecretScope{Kind: ScopeSystem, Owner: "x"}, false},
		{"unknown kind", SecretScope{Kind: "team", Owner: "x"}, false},
	}

	for _, c := range cases {
		if got := c.scope.Valid(); got != c.want {
			t.Errorf("%s: Valid() = %v, want %v", c.label, got, c.want)
		}
	}
}

func TestMemberScope_Member(t *testing.T) {
	board := uuid.New()

	gotBoard, gotUser, ok := MemberScope(board, "cognito-123").Member()
	if !ok || gotBoard != board || gotUser != "cognito-123" {
		t.Errorf("Member() = (%v, %q, %v), want (%v, %q, true)", gotBoard, gotUser, ok, board, "cognito-123")
	}

	if _, _, ok := BoardScope(board).Member(); ok {
		t.Error("a board scope reported itself as a member scope")
	}
}
