package postittemplates

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
)

type recordedTemplates struct {
	*mocks.MemoryPostitTemplateStore
	s *mocks.Steps
}

func (r recordedTemplates) CountPostitTemplates(scope models.PostitTemplateScope) (int64, error) {
	r.s.Add("count")
	return r.MemoryPostitTemplateStore.CountPostitTemplates(scope)
}

func (r recordedTemplates) PutPostitTemplate(c *models.PostitTemplate) error {
	r.s.Add("put")
	return r.MemoryPostitTemplateStore.PutPostitTemplate(c)
}

// A scope's last template is counted and stored in the scope's turn.
func TestCreate_CountsAndStoresInTheScopesTurn(t *testing.T) {
	f := setup(t)
	var s mocks.Steps
	f.srv = New(recordedTemplates{f.store, &s}, f.srv.boards, access.New(&mocks.MemoryOrgStore{}), s.Plans())
	scope := f.scope(models.TemplateScopeUser)

	if _, err := f.srv.Create(ana, scope, body()); err != nil {
		t.Fatal(err)
	}

	if !s.In("hold "+models.LimitUserTemplates+" "+string(scope.Kind)+":"+scope.Owner, "count", "put") {
		t.Errorf("steps = %v", s.List())
	}
}
