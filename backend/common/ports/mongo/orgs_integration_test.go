//go:build integration

package mongo

import (
	"errors"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func newOrg(t *testing.T, db *MongoDB, admin string) *models.Org {
	t.Helper()
	org := &models.Org{
		Id:        uuid.New(),
		Name:      "acme",
		Members:   []models.OrgMember{{User: admin, Role: models.OrgRoleAdmin}},
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := db.CreateOrg(org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	return org
}

func TestMongo_OrgsRoundTrip(t *testing.T) {
	db := integrationDB(t)
	org := newOrg(t, db, "ana")

	if err := db.SetOrgMember(org.Id, "bob", models.OrgRoleMember); err != nil {
		t.Fatalf("add bob: %v", err)
	}
	if err := db.SetOrgMember(org.Id, "bob", models.OrgRoleAdmin); err != nil {
		t.Fatalf("promote bob: %v", err)
	}
	if err := db.RenameOrg(org.Id, "acme inc"); err != nil {
		t.Fatalf("rename: %v", err)
	}

	found, err := db.FindOrg(org.Id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Name != "acme inc" || len(found.Members) != 2 || found.RoleOf("ana") != models.OrgRoleAdmin || found.RoleOf("bob") != models.OrgRoleAdmin {
		t.Errorf("found = %+v (changing a role must not add a second entry)", found)
	}

	for _, user := range []string{"ana", "bob"} {
		orgs, err := db.FindUserOrgs(user)
		if err != nil {
			t.Fatalf("orgs of %s: %v", user, err)
		}
		if len(orgs) != 1 || orgs[0].Id != org.Id {
			t.Errorf("%s's orgs = %+v, want the org", user, orgs)
		}
	}
	if orgs, _ := db.FindUserOrgs("eve"); len(orgs) != 0 {
		t.Errorf("eve's orgs = %+v, want none", orgs)
	}

	if err := db.RemoveOrgMember(org.Id, "bob"); err != nil {
		t.Fatalf("remove bob: %v", err)
	}
	if orgs, _ := db.FindUserOrgs("bob"); len(orgs) != 0 {
		t.Errorf("bob still in %+v", orgs)
	}

	if err := db.DeleteOrg(org.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.FindOrg(org.Id); !errors.Is(err, infrastructure.ErrOrgNotFound) {
		t.Errorf("after delete: %v, want ErrOrgNotFound", err)
	}
	for name, err := range map[string]error{
		"rename": db.RenameOrg(org.Id, "x"),
		"set":    db.SetOrgMember(org.Id, "bob", models.OrgRoleMember),
		"remove": db.RemoveOrgMember(org.Id, "bob"),
		"delete": db.DeleteOrg(org.Id),
	} {
		if !errors.Is(err, infrastructure.ErrOrgNotFound) {
			t.Errorf("%s on a missing org: %v, want ErrOrgNotFound", name, err)
		}
	}
}

// The store itself keeps one admin, so two admins demoting each other at the
// same time cannot leave the organization without one.
func TestMongo_OrgKeepsAnAdmin(t *testing.T) {
	db := integrationDB(t)
	org := newOrg(t, db, "ana")
	_ = db.SetOrgMember(org.Id, "bob", models.OrgRoleMember)

	if err := db.SetOrgMember(org.Id, "ana", models.OrgRoleMember); !errors.Is(err, infrastructure.ErrLastOrgAdmin) {
		t.Errorf("demoting the last admin: %v, want ErrLastOrgAdmin", err)
	}
	if err := db.RemoveOrgMember(org.Id, "ana"); !errors.Is(err, infrastructure.ErrLastOrgAdmin) {
		t.Errorf("removing the last admin: %v, want ErrLastOrgAdmin", err)
	}

	_ = db.SetOrgMember(org.Id, "bob", models.OrgRoleAdmin)
	if err := db.SetOrgMember(org.Id, "ana", models.OrgRoleMember); err != nil {
		t.Fatalf("demoting one of two admins: %v", err)
	}
	if err := db.RemoveOrgMember(org.Id, "ana"); err != nil {
		t.Fatalf("removing a member: %v", err)
	}

	found, _ := db.FindOrg(org.Id)
	if len(found.Members) != 1 || found.RoleOf("bob") != models.OrgRoleAdmin {
		t.Errorf("members = %+v, want bob as the only admin", found.Members)
	}
}

func TestMongo_OrgBoardsAndGroups(t *testing.T) {
	db := integrationDB(t)
	org := newOrg(t, db, "ana")

	orgBoard, err := db.CreateBoard("org board", "ana", &org.Id)
	if err != nil {
		t.Fatalf("create org board: %v", err)
	}
	personal, _ := db.CreateBoard("personal", "bob", nil)
	group := &models.Group{Id: uuid.New(), Name: "ops", Owner: "ana", Members: []string{}, Org: &org.Id, CreatedAt: time.Now().UTC()}
	if err := db.CreateGroup(group); err != nil {
		t.Fatalf("create group: %v", err)
	}

	found, _ := db.FindBoard(orgBoard.Id)
	if found.Org == nil || *found.Org != org.Id {
		t.Errorf("board org = %v, want %s", found.Org, org.Id)
	}

	// bob reaches the org's board and group through the org, and keeps his own.
	boards, err := db.FindUserBoards("bob", []uuid.UUID{org.Id})
	if err != nil {
		t.Fatalf("boards: %v", err)
	}
	ids := map[uuid.UUID]bool{}
	for _, board := range boards {
		ids[board.Id] = true
	}
	if len(boards) != 2 || !ids[orgBoard.Id] || !ids[personal.Id] {
		t.Errorf("bob's boards = %+v, want the org's and his own", boards)
	}
	if boards, _ := db.FindUserBoards("bob", nil); len(boards) != 1 || boards[0].Id != personal.Id {
		t.Errorf("bob outside the org lists %+v", boards)
	}

	groups, _ := db.FindUserGroups("bob", []uuid.UUID{org.Id})
	if len(groups) != 1 || groups[0].Id != group.Id || groups[0].Org == nil {
		t.Errorf("bob's groups = %+v, want the org's", groups)
	}
	if groups, _ := db.FindUserGroups("bob", nil); len(groups) != 0 {
		t.Errorf("bob outside the org lists %+v", groups)
	}

	for name, count := range map[string]func(uuid.UUID) (int64, error){"boards": db.CountOrgBoards, "groups": db.CountOrgGroups} {
		n, err := count(org.Id)
		if err != nil || n != 1 {
			t.Errorf("count %s = %d, %v, want 1", name, n, err)
		}
		if n, _ := count(uuid.New()); n != 0 {
			t.Errorf("count %s of another org = %d", name, n)
		}
	}
}

func TestMongo_OrgIndexes(t *testing.T) {
	db := integrationDB(t)

	for collection, want := range map[string][]string{
		"orgs":   {"members.user_1"},
		"boards": {"org_1"},
		"groups": {"org_1"},
	} {
		names := indexNames(t, db.client.Database(db.boards.Database().Name()).Collection(collection))
		for _, index := range want {
			if !names[index] {
				t.Errorf("%s: index %s missing, have %v", collection, index, names)
			}
		}
	}
}
