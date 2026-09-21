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

func TestMongo_GroupsRoundTrip(t *testing.T) {
	db := integrationDB(t)

	group := &models.Group{Id: uuid.New(), Name: "ops", Owner: "owner", Members: []string{}, CreatedAt: time.Now().UTC()}
	if err := db.CreateGroup(group); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := db.AddGroupMember(group.Id, "ana"); err != nil {
		t.Fatalf("add member: %v", err)
	}
	if err := db.AddGroupMember(group.Id, "ana"); err != nil {
		t.Fatalf("adding twice must be idempotent: %v", err)
	}

	found, err := db.FindGroup(group.Id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Name != "ops" || len(found.Members) != 1 || found.Members[0] != "ana" {
		t.Errorf("found = %+v", found)
	}

	for _, user := range []string{"owner", "ana"} {
		groups, err := db.FindUserGroups(user)
		if err != nil {
			t.Fatalf("user groups: %v", err)
		}
		if len(groups) != 1 || groups[0].Id != group.Id {
			t.Errorf("%s belongs to %v, want the group", user, groups)
		}
	}
	if groups, _ := db.FindUserGroups("eve"); len(groups) != 0 {
		t.Errorf("eve belongs to %v, want nothing", groups)
	}

	if err := db.RemoveGroupMember(group.Id, "ana"); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if groups, _ := db.FindUserGroups("ana"); len(groups) != 0 {
		t.Errorf("ana still belongs to %v", groups)
	}

	if err := db.DeleteGroup(group.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.FindGroup(group.Id); !errors.Is(err, infrastructure.ErrGroupNotFound) {
		t.Errorf("after delete: %v, want ErrGroupNotFound", err)
	}
	if err := db.AddGroupMember(group.Id, "ana"); !errors.Is(err, infrastructure.ErrGroupNotFound) {
		t.Errorf("update of a missing group: %v, want ErrGroupNotFound", err)
	}
}
