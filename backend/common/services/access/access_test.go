package access

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// acme has ana as admin and bob as member; eve is in no organization.
var acme = models.Org{Id: uuid.New(), Name: "acme", Members: []models.OrgMember{
	{User: "ana", Role: models.OrgRoleAdmin},
	{User: "bob", Role: models.OrgRoleMember},
}}

func withAcme() Resolver {
	return New(&mocks.MemoryOrgStore{Orgs: []models.Org{acme}})
}

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
		if got := withAcme().BoardRole(c.principal, board); got != c.want {
			t.Errorf("BoardRole(%+v) = %q, want %q", c.principal, got, c.want)
		}
	}

	if got := withAcme().BoardRole(models.Principal{ID: "owner"}, nil); got != "" {
		t.Errorf("BoardRole on no board = %q, want none", got)
	}
}

// On an organization's board its admins act as owners and its members as
// viewers; a role the board gives explicitly counts when it is higher.
func TestBoardRole_ThroughTheOrg(t *testing.T) {
	board := &models.Board{Owner: "owner", Org: &acme.Id, Members: []models.BoardMember{
		{User: "cy", Role: models.BoardEditor},
	}}

	cases := map[string]models.BoardRole{
		"owner": models.BoardOwner,
		"ana":   models.BoardOwner,
		"bob":   models.BoardViewer,
		"cy":    models.BoardEditor,
		"eve":   "",
	}
	for user, want := range cases {
		if got := withAcme().BoardRole(models.Principal{ID: user}, board); got != want {
			t.Errorf("BoardRole(%q) = %q, want %q", user, got, want)
		}
	}

	// bob promoted on the board is an editor there, and ana stays an owner
	// even if the board lists her lower.
	board.Members = append(board.Members,
		models.BoardMember{User: "bob", Role: models.BoardEditor},
		models.BoardMember{User: "ana", Role: models.BoardViewer})
	if got := withAcme().BoardRole(models.Principal{ID: "bob"}, board); got != models.BoardEditor {
		t.Errorf("promoted member = %q, want editor", got)
	}
	if got := withAcme().BoardRole(models.Principal{ID: "ana"}, board); got != models.BoardOwner {
		t.Errorf("admin listed as viewer = %q, want owner", got)
	}
}

func TestBoardRole_PersonalBoardIgnoresOrgs(t *testing.T) {
	board := &models.Board{Owner: "owner"}

	if got := withAcme().BoardRole(models.Principal{ID: "ana"}, board); got != "" {
		t.Errorf("an org admin on a personal board = %q, want none", got)
	}
}

func TestBoardRole_OrgThatIsGone(t *testing.T) {
	gone := uuid.New()
	board := &models.Board{Owner: "owner", Org: &gone}

	if got := withAcme().BoardRole(models.Principal{ID: "owner"}, board); got != models.BoardOwner {
		t.Errorf("owner = %q, want owner", got)
	}
	if got := withAcme().BoardRole(models.Principal{ID: "ana"}, board); got != "" {
		t.Errorf("ana = %q, want none", got)
	}
	if got := New(nil).BoardRole(models.Principal{ID: "ana"}, board); got != "" {
		t.Errorf("without organizations ana = %q, want none", got)
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
		if got := withAcme().GroupRole(models.Principal{ID: user}, group); got != want {
			t.Errorf("GroupRole(%q) = %q, want %q", user, got, want)
		}
	}

	if got := withAcme().GroupRole(models.Principal{ID: "owner"}, nil); got != "" {
		t.Errorf("GroupRole on no group = %q, want none", got)
	}
}

// An organization's admins own its groups; its members only see them, the
// group's secrets stay with the people the group lists.
func TestGroupRole_ThroughTheOrg(t *testing.T) {
	group := &models.Group{Owner: "owner", Members: []string{"cy"}, Org: &acme.Id}

	cases := map[string]models.GroupRole{
		"owner": models.GroupOwner,
		"ana":   models.GroupOwner,
		"bob":   models.GroupViewer,
		"cy":    models.GroupMember,
		"eve":   "",
	}
	for user, want := range cases {
		if got := withAcme().GroupRole(models.Principal{ID: user}, group); got != want {
			t.Errorf("GroupRole(%q) = %q, want %q", user, got, want)
		}
	}

	group.Members = append(group.Members, "bob")
	if got := withAcme().GroupRole(models.Principal{ID: "bob"}, group); got != models.GroupMember {
		t.Errorf("an org member listed by the group = %q, want member", got)
	}
}

func TestOrgRole(t *testing.T) {
	cases := map[string]models.OrgRole{"ana": models.OrgRoleAdmin, "bob": models.OrgRoleMember, "eve": "", "": ""}
	for user, want := range cases {
		if got := withAcme().OrgRole(models.Principal{ID: user}, acme.Id); got != want {
			t.Errorf("OrgRole(%q) = %q, want %q", user, got, want)
		}
	}

	if got := withAcme().OrgRole(models.Principal{ID: "ana"}, uuid.New()); got != "" {
		t.Errorf("OrgRole in an org that does not exist = %q", got)
	}
}

func TestOrgs(t *testing.T) {
	orgs, err := withAcme().Orgs(models.Principal{ID: "bob"})
	if err != nil || len(orgs) != 1 || orgs[0] != acme.Id {
		t.Errorf("bob's orgs = %v, %v, want acme", orgs, err)
	}

	for _, principal := range []models.Principal{{ID: "eve"}, {}} {
		if orgs, err := withAcme().Orgs(principal); err != nil || len(orgs) != 0 {
			t.Errorf("Orgs(%+v) = %v, %v, want none", principal, orgs, err)
		}
	}
	if orgs, err := New(nil).Orgs(models.Principal{ID: "bob"}); err != nil || len(orgs) != 0 {
		t.Errorf("without organizations = %v, %v", orgs, err)
	}
}
