//go:build integration

package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

type refusal struct {
	Error string `json:"error"`
	Limit string `json:"limit"`
	Max   int    `json:"max"`
	Plan  string `json:"plan"`
}

func TestEndToEnd_PlansLimitsAndMercadoPagoWebhook(t *testing.T) {
	gateway := mocks.NewFakeGateway()
	s := newStackWith(t, func(cfg *config) {
		cfg.gateway = gateway
		cfg.billingUSDARS = 1570
	})

	// The plans are public.
	var plans struct {
		Checkout bool `json:"checkout"`
		Plans    []struct {
			Plan     string `json:"plan"`
			PriceARS int    `json:"price_ars"`
		} `json:"plans"`
	}
	s.mustJSON(s.do(http.MethodGet, "/api/v1/billing/plans", nil), http.StatusOK, &plans)
	if !plans.Checkout || len(plans.Plans) != 3 || plans.Plans[1].PriceARS != 14100 {
		t.Fatalf("plans = %+v", plans)
	}

	// A new account is on free: five boards, no tokens.
	ana := s.register("ana-" + strings.ReplaceAll(time.Now().Format("150405.000"), ".", "") + "@it.test")
	var profile models.Profile
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/users/"+ana.id, nil), http.StatusOK, &profile)
	if profile.Plan != models.PlanFree || profile.Limits != models.LimitsFor(models.PlanFree) {
		t.Fatalf("profile = %+v", profile)
	}

	var boards []models.Board
	for i := range 5 {
		boards = append(boards, ana.newBoard("free "+string(rune('a'+i))))
	}
	defer func() {
		for _, b := range boards {
			ana.do(http.MethodDelete, "/api/v1/boards/"+b.Id.String(), nil)
		}
	}()

	var refused refusal
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "sixth"}), http.StatusPaymentRequired, &refused)
	if refused != (refusal{"plan_limit", "boards", 5, "free"}) {
		t.Errorf("refusal = %+v", refused)
	}
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/users/"+ana.id+"/tokens", map[string]any{"name": "tool"}), http.StatusPaymentRequired, &refused)
	if refused.Limit != "api_access" {
		t.Errorf("token refusal = %+v", refused)
	}

	var board models.Board
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/boards/"+boards[0].Id.String(), nil), http.StatusOK, &board)
	if board.Limits == nil || board.Limits.BoardsPerSpace != 5 || board.SeatsUsed != 1 || board.SecretsLeft == nil || *board.SecretsLeft != 5 {
		t.Errorf("board limits = %+v seats %d left %v", board.Limits, board.SeatsUsed, board.SecretsLeft)
	}

	// Only an admin may set a plan by hand.
	if w := ana.do(http.MethodPut, "/api/v1/system/users/"+ana.id+"/plan", map[string]any{"plan": "max"}); w.Code != http.StatusForbidden {
		t.Errorf("a user set their own plan: %d", w.Code)
	}

	// The admin's override lifts the limits.
	s.setPlan(ana, models.PlanPro)
	boards = append(boards, ana.newBoard("sixth"))
	ana.tokenFor("tool")

	// Back to free by hand, then a real subscription through the webhook.
	s.setPlan(ana, models.PlanFree)
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/billing/subscribe", map[string]string{"plan": "max", "payer_email": ana.email}), http.StatusOK, nil)
	gateway.Authorize("pre-1", time.Now().Add(30*24*time.Hour))

	hook := "/api/v1/billing/webhook?type=subscription_preapproval&data.id=pre-1"
	gateway.Signature = false
	if w := s.do(http.MethodPost, hook, map[string]any{}); w.Code != http.StatusUnauthorized {
		t.Errorf("bad signature: %d", w.Code)
	}
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/users/"+ana.id, nil), http.StatusOK, &profile)
	if profile.Plan != models.PlanFree {
		t.Errorf("the plan changed on a bad signature: %s", profile.Plan)
	}

	gateway.Signature = true
	if w := s.do(http.MethodPost, hook, map[string]any{}); w.Code != http.StatusOK {
		t.Fatalf("webhook: %d %s", w.Code, w.Body.String())
	}
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/users/"+ana.id, nil), http.StatusOK, &profile)
	if profile.Plan != models.PlanMax || profile.PlanExpires == nil {
		t.Errorf("profile after the webhook = %+v", profile)
	}

	var sub models.SubscriptionView
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/billing/subscription", nil), http.StatusOK, &sub)
	if sub.Plan != models.PlanMax || sub.Status != models.SubscriptionAuthorized || sub.AmountARS != 45500 {
		t.Errorf("subscription = %+v", sub)
	}
}

// Migrating twice changes nothing and fails nowhere: the billing backfill
// only writes where a field is absent.
func TestEndToEnd_MigrateTwiceIsIdempotent(t *testing.T) {
	s := newStack(t)
	owner := s.signUp("owner")

	var org models.Org
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/orgs", map[string]string{"name": "acme"}), http.StatusCreated, &org)
	defer owner.do(http.MethodDelete, "/api/v1/orgs/"+org.Id.String(), nil)

	for range 2 {
		if err := migrate(s.db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	var orgs []models.Org
	s.mustJSON(owner.do(http.MethodGet, "/api/v1/orgs", nil), http.StatusOK, &orgs)
	if len(orgs) != 1 || orgs[0].Owner != owner.id {
		t.Errorf("orgs = %+v", orgs)
	}
}
