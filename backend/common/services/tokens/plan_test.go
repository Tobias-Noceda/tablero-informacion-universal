package tokens

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

func TestCreate_NeedsAPIAccess(t *testing.T) {
	for plan, ok := range map[models.Plan]bool{models.PlanFree: false, models.PlanPro: true, models.PlanMax: true} {
		f := setup()
		f.svc.plans = mocks.StaticEntitlements{Plan: plan}
		_, err := f.svc.Create(ana, "laptop", 0)
		var le *models.LimitError
		if ok && err != nil {
			t.Errorf("%s: %v", plan, err)
		}
		if !ok && (!errors.As(err, &le) || le.Limit != models.LimitAPIAccess) {
			t.Errorf("%s: %v", plan, err)
		}
	}
}

func TestResolve_RejectsAnOwnerWithoutAPIAccess(t *testing.T) {
	f := setup()
	created, err := f.svc.Create(ana, "laptop", 0)
	if err != nil {
		t.Fatal(err)
	}

	f.svc.plans = mocks.StaticEntitlements{Plan: models.PlanFree}
	_, err = f.svc.Resolve(created.Token)
	var le *models.LimitError
	if !errors.As(err, &le) || le.Limit != models.LimitAPIAccess || errors.Is(err, infrastructure.ErrInvalidToken) {
		t.Errorf("err = %v, want api_access", err)
	}

	f.svc.plans = mocks.StaticEntitlements{Plan: models.PlanPro}
	if _, err := f.svc.Resolve(created.Token); err != nil {
		t.Errorf("pro: %v", err)
	}
}
