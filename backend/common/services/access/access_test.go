package access

import (
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

func TestBoardRole(t *testing.T) {
	board := &models.Board{Owner: "owner", Members: []models.BoardMember{
		{User: "ana", Role: models.BoardEditor},
		{User: "bob", Role: models.BoardViewer},
	}}

	cases := []struct {
		principal models.Principal
		want      models.BoardRole
	}{
		{models.Principal{ID: "owner"}, models.BoardOwner},
		{models.Principal{ID: "ana"}, models.BoardEditor},
		{models.Principal{ID: "bob"}, models.BoardViewer},
		{models.Principal{ID: "eve"}, ""},
		{models.Principal{ID: "root", Admin: true}, ""},
		{models.Principal{}, ""},
	}

	for _, c := range cases {
		if got := New().BoardRole(c.principal, board); got != c.want {
			t.Errorf("BoardRole(%+v) = %q, want %q", c.principal, got, c.want)
		}
	}

	if got := New().BoardRole(models.Principal{ID: "owner"}, nil); got != "" {
		t.Errorf("BoardRole on no board = %q, want none", got)
	}
}

func TestGroupRole(t *testing.T) {
	group := &models.Group{Owner: "owner", Members: []string{"ana"}}

	cases := map[string]models.GroupRole{
		"owner": models.GroupOwner,
		"ana":   models.GroupMember,
		"eve":   "",
		"":      "",
	}
	for user, want := range cases {
		if got := New().GroupRole(models.Principal{ID: user}, group); got != want {
			t.Errorf("GroupRole(%q) = %q, want %q", user, got, want)
		}
	}

	if got := New().GroupRole(models.Principal{ID: "owner"}, nil); got != "" {
		t.Errorf("GroupRole on no group = %q, want none", got)
	}
}
