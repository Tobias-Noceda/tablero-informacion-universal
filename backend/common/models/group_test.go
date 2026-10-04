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

func TestGroupRole_AtLeast(t *testing.T) {
	cases := []struct {
		role, min GroupRole
		want      bool
	}{
		{GroupOwner, GroupMember, true},
		{GroupMember, GroupOwner, false},
		{GroupMember, GroupMember, true},
		{GroupViewer, GroupMember, false},
		{GroupViewer, GroupViewer, true},
		{"", GroupViewer, false},
	}

	for _, c := range cases {
		if got := c.role.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}
