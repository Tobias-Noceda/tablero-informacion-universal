package groups

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

type recordedGroups struct {
	*mocks.MemoryGroupStore
	s *mocks.Steps
}

func (r recordedGroups) CountOwnedGroups(user string) (int64, error) {
	r.s.Add("count")
	return r.MemoryGroupStore.CountOwnedGroups(user)
}

func (r recordedGroups) CreateGroup(group *models.Group) error {
	r.s.Add("create")
	return r.MemoryGroupStore.CreateGroup(group)
}

func (r recordedGroups) FindGroup(id uuid.UUID) (*models.Group, error) {
	r.s.Add("find")
	return r.MemoryGroupStore.FindGroup(id)
}

func (r recordedGroups) AddGroupMember(id uuid.UUID, user string) error {
	r.s.Add("add")
	return r.MemoryGroupStore.AddGroupMember(id, user)
}

// CONC-04: the count and the write that may take the plan's last group or
// seat happen in the owner's (or the group's) turn, on the group as it is then.
func TestGroups_CountAndWriteInTheirTurn(t *testing.T) {
	var s mocks.Steps
	store := recordedGroups{&mocks.MemoryGroupStore{}, &s}
	srv := New(store, &purgeRecorder{}, access.New(nil), s.Plans())

	group, err := srv.Create(owner, "ops", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !s.In("hold "+models.LimitGroups+" user:"+owner.ID, "count", "create") {
		t.Errorf("create: steps = %v", s.List())
	}

	s = mocks.Steps{}
	if err := srv.AddMember(owner, group.Id, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	if !s.In("find", "hold "+models.LimitSeats+" group:"+group.Id.String(), "find", "add") {
		t.Errorf("add member: steps = %v", s.List())
	}
}
