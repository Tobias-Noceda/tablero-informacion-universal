package models

import "testing"

func TestBoardRole_AtLeast(t *testing.T) {
	cases := []struct {
		role, min BoardRole
		want      bool
	}{
		{BoardOwner, BoardOwner, true},
		{BoardOwner, BoardEditor, true},
		{BoardOwner, BoardViewer, true},
		{BoardEditor, BoardOwner, false},
		{BoardEditor, BoardEditor, true},
		{BoardEditor, BoardViewer, true},
		{BoardViewer, BoardEditor, false},
		{BoardViewer, BoardViewer, true},
		{"", BoardViewer, false},
		{"admin", BoardViewer, false},
	}

	for _, c := range cases {
		if got := c.role.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}

// Only editor and viewer can be handed out; the owner is whoever made the board.
func TestBoardRole_Assignable(t *testing.T) {
	for role, want := range map[BoardRole]bool{BoardEditor: true, BoardViewer: true, BoardOwner: false, "": false, "admin": false} {
		if got := role.Assignable(); got != want {
			t.Errorf("%q.Assignable() = %v, want %v", role, got, want)
		}
	}
}

func TestBoard_ExplicitRole(t *testing.T) {
	board := Board{Owner: "owner", Members: []BoardMember{{User: "ana", Role: BoardEditor}, {User: "bob", Role: BoardViewer}}}

	cases := map[string]BoardRole{
		"owner": BoardOwner,
		"ana":   BoardEditor,
		"bob":   BoardViewer,
		"eve":   "",
		"":      "",
	}
	for user, want := range cases {
		if got := board.ExplicitRole(user); got != want {
			t.Errorf("ExplicitRole(%q) = %q, want %q", user, got, want)
		}
	}
}

func TestBoard_Everyone(t *testing.T) {
	board := Board{Owner: "owner", Members: []BoardMember{{User: "ana", Role: BoardEditor}}}
	if got := board.Everyone(); len(got) != 2 || got[0] != "owner" || got[1] != "ana" {
		t.Errorf("Everyone() = %v, want [owner ana]", got)
	}
}
