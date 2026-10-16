package models

import "testing"

func TestOrg_RoleOf(t *testing.T) {
	org := Org{Members: []OrgMember{{User: "ana", Role: OrgRoleAdmin}, {User: "bob", Role: OrgRoleMember}}}

	for user, want := range map[string]OrgRole{"ana": OrgRoleAdmin, "bob": OrgRoleMember, "eve": "", "": ""} {
		if got := org.RoleOf(user); got != want {
			t.Errorf("RoleOf(%q) = %q, want %q", user, got, want)
		}
	}
}

func TestOrg_Admins(t *testing.T) {
	org := Org{Members: []OrgMember{{User: "ana", Role: OrgRoleAdmin}, {User: "bob", Role: OrgRoleMember}, {User: "cy", Role: OrgRoleAdmin}}}

	if got := org.Admins(); got != 2 {
		t.Errorf("Admins() = %d, want 2", got)
	}
}

func TestOrgRole_Valid(t *testing.T) {
	for role, want := range map[OrgRole]bool{OrgRoleAdmin: true, OrgRoleMember: true, "": false, "owner": false} {
		if got := role.Valid(); got != want {
			t.Errorf("%q.Valid() = %v, want %v", role, got, want)
		}
	}
}

func TestOrg_UserIDs(t *testing.T) {
	org := Org{Members: []OrgMember{{User: "ana", Role: OrgRoleAdmin}, {User: "bob", Role: OrgRoleMember}}}

	got := org.UserIDs()
	if len(got) != 2 || got[0] != "ana" || got[1] != "bob" {
		t.Errorf("UserIDs() = %v", got)
	}
}
