package groups

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

var (
	owner    = models.Principal{ID: "owner-id"}
	member   = models.Principal{ID: "member-id"}
	stranger = models.Principal{ID: "stranger"}
)

type purgeRecorder struct {
	purged []models.SecretScope
}

func (p *purgeRecorder) Purge(scope models.SecretScope) error {
	p.purged = append(p.purged, scope)
	return nil
}

func service() (*GroupService, *mocks.MemoryGroupStore, *purgeRecorder) {
	store := &mocks.MemoryGroupStore{}
	purger := &purgeRecorder{}
	return New(store, purger, access.New(nil)), store, purger
}

func TestCreate_OwnerIsTheCaller(t *testing.T) {
	srv, store, _ := service()

	group, err := srv.Create(owner, "ops", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if group.Owner != owner.ID || group.Name != "ops" || group.Id == uuid.Nil {
		t.Errorf("group = %+v", group)
	}
	if len(store.Groups) != 1 || store.Groups[0].Id != group.Id {
		t.Errorf("stored = %+v", store.Groups)
	}
}

func TestCreate_Rejects(t *testing.T) {
	srv, _, _ := service()

	if _, err := srv.Create(models.Principal{}, "ops", nil); !errors.Is(err, ErrForbidden) {
		t.Errorf("anonymous created a group: %v", err)
	}
	if _, err := srv.Create(owner, "  ", nil); err == nil {
		t.Error("a group with no name was created")
	}
}

func TestMembers_OnlyTheOwnerChangesThem(t *testing.T) {
	srv, store, _ := service()
	group, _ := srv.Create(owner, "ops", nil)

	if err := srv.AddMember(owner, group.Id, member.ID); err != nil {
		t.Fatalf("add: %v", err)
	}
	if got, _ := store.FindGroup(group.Id); !got.IsMember(member.ID) {
		t.Errorf("member not added: %+v", got)
	}

	if err := srv.AddMember(member, group.Id, stranger.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member added someone: %v", err)
	}
	if err := srv.RemoveMember(member, group.Id, member.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member removed someone: %v", err)
	}
	if err := srv.AddMember(owner, group.Id, ""); err == nil {
		t.Error("an empty member id was accepted")
	}

	if err := srv.RemoveMember(owner, group.Id, member.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got, _ := store.FindGroup(group.Id); got.IsMember(member.ID) {
		t.Errorf("member not removed: %+v", got)
	}
}

func TestGet_MembersSeeItStrangersDoNot(t *testing.T) {
	srv, _, _ := service()
	group, _ := srv.Create(owner, "ops", nil)
	_ = srv.AddMember(owner, group.Id, member.ID)

	for _, caller := range []models.Principal{owner, member} {
		if _, err := srv.Get(caller, group.Id); err != nil {
			t.Errorf("%s: %v", caller.ID, err)
		}
	}
	if _, err := srv.Get(stranger, group.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("stranger: %v, want ErrForbidden", err)
	}
	if _, err := srv.Get(owner, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Errorf("unknown group: %v, want ErrForbidden", err)
	}
}

func TestListMine(t *testing.T) {
	srv, _, _ := service()
	ops, _ := srv.Create(owner, "ops", nil)
	_, _ = srv.Create(owner, "private", nil)
	_ = srv.AddMember(owner, ops.Id, member.ID)

	mine, err := srv.ListMine(member)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(mine) != 1 || mine[0].Id != ops.Id {
		t.Errorf("member's groups = %+v, want only ops", mine)
	}

	if mine, _ := srv.ListMine(owner); len(mine) != 2 {
		t.Errorf("owner's groups = %+v, want both", mine)
	}
}

// Deleting a group destroys what it owned: the secrets and the key that
// encrypted them.
func TestDelete_OwnerOnlyAndPurgesTheScope(t *testing.T) {
	srv, store, purger := service()
	group, _ := srv.Create(owner, "ops", nil)
	_ = srv.AddMember(owner, group.Id, member.ID)

	if err := srv.Delete(member, group.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member deleted the group: %v", err)
	}

	if err := srv.Delete(owner, group.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.FindGroup(group.Id); !errors.Is(err, infrastructure.ErrGroupNotFound) {
		t.Error("the group survived")
	}
	if len(purger.purged) != 1 || purger.purged[0] != models.GroupScope(group.Id) {
		t.Errorf("purged %v, want the group's scope", purger.purged)
	}
}

// acme has orgAdmin as admin and orgMember as member.
var (
	orgAdmin  = models.Principal{ID: "org-admin"}
	orgMember = models.Principal{ID: "org-member"}
	acme      = models.Org{Id: uuid.New(), Members: []models.OrgMember{
		{User: orgAdmin.ID, Role: models.OrgRoleAdmin},
		{User: orgMember.ID, Role: models.OrgRoleMember},
	}}
)

func acmeService() (*GroupService, *mocks.MemoryGroupStore) {
	store := &mocks.MemoryGroupStore{}
	return New(store, &purgeRecorder{}, access.New(&mocks.MemoryOrgStore{Orgs: []models.Org{acme}})), store
}

func TestCreate_InAnOrgTheCallerBelongsTo(t *testing.T) {
	srv, _ := acmeService()

	group, err := srv.Create(orgMember, "ops", &acme.Id)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if group.Org == nil || *group.Org != acme.Id || group.Owner != orgMember.ID || group.Role != models.GroupOwner {
		t.Errorf("group = %+v", group)
	}

	if _, err := srv.Create(stranger, "ops", &acme.Id); !errors.Is(err, ErrOrgNotFound) {
		t.Errorf("a stranger created a group in acme: %v", err)
	}
}

// The organization's admins own its groups; its members see them and that
// is all.
func TestOrgGroup_AdminsOwnMembersSee(t *testing.T) {
	srv, store := acmeService()
	group, _ := srv.Create(orgMember, "ops", &acme.Id)
	// Whoever created it has since left; the group stays the org's.
	store.Groups[0].Owner = "someone-who-left"

	seen, err := srv.Get(orgMember, group.Id)
	if err != nil || seen.Role != models.GroupViewer {
		t.Errorf("org member: %+v, %v, want the group as viewer", seen, err)
	}
	mine, _ := srv.ListMine(orgMember)
	if len(mine) != 1 || mine[0].Role != models.GroupViewer {
		t.Errorf("org member lists %+v", mine)
	}
	if err := srv.AddMember(orgMember, group.Id, orgMember.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("an org member added herself: %v", err)
	}

	if err := srv.AddMember(orgAdmin, group.Id, orgMember.ID); err != nil {
		t.Errorf("the org admin could not add a member: %v", err)
	}
	if seen, _ := srv.Get(orgMember, group.Id); seen.Role != models.GroupMember {
		t.Errorf("listed by the group = %q, want member", seen.Role)
	}
	if err := srv.Delete(orgAdmin, group.Id); err != nil {
		t.Errorf("the org admin could not delete the group: %v", err)
	}

	if _, err := srv.Get(stranger, group.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("a stranger saw the group: %v", err)
	}
}
