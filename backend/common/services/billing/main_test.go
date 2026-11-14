package billing

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var now = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

type fixture struct {
	srv     *Service
	gateway *mocks.FakeGateway
	store   *mocks.MemorySubscriptionStore
	users   *mocks.MemoryUserStore
	user    models.User
	ana     models.Principal
	clock   *time.Time
}

func setup() *fixture {
	user := models.User{Id: uuid.New(), Email: "ana@example.com"}
	f := &fixture{gateway: mocks.NewFakeGateway(), store: &mocks.MemorySubscriptionStore{}, users: &mocks.MemoryUserStore{Users: []models.User{user}}, user: user, ana: user.Principal()}
	t := now
	f.clock = &t
	f.srv = New(f.gateway, f.store, f.users, Config{USDARS: 1570}, func() time.Time { return *f.clock })
	return f
}

func (f *fixture) plan() models.Plan {
	u, _ := f.users.FindUser(f.user.Id)
	return u.EffectivePlan(*f.clock)
}

func (f *fixture) expires() *time.Time {
	u, _ := f.users.FindUser(f.user.Id)
	return u.PlanExpires
}

// subscribed walks a user through subscribe and the provider's authorization.
func (f *fixture) subscribed(t *testing.T, plan models.Plan, next time.Time) string {
	t.Helper()
	if _, err := f.srv.Subscribe(f.ana, plan, "pay@example.com", "https://x/billing/return"); err != nil {
		t.Fatal(err)
	}
	subID := fmt.Sprintf("pre-%d", len(f.gateway.Created))
	f.gateway.Authorize(subID, next)
	if _, err := f.srv.Sync(subID); err != nil {
		t.Fatal(err)
	}
	return subID
}

func TestPlans_PricesFollowTheRate(t *testing.T) {
	f := setup()
	plans := f.srv.Plans()
	if !plans.Checkout || plans.Currency != "ARS" || len(plans.Plans) != 3 {
		t.Fatalf("plans = %+v", plans)
	}
	got := map[models.Plan]int{}
	for _, p := range plans.Plans {
		got[p.Plan] = p.PriceARS
		if p.Limits != models.LimitsFor(p.Plan) || p.PriceUSD != p.Limits.PriceUSD {
			t.Errorf("offer = %+v", p)
		}
	}
	// 9 x 1570 = 14130 and 29 x 1570 = 45530, to the nearest 100.
	if got[models.PlanFree] != 0 || got[models.PlanPro] != 14100 || got[models.PlanMax] != 45500 {
		t.Errorf("ars = %v", got)
	}
}

func TestPlans_WithoutAGatewayNothingCanBeBought(t *testing.T) {
	srv := New(nil, &mocks.MemorySubscriptionStore{}, &mocks.MemoryUserStore{}, Config{}, time.Now)
	plans := srv.Plans()
	if plans.Checkout || len(plans.Plans) != 3 || plans.Plans[1].PriceARS != 0 {
		t.Errorf("plans = %+v", plans)
	}
	if _, err := srv.Subscribe(models.Principal{ID: uuid.NewString()}, models.PlanPro, "a@b.co", ""); !errors.Is(err, ErrDisabled) {
		t.Errorf("subscribe: %v", err)
	}
}

func TestSubscribe_CreatesPendingAndGivesNothingYet(t *testing.T) {
	f := setup()
	point, err := f.srv.Subscribe(f.ana, models.PlanPro, " Pay@Example.com ", "https://x/billing/return")
	if err != nil || point == "" {
		t.Fatalf("subscribe: %q %v", point, err)
	}
	req := f.gateway.Created[0]
	if req.ExternalReference != f.user.Id.String()+":pro" || req.PayerEmail != "pay@example.com" || req.AmountARS != 14100 || req.BackURL != "https://x/billing/return" {
		t.Errorf("request = %+v", req)
	}
	if sub, _ := f.store.LatestSubscription(f.ana.ID); sub == nil || sub.Status != models.SubscriptionPending || sub.Plan != models.PlanPro {
		t.Errorf("stored = %+v", sub)
	}
	if f.plan() != models.PlanFree {
		t.Error("a plan was granted before the payer approved")
	}
}

