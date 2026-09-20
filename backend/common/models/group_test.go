package models

import "testing"

func TestGroupIsMember(t *testing.T) {
	group := Group{Owner: "owner", Members: []string{"ana"}}

	cases := []struct {
		user string
		want bool
	}{
		{"owner", true},
		{"ana", true},
		{"eve", false},
		{"", false},
	}

	for _, c := range cases {
		if got := group.IsMember(c.user); got != c.want {
			t.Errorf("IsMember(%q) = %v, want %v", c.user, got, c.want)
		}
	}
}
