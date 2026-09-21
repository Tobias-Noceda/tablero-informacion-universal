package models

import "testing"

func TestBoardIsMember(t *testing.T) {
	board := Board{Owner: "owner", Collaborators: []string{"ana", "bob"}}

	cases := []struct {
		user string
		want bool
	}{
		{"owner", true},
		{"ana", true},
		{"bob", true},
		{"eve", false},
		{"", false},
	}

	for _, c := range cases {
		if got := board.IsMember(c.user); got != c.want {
			t.Errorf("IsMember(%q) = %v, want %v", c.user, got, c.want)
		}
	}
}
