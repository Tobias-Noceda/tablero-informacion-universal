package billing

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/billing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type env struct {
	r       *gin.Engine
	gateway *mocks.FakeGateway
	users   *mocks.MemoryUserStore
	ana     models.User
}

func setup(enabled bool) *env {
	ana := models.User{Id: uuid.New(), Email: "ana@example.com"}
	users := &mocks.MemoryUserStore{Users: []models.User{ana}}
	gateway := mocks.NewFakeGateway()
	var pay infrastructure.PaymentGateway
	if enabled {
		pay = gateway
	}
	service := srv.New(pay, &mocks.MemorySubscriptionStore{}, users, srv.Config{USDARS: 1570}, time.Now)
	ctrl := NewController(mocks.StaticEntitlements{Plan: models.PlanPro, Used: map[string]models.Usage{ana.Id.String(): {Orgs: 1, Secrets: 4}}}, service, pay)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	ctrl.RegisterPublicRoutes(api)
	ctrl.RegisterRoutes(api.Group("", middleware.RequireAuth(mocks.SubjectVerifier{})))
	return &env{r, gateway, users, ana}
}

func (e *env) do(method, path, body, token string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestPlans_ArePublic(t *testing.T) {
	e := setup(true)
	w := e.do(http.MethodGet, "/api/v1/billing/plans", "", "")
	var got struct {
		Checkout bool   `json:"checkout"`
		Currency string `json:"currency"`
		Plans    []struct {
			Plan     string         `json:"plan"`
			PriceUSD int            `json:"price_usd"`
			PriceARS int            `json:"price_ars"`
			Limits   map[string]any `json:"limits"`
		} `json:"plans"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if !got.Checkout || got.Currency != "ARS" || len(got.Plans) != 3 || got.Plans[1].Plan != "pro" || got.Plans[1].PriceARS != 14100 || got.Plans[2].PriceUSD != 29 || got.Plans[0].Limits["boards_per_space"] != 5.0 {
		t.Errorf("plans = %+v", got)
	}

	if off := setup(false).do(http.MethodGet, "/api/v1/billing/plans", "", ""); !strings.Contains(off.Body.String(), `"checkout":false`) {
		t.Errorf("disabled: %s", off.Body.String())
	}
}

func TestUsage_IsTheCallers(t *testing.T) {
	e := setup(true)
	if w := e.do(http.MethodGet, "/api/v1/billing/usage", "", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", w.Code)
	}
	w := e.do(http.MethodGet, "/api/v1/billing/usage", "", e.ana.Id.String())
	if w.Code != 200 || w.Body.String() != `{"orgs":1,"groups":0,"secrets":4,"user_templates":0}` {
		t.Errorf("usage = %d %s", w.Code, w.Body.String())
	}
}

func TestSubscribe_FlowThroughTheRoutes(t *testing.T) {
	e := setup(true)
	tok := e.ana.Id.String()

	if w := e.do(http.MethodGet, "/api/v1/billing/subscription", "", tok); w.Body.String() != `{"plan":"free","status":"none"}` {
		t.Errorf("none = %s", w.Body.String())
	}
	if w := e.do(http.MethodPost, "/api/v1/billing/subscribe", `{"plan":"free","payer_email":"a@b.co"}`, tok); w.Code != 400 || !strings.Contains(w.Body.String(), "invalid_plan") {
		t.Errorf("free: %d %s", w.Code, w.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/subscribe", strings.NewReader(`{"plan":"pro","payer_email":"pay@example.com"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "tablero.example"
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"init_point":"https://mp.test/checkout/`) {
		t.Fatalf("subscribe: %d %s", w.Code, w.Body.String())
	}
	if got := e.gateway.Created[0].BackURL; got != "https://tablero.example/billing/return" {
		t.Errorf("back_url = %s", got)
	}

	e.gateway.Authorize("pre-1", time.Now().Add(30*24*time.Hour))
	w = e.do(http.MethodPost, "/api/v1/billing/sync", "", tok)
	var view models.SubscriptionView
	_ = json.Unmarshal(w.Body.Bytes(), &view)
	if w.Code != 200 || view.Plan != models.PlanPro || view.Status != models.SubscriptionAuthorized || view.AmountARS != 14100 || view.Expires == nil {
		t.Fatalf("sync: %d %s", w.Code, w.Body.String())
	}

	if w := e.do(http.MethodPost, "/api/v1/billing/subscribe", `{"plan":"max","payer_email":"pay@example.com"}`, tok); w.Code != 409 || !strings.Contains(w.Body.String(), "already_subscribed") {
		t.Errorf("again: %d %s", w.Code, w.Body.String())
	}
	if w := e.do(http.MethodPut, "/api/v1/billing/subscription", `{"plan":"max"}`, tok); w.Code != 200 || !strings.Contains(w.Body.String(), `"plan":"max"`) {
		t.Errorf("change: %d %s", w.Code, w.Body.String())
	}
	if w := e.do(http.MethodDelete, "/api/v1/billing/subscription", "", tok); w.Code != 204 {
		t.Errorf("cancel: %d", w.Code)
	}
	if w := e.do(http.MethodDelete, "/api/v1/billing/subscription", "", tok); w.Code != 404 {
		t.Errorf("cancel twice: %d", w.Code)
	}
}

func TestSubscribe_DisabledAnswers503(t *testing.T) {
	e := setup(false)
	w := e.do(http.MethodPost, "/api/v1/billing/subscribe", `{"plan":"pro","payer_email":"a@b.co"}`, e.ana.Id.String())
	if w.Code != 503 || !strings.Contains(w.Body.String(), "billing_disabled") {
		t.Errorf("%d %s", w.Code, w.Body.String())
	}
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook", `{}`, ""); w.Code != 503 {
		t.Errorf("webhook: %d", w.Code)
	}
}

func TestWebhook(t *testing.T) {
	e := setup(true)
	tok := e.ana.Id.String()
	e.do(http.MethodPost, "/api/v1/billing/subscribe", `{"plan":"pro","payer_email":"pay@example.com"}`, tok)
	e.gateway.Authorize("pre-1", time.Now().Add(30*24*time.Hour))

	e.gateway.Signature = false
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=subscription_preapproval&data.id=pre-1", `{}`, ""); w.Code != 401 {
		t.Errorf("bad signature: %d", w.Code)
	}
	if u, _ := e.users.FindUser(e.ana.Id); u.Plan != "" {
		t.Error("a webhook with a bad signature changed the plan")
	}

	e.gateway.Signature = true
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=subscription_preapproval&data.id=pre-1", `{"type":"subscription_preapproval","data":{"id":"ignored"}}`, ""); w.Code != 200 {
		t.Fatalf("preapproval: %d %s", w.Code, w.Body.String())
	}
	if u, _ := e.users.FindUser(e.ana.Id); u.Plan != models.PlanPro {
		t.Errorf("plan = %q after the webhook", u.Plan)
	}

	e.gateway.Invoices["77"] = infrastructure.GatewayInvoice{Id: "77", SubscriptionId: "pre-1", Paid: true}
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?data.id=77", `{"type":"subscription_authorized_payment"}`, ""); w.Code != 200 {
		t.Errorf("invoice by body topic: %d", w.Code)
	}
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=payment&data.id=5", `{}`, ""); w.Code != 200 {
		t.Errorf("other topic: %d", w.Code)
	}
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=subscription_preapproval&data.id=not-ours", `{}`, ""); w.Code != 200 {
		t.Errorf("unknown subscription: %d", w.Code)
	}
	e.gateway.Err = io.ErrUnexpectedEOF
	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=subscription_preapproval&data.id=pre-1", `{}`, ""); w.Code != 502 {
		t.Errorf("provider down: %d, want a failure so it retries", w.Code)
	}
}

func TestWebhook_OnlyTheSignedQueryIdCounts(t *testing.T) {
	e := setup(true)
	tok := e.ana.Id.String()
	e.do(http.MethodPost, "/api/v1/billing/subscribe", `{"plan":"pro","payer_email":"pay@example.com"}`, tok)
	e.gateway.Authorize("pre-1", time.Now().Add(30*24*time.Hour))

	if w := e.do(http.MethodPost, "/api/v1/billing/webhook?type=subscription_preapproval", `{"type":"subscription_preapproval","data":{"id":"pre-1"}}`, ""); w.Code != 400 {
		t.Errorf("id only in the body: %d", w.Code)
	}
	if u, _ := e.users.FindUser(e.ana.Id); u.Plan != "" {
		t.Error("an unsigned body id changed the plan")
	}
}
