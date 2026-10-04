package orgs

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	anaUser = models.User{Id: uuid.New(), Name: "Ana", Email: "ana@example.com"}
	bobUser = models.User{Id: uuid.New(), Name: "Bob", Email: "bob@example.com"}
	cyUser  = models.User{Id: uuid.New(), Name: "Cy", Email: "cy@example.com"}
	eveUser = models.User{Id: uuid.New(), Name: "Eve", Email: "eve@example.com"}

	ana = anaUser.Principal()
	bob = bobUser.Principal()
	cy  = cyUser.Principal()
	eve = eveUser.Principal()
)

func service() (*OrgService, *mocks.MemoryOrgStore) {
	store := &mocks.MemoryOrgStore{Boards: map[uuid.UUID]int64{}, Groups: map[uuid.UUID]int64{}}
	users := &mocks.MemoryUserStore{Users: []models.User{anaUser, bobUser, cyUser, eveUser}}
	return New(store, users), store
}

// acme is created by ana, with bob as a member.
func acme(t *testing.T, srv *OrgService) *models.Org {
	t.Helper()
	org, err := srv.Create(ana, "  acme ")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := srv.SetMember(ana, org.Id, bobUser.Email, models.OrgRoleMember); err != nil {
		t.Fatalf("add bob: %v", err)
	}
	return org
}

func TestCreate_TheCallerIsItsAdmin(t *testing.T) {
	srv, store := service()

	org, err := srv.Create(ana, "  acme ")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if org.Name != "acme" || org.Id == uuid.Nil || org.Role != models.OrgRoleAdmin || org.CreatedAt.IsZero() {
		t.Errorf("org = %+v", org)
	}
	stored, _ := store.FindOrg(org.Id)
	if len(stored.Members) != 1 || stored.RoleOf(ana.ID) != models.OrgRoleAdmin {
		t.Errorf("stored members = %+v, want ana as admin", stored.Members)
	}
}

func TestCreate_Rejects(t *testing.T) {
	srv, _ := service()

	if _, err := srv.Create(models.Principal{}, "acme"); !errors.Is(err, ErrForbidden) {
		t.Errorf("anonymous: %v, want ErrForbidden", err)
	}
	if _, err := srv.Create(ana, "   "); !errors.Is(err, ErrInvalidName) {
		t.Errorf("no name: %v, want ErrInvalidName", err)
	}
}

func TestListMine_CarriesTheCallersRole(t *testing.T) {
	srv, _ := service()
	org := acme(t, srv)
	_, _ = srv.Create(cy, "other")

	for caller, want := range map[*models.Principal]models.OrgRole{&ana: models.OrgRoleAdmin, &bob: models.OrgRoleMember} {
		mine, err := srv.ListMine(*caller)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(mine) != 1 || mine[0].Id != org.Id || mine[0].Role != want {
			t.Errorf("%s lists %+v, want acme as %s", caller.ID, mine, want)
		}
	}

	mine, err := srv.ListMine(eve)
	if err != nil || mine == nil || len(mine) != 0 {
		t.Errorf("eve lists %v, %v, want an empty list", mine, err)
	}
}