func TestSubscribe_Refusals(t *testing.T) {
	f := setup()
	for name, c := range map[string]struct {
		plan  models.Plan
		email string
		who   models.Principal
		want  error
	}{
		"free":      {models.PlanFree, "a@b.co", f.ana, ErrInvalidPlan},
		"unknown":   {"gold", "a@b.co", f.ana, ErrInvalidPlan},
		"bad email": {models.PlanPro, "nope", f.ana, ErrInvalidEmail},
		"anonymous": {models.PlanPro, "a@b.co", models.Principal{}, ErrInvalidPlan},
		"token":     {models.PlanPro, "a@b.co", models.Principal{ID: f.ana.ID, Token: true}, ErrInvalidPlan},
	} {
		if _, err := f.srv.Subscribe(c.who, c.plan, c.email, ""); !errors.Is(err, c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(f.gateway.Created) != 0 {
		t.Error("a refused request reached the provider")
	}
}

func TestSync_AuthorizedGrantsThePlanUntilNextPaymentPlusGrace(t *testing.T) {
	f := setup()
	next := now.Add(30 * 24 * time.Hour)
	f.subscribed(t, models.PlanPro, next)

	if f.plan() != models.PlanPro {
		t.Fatalf("plan = %s", f.plan())
	}
	if exp := f.expires(); exp == nil || !exp.Equal(next.Add(5*24*time.Hour)) {
		t.Errorf("expires = %v, want next payment + 5 days", exp)
	}
	if _, err := f.srv.Subscribe(f.ana, models.PlanMax, "pay@example.com", ""); !errors.Is(err, ErrAlreadySubscribed) {
		t.Errorf("second subscribe: %v", err)
	}
}

func TestSync_RejectsAForeignExternalReference(t *testing.T) {
	f := setup()
	if _, err := f.srv.Subscribe(f.ana, models.PlanPro, "pay@example.com", ""); err != nil {
		t.Fatal(err)
	}
	f.gateway.Subs["pre-1"].ExternalReference = uuid.NewString() + ":max"
	f.gateway.Authorize("pre-1", now.Add(time.Hour))

	if _, err := f.srv.Sync("pre-1"); !errors.Is(err, ErrForeignSubscription) {
		t.Errorf("err = %v", err)
	}
	if f.plan() != models.PlanFree {
		t.Error("a foreign subscription granted a plan")
	}
}

func TestSync_UnknownSubscriptionIsIgnored(t *testing.T) {
	f := setup()
	if _, err := f.srv.Sync("someone-elses"); !errors.Is(err, ErrUnknownSubscription) {
		t.Errorf("err = %v", err)
	}
}

func TestSync_PendingAndPausedGrantNothing(t *testing.T) {
	f := setup()
	if _, err := f.srv.Subscribe(f.ana, models.PlanPro, "pay@example.com", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.Sync("pre-1"); err != nil || f.plan() != models.PlanFree {
		t.Errorf("pending: %v %s", err, f.plan())
	}
}

func TestCancel_KeepsAccessUntilItLapses(t *testing.T) {
	f := setup()
	next := now.Add(10 * 24 * time.Hour)
	id := f.subscribed(t, models.PlanPro, next)

	if err := f.srv.Cancel(f.ana); err != nil {
		t.Fatal(err)
	}
	if len(f.gateway.Canceled) != 1 {
		t.Error("the provider was not told")
	}
	if f.plan() != models.PlanPro {
		t.Error("cancelling cut access at once")
	}

	// The provider spells it both ways; syncing keeps the plan.
	if _, err := f.srv.Sync(id); err != nil {
		t.Fatal(err)
	}
	if f.plan() != models.PlanPro {
		t.Error("syncing a canceled subscription cut access")
	}

	*f.clock = next.Add(6 * 24 * time.Hour)
	if f.plan() != models.PlanFree {
		t.Errorf("plan after the end = %s", f.plan())
	}

	view, _ := f.srv.Subscription(f.ana)
	if view.Status != models.SubscriptionCanceled {
		t.Errorf("view = %+v", view)
	}
	if err := f.srv.Cancel(f.ana); !errors.Is(err, ErrNoSubscription) {
		t.Errorf("second cancel: %v", err)
	}
}

func TestChangePlan_UpgradeAppliesNow(t *testing.T) {
	f := setup()
	f.subscribed(t, models.PlanPro, now.Add(20*24*time.Hour))

	view, err := f.srv.ChangePlan(f.ana, models.PlanMax)
	if err != nil {
		t.Fatal(err)
	}
	if f.gateway.Updates[0] != 45500 || f.plan() != models.PlanMax || view.Plan != models.PlanMax || view.AmountARS != 45500 || view.PendingPlan != "" {
		t.Errorf("view = %+v, updates %v, plan %s", view, f.gateway.Updates, f.plan())
	}
}

func TestChangePlan_DowngradeWaitsForTheNextInvoice(t *testing.T) {
	f := setup()
	id := f.subscribed(t, models.PlanMax, now.Add(20*24*time.Hour))

	view, err := f.srv.ChangePlan(f.ana, models.PlanPro)
	if err != nil {
		t.Fatal(err)
	}
	if view.PendingPlan != models.PlanPro || view.Plan != models.PlanMax || f.plan() != models.PlanMax || f.gateway.Updates[0] != 14100 {
		t.Fatalf("view = %+v plan %s", view, f.plan())
	}

	// An unpaid invoice changes nothing.
	f.gateway.Invoices["inv-1"] = infrastructure.GatewayInvoice{Id: "inv-1", SubscriptionId: id, Paid: false}
	if _, err := f.srv.SyncInvoice("inv-1"); err != nil || f.plan() != models.PlanMax {
		t.Fatalf("unpaid: %v %s", err, f.plan())
	}

	f.gateway.Invoices["inv-2"] = infrastructure.GatewayInvoice{Id: "inv-2", SubscriptionId: id, Paid: true}
	f.gateway.Authorize(id, now.Add(50*24*time.Hour))
	if _, err := f.srv.SyncInvoice("inv-2"); err != nil {
		t.Fatal(err)
	}
	view, _ = f.srv.Subscription(f.ana)
	if f.plan() != models.PlanPro || view.PendingPlan != "" || view.Plan != models.PlanPro {
		t.Errorf("after the invoice: plan %s view %+v", f.plan(), view)
	}
}

func TestChangePlan_Refusals(t *testing.T) {
	f := setup()
	if _, err := f.srv.ChangePlan(f.ana, models.PlanMax); !errors.Is(err, ErrNoSubscription) {
		t.Errorf("without a subscription: %v", err)
	}
	f.subscribed(t, models.PlanPro, now.Add(time.Hour))
	if _, err := f.srv.ChangePlan(f.ana, models.PlanFree); !errors.Is(err, ErrInvalidPlan) {
		t.Errorf("to free: %v", err)
	}
	if _, err := f.srv.ChangePlan(f.ana, models.PlanPro); err != nil || len(f.gateway.Updates) != 0 {
		t.Errorf("to the same plan: %v %v", err, f.gateway.Updates)
	}
}

func TestSyncMine_AndSubscriptionView(t *testing.T) {
	f := setup()
	if view, err := f.srv.SyncMine(f.ana); err != nil || view.Plan != models.PlanFree || view.Status != models.SubscriptionNone {
		t.Errorf("none: %+v %v", view, err)
	}

	if _, err := f.srv.Subscribe(f.ana, models.PlanMax, "pay@example.com", ""); err != nil {
		t.Fatal(err)
	}
	next := now.Add(30 * 24 * time.Hour)
	f.gateway.Authorize("pre-1", next)
	view, err := f.srv.SyncMine(f.ana)
	if err != nil || view.Plan != models.PlanMax || view.Status != models.SubscriptionAuthorized || view.AmountARS != 45500 || view.NextPayment == nil || view.Expires == nil {
		t.Errorf("view = %+v %v", view, err)
	}
}

func TestReconcile_SyncsOpenSubscriptionsOnly(t *testing.T) {
	f := setup()
	id := f.subscribed(t, models.PlanPro, now.Add(time.Hour))
	f.gateway.Subs[id].Status = "cancelled"
	if _, err := f.srv.Sync(id); err != nil {
		t.Fatal(err)
	}
	// One that never reached the provider's side: its Sync fails, the run goes on.
	_ = f.store.UpsertSubscription(&models.Subscription{Id: "ghost", User: f.ana.ID, Status: models.SubscriptionPending})

	synced, failed, err := f.srv.Reconcile()
	if err != nil || synced != 0 || failed != 1 {
		t.Errorf("reconcile = %d %d %v", synced, failed, err)
	}
}

func TestSubscribe_PausedCountsAsSubscribed(t *testing.T) {
	f := setup()
	id := f.subscribed(t, models.PlanPro, now.Add(10*24*time.Hour))
	f.gateway.Subs[id].Status = "paused"
	if _, err := f.srv.Sync(id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.Subscribe(f.ana, models.PlanMax, "pay@example.com", ""); !errors.Is(err, ErrAlreadySubscribed) {
		t.Errorf("paused: %v", err)
	}
}

func TestSubscribe_SameRequestWithinAMinuteSharesItsKey(t *testing.T) {
	f := setup()
	for range 2 {
		if _, err := f.srv.Subscribe(f.ana, models.PlanPro, "pay@example.com", ""); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.gateway.Created) != 2 || f.gateway.Created[0].Key == "" || f.gateway.Created[0].Key != f.gateway.Created[1].Key {
		t.Errorf("keys = %+v", f.gateway.Created)
	}

	*f.clock = f.clock.Add(2 * time.Minute)
	if _, err := f.srv.Subscribe(f.ana, models.PlanPro, "pay@example.com", ""); err != nil {
		t.Fatal(err)
	}
	if f.gateway.Created[2].Key == f.gateway.Created[0].Key {
		t.Error("a later request reused the key")
	}
	if _, err := f.srv.Subscribe(f.ana, models.PlanMax, "pay@example.com", ""); err != nil {
		t.Fatal(err)
	}
	if f.gateway.Created[3].Key == f.gateway.Created[2].Key {
		t.Error("another plan reused the key")
	}
}

func TestSync_AStaleSubscriptionDoesNotOverwriteTheNewerPlan(t *testing.T) {
	f := setup()
	next := now.Add(10 * 24 * time.Hour)
	old := f.subscribed(t, models.PlanPro, next)
	if err := f.srv.Cancel(f.ana); err != nil {
		t.Fatal(err)
	}

	*f.clock = f.clock.Add(time.Hour)
	f.subscribed(t, models.PlanMax, next.Add(20*24*time.Hour))
	if f.plan() != models.PlanMax {
		t.Fatalf("plan = %s", f.plan())
	}
	want := f.expires()

	// The provider still reports the superseded one as authorized.
	f.gateway.Authorize(old, next.Add(90*24*time.Hour))
	if _, err := f.srv.Sync(old); err != nil {
		t.Fatal(err)
	}
	if f.plan() != models.PlanMax || !f.expires().Equal(*want) {
		t.Errorf("plan = %s expires = %v, want max until %v", f.plan(), f.expires(), want)
	}
}

func TestSync_GrantIsNeverAlreadyExpired(t *testing.T) {
	f := setup()
	f.subscribed(t, models.PlanPro, now.Add(-10*24*time.Hour))
	if exp := f.expires(); exp == nil || !exp.Equal(now.Add(GRACE)) {
		t.Errorf("expires = %v, want now + grace", exp)
	}
	if f.plan() != models.PlanPro {
		t.Errorf("plan = %s", f.plan())
	}
}
