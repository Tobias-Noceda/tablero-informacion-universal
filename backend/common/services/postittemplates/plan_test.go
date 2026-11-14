package postittemplates

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

func TestCreate_LimitPerKindByPlan(t *testing.T) {
	kinds := map[models.PostitTemplateScopeKind]func(models.Limits) (string, int){
		models.TemplateScopeUser:  func(l models.Limits) (string, int) { return models.LimitUserTemplates, l.UserTemplates },
		models.TemplateScopeBoard: func(l models.Limits) (string, int) { return models.LimitBoardTemplates, l.BoardTemplates },
	}
	for _, plan := range models.Plans {
		for kind, of := range kinds {
			code, max := of(models.LimitsFor(plan))
			f := setup(t)
			f.srv.plans = mocks.StaticEntitlements{Plan: plan}
			scope := f.scope(kind)
			who := ana
			for i := range max {
				if _, err := f.srv.Create(who, scope, body()); err != nil {
					t.Fatalf("%s %s #%d of %d: %v", plan, kind, i+1, max, err)
				}
			}
			_, err := f.srv.Create(who, scope, body())
			var le *models.LimitError
			if !errors.As(err, &le) || le.Limit != code || le.Max != max {
				t.Errorf("%s %s over %d: %v", plan, kind, max, err)
			}
		}
	}
}
