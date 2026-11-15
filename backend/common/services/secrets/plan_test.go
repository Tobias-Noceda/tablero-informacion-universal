package secrets

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

func TestPut_SecretsByPlanAndBilledTo(t *testing.T) {
	for _, plan := range models.Plans {
		max := models.LimitsFor(plan).Secrets
		for _, used := range []int{max - 1, max} {
			store := newStore()
			srv := serviceOn(t, store, mocks.StaticEntitlements{Plan: plan, Secrets: map[string]int{owner.ID: used}})
			err := srv.Put(models.UserScope(owner.ID), owner, "KEY", models.SecretApiKey, "v")
			var le *models.LimitError
			if used >= max {
				if !errors.As(err, &le) || le.Limit != models.LimitSecrets || le.Max != max {
					t.Errorf("%s with %d: %v", plan, used, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s with %d: %v", plan, used, err)
				continue
			}
			if store.rows[0].BilledTo != owner.ID {
				t.Errorf("billed to %q, want the payer", store.rows[0].BilledTo)
			}
		}
	}
}

func TestPut_RewritingASecretIsNeverANewOne(t *testing.T) {
	store := newStore()
	board := models.UserScope(owner.ID)
	srv := serviceOn(t, store, mocks.StaticEntitlements{Plan: models.PlanFree})
	if err := srv.Put(board, owner, "KEY", models.SecretApiKey, "v1"); err != nil {
		t.Fatal(err)
	}
	// The fake counts nothing, so make the plan full and rewrite.
	full := serviceOn(t, store, mocks.StaticEntitlements{Plan: models.PlanFree, Secrets: map[string]int{owner.ID: 5}})
	if err := full.Put(board, owner, "KEY", models.SecretApiKey, "v2"); err != nil {
		t.Errorf("rewrite on a full plan: %v", err)
	}
	if err := full.Put(board, owner, "OTHER", models.SecretApiKey, "v"); err == nil {
		t.Error("a new secret on a full plan went through")
	}
}

func TestPut_SystemScopeIsNeverLimited(t *testing.T) {
	store := newStore()
	srv := serviceOn(t, store, mocks.StaticEntitlements{Plan: models.PlanFree, Secrets: map[string]int{"": 999}})
	if err := srv.Put(models.SystemScope, admin, "NASA_API_KEY", models.SecretApiKey, "v"); err != nil {
		t.Errorf("system: %v", err)
	}
}