func TestGet_MembersSeeWhoIsInIt(t *testing.T) {
	srv, _ := service()
	org := acme(t, srv)

	detail, err := srv.Get(bob, org.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Name != "acme" || detail.Role != models.OrgRoleMember || len(detail.Members) != 2 {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.Members[0].User.Email != anaUser.Email || detail.Members[0].Role != models.OrgRoleAdmin ||
		detail.Members[1].User.Name != "Bob" || detail.Members[1].Role != models.OrgRoleMember {
		t.Errorf("members = %+v", detail.Members)
	}

	for _, id := range []uuid.UUID{org.Id, uuid.New()} {
		if _, err := srv.Get(eve, id); !errors.Is(err, ErrForbidden) {
			t.Errorf("eve on %s: %v, want ErrForbidden", id, err)
		}
	}
}

func TestRename_AdminsOnly(t *testing.T) {
	srv, store := service()
	org := acme(t, srv)

	if err := srv.Rename(bob, org.Id, "mine"); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member renamed it: %v", err)
	}
	if err := srv.Rename(ana, org.Id, "  "); !errors.Is(err, ErrInvalidName) {
		t.Errorf("empty name: %v", err)
	}
	if err := srv.Rename(ana, org.Id, " acme inc "); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if stored, _ := store.FindOrg(org.Id); stored.Name != "acme inc" {
		t.Errorf("name = %q", stored.Name)
	}
}

func TestSetMember(t *testing.T) {
	srv, store := service()
	org := acme(t, srv)

	if _, err := srv.SetMember(bob, org.Id, cyUser.Email, models.OrgRoleMember); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member added someone: %v", err)
	}
	if _, err := srv.SetMember(ana, org.Id, cyUser.Email, "owner"); !errors.Is(err, ErrInvalidRole) {
		t.Errorf("role owner: %v, want ErrInvalidRole", err)
	}
	if _, err := srv.SetMember(ana, org.Id, "nobody@example.com", models.OrgRoleMember); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("unknown email: %v, want ErrUserNotFound", err)
	}

	added, err := srv.SetMember(ana, org.Id, " CY@example.com ", models.OrgRoleAdmin)
	if err != nil {
		t.Fatalf("add cy: %v", err)
	}
	if added.User.Id != cyUser.Id || added.Role != models.OrgRoleAdmin {
		t.Errorf("added = %+v", added)
	}
	if stored, _ := store.FindOrg(org.Id); stored.RoleOf(cy.ID) != models.OrgRoleAdmin {
		t.Errorf("cy = %q, want admin", stored.RoleOf(cy.ID))
	}
}

// Someone must always be able to administer the organization.
func TestLastAdmin(t *testing.T) {
	srv, _ := service()
	org := acme(t, srv)

	if _, err := srv.SetMember(ana, org.Id, anaUser.Email, models.OrgRoleMember); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("the last admin demoted herself: %v", err)
	}
	if err := srv.RemoveMember(ana, org.Id, ana.ID); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("the last admin left: %v", err)
	}

	_, _ = srv.SetMember(ana, org.Id, bobUser.Email, models.OrgRoleAdmin)
	if err := srv.RemoveMember(ana, org.Id, ana.ID); err != nil {
		t.Errorf("one of two admins could not leave: %v", err)
	}
}

func TestRemoveMember_AdminOrOneself(t *testing.T) {
	srv, store := service()
	org := acme(t, srv)
	_, _ = srv.SetMember(ana, org.Id, cyUser.Email, models.OrgRoleMember)

	if err := srv.RemoveMember(bob, org.Id, cy.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member removed someone else: %v", err)
	}
	if err := srv.RemoveMember(eve, org.Id, eve.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a stranger: %v", err)
	}
	if err := srv.RemoveMember(bob, org.Id, bob.ID); err != nil {
		t.Errorf("bob could not leave: %v", err)
	}
	if err := srv.RemoveMember(ana, org.Id, cy.ID); err != nil {
		t.Errorf("the admin could not remove cy: %v", err)
	}

	stored, _ := store.FindOrg(org.Id)
	if len(stored.Members) != 1 || stored.RoleOf(ana.ID) != models.OrgRoleAdmin {
		t.Errorf("members = %+v, want only ana", stored.Members)
	}
}

func TestDelete_RefusedWhileItOwnsSomething(t *testing.T) {
	srv, store := service()
	org := acme(t, srv)

	if err := srv.Delete(bob, org.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member deleted it: %v", err)
	}

	store.Boards[org.Id] = 1
	if err := srv.Delete(ana, org.Id); !errors.Is(err, ErrOrgNotEmpty) {
		t.Errorf("with a board: %v, want ErrOrgNotEmpty", err)
	}
	store.Boards[org.Id], store.Groups[org.Id] = 0, 2
	if err := srv.Delete(ana, org.Id); !errors.Is(err, ErrOrgNotEmpty) {
		t.Errorf("with groups: %v, want ErrOrgNotEmpty", err)
	}

	store.Groups[org.Id] = 0
	if err := srv.Delete(ana, org.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.FindOrg(org.Id); !errors.Is(err, infrastructure.ErrOrgNotFound) {
		t.Errorf("still there: %v", err)
	}
}
