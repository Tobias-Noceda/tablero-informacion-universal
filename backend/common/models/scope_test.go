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
