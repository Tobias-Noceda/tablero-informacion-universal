package orgs

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type recordedOrgs struct {
	*mocks.MemoryOrgStore
	s *mocks.Steps
}

func (r recordedOrgs) CountOwnedOrgs(user string) (int64, error) {
	r.s.Add("count")
	return r.MemoryOrgStore.CountOwnedOrgs(user)
}

func (r recordedOrgs) CreateOrg(org *models.Org) error {
	r.s.Add("create")
	return r.MemoryOrgStore.CreateOrg(org)
}

func (r recordedOrgs) FindOrg(id uuid.UUID) (*models.Org, error) {
	r.s.Add("find")
	return r.MemoryOrgStore.FindOrg(id)
}

func (r recordedOrgs) SetOrgMember(id uuid.UUID, user string, role models.OrgRole) error {
	r.s.Add("set")
	return r.MemoryOrgStore.SetOrgMember(id, user, role)
}

// CONC-04: an organization, or a seat in one, is taken in its turn, on the
// organization as it is then.
func TestOrgs_CountAndWriteInTheirTurn(t *testing.T) {
	var s mocks.Steps
	store := recordedOrgs{&mocks.MemoryOrgStore{}, &s}
	srv := New(store, &mocks.MemoryUserStore{Users: []models.User{anaUser, bobUser, cyUser, eveUser}}, &purgeRecorder{}, &mocks.CountingLimiter{}, s.Plans())

	org, err := srv.Create(ana, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if !s.In("hold "+models.LimitOrgs+" user:"+ana.ID, "count", "create") {
		t.Errorf("create: steps = %v", s.List())
	}

	s = mocks.Steps{}
	if _, err := srv.SetMember(ana, org.Id, eveUser.Email, models.OrgRoleMember); err != nil {
		t.Fatal(err)
	}
	if !s.In("find", "hold "+models.LimitSeats+" org:"+org.Id.String(), "find", "set") {
		t.Errorf("set member: steps = %v", s.List())
	}
}
