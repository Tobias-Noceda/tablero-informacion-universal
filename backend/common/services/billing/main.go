// Package billing sells the paid plans: it starts monthly subscriptions at
// the payment provider and turns what the provider reports into the plan of
// the user who pays. The provider is the source of truth; a webhook only says
// something changed, so every sync asks it again.
package billing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// GRACE is how long a plan outlasts the date of the next charge, so a late
// payment or a missed webhook does not cut a paying user off.
const GRACE = 5 * 24 * time.Hour

// CURRENCY is what subscriptions are charged in.
const CURRENCY = "ARS"

var (
	ErrDisabled            = errors.New("billing_disabled")
	ErrInvalidPlan         = errors.New("invalid_plan")
	ErrInvalidEmail        = errors.New("invalid_email")
	ErrAlreadySubscribed   = errors.New("already_subscribed")
	ErrNoSubscription      = errors.New("no_subscription")
	ErrUnknownSubscription = errors.New("unknown_subscription")
	ErrForeignSubscription = errors.New("foreign_subscription")
)

// Users is the slice of the user store billing needs.
type Users interface {
	FindUser(id uuid.UUID) (*models.User, error)
	SetUserPlan(id uuid.UUID, plan models.Plan, expires *time.Time) error
}

// Config is what the service takes from the environment: the exchange rate
// subscriptions are priced at (ARS per USD, zero when none is set).
type Config struct {
	USDARS float64
}

type Service struct {
	gateway infrastructure.PaymentGateway
	store   infrastructure.SubscriptionStore
	users   Users
	cfg     Config
	now     func() time.Time
}

// New is the billing service. A nil gateway disables checkout: plans are
// still listed, nothing can be bought.
func New(gateway infrastructure.PaymentGateway, store infrastructure.SubscriptionStore, users Users, cfg Config, now func() time.Time) *Service {
	return &Service{gateway, store, users, cfg, now}
}

// Enabled is whether a plan can be bought.
func (srv *Service) Enabled() bool {
	return srv.gateway != nil
}

// PlanOffer is one plan as the pricing page shows it.
type PlanOffer struct {
	Plan     models.Plan   `json:"plan"`
	PriceUSD int           `json:"price_usd"`
	PriceARS int           `json:"price_ars"`
	Limits   models.Limits `json:"limits"`
}

// PlansResponse lists the plans, cheapest first. Checkout tells whether one
// can be bought here; without it the prices are only for show.
type PlansResponse struct {
	Checkout bool        `json:"checkout"`
	Currency string      `json:"currency"`
	Plans    []PlanOffer `json:"plans"`
}

// PriceARS is what a plan costs a month in pesos: its price in dollars at the
// configured rate, to the nearest hundred. Zero without a rate.
func (srv *Service) PriceARS(plan models.Plan) int {
	usd := models.LimitsFor(plan).PriceUSD
	if usd == 0 || srv.cfg.USDARS <= 0 {
		return 0
	}
	return int(math.Round(float64(usd)*srv.cfg.USDARS/100)) * 100
}

func (srv *Service) Plans() PlansResponse {
	offers := make([]PlanOffer, 0, len(models.Plans))
	for _, plan := range models.Plans {
		limits := models.LimitsFor(plan)
		offers = append(offers, PlanOffer{Plan: plan, PriceUSD: limits.PriceUSD, PriceARS: srv.PriceARS(plan), Limits: limits})
	}
	return PlansResponse{Checkout: srv.Enabled(), Currency: CURRENCY, Plans: offers}
}

func paid(plan models.Plan) bool {
	return plan.Valid() && plan != models.PlanFree
}

func reference(user string, plan models.Plan) string {
	return user + ":" + string(plan)
}

// referenceUser is the user a subscription was created for.
func referenceUser(external string) string {
	user, _, _ := strings.Cut(external, ":")
	return user
}

// Subscribe starts a subscription to plan and answers where the payer
// approves it. The plan only changes once the provider reports it authorized.
func (srv *Service) Subscribe(principal models.Principal, plan models.Plan, payerEmail, backURL string) (string, error) {
	if !srv.Enabled() {
		return "", ErrDisabled
	}
	if principal.Anonymous() || principal.Token || !paid(plan) {
		return "", ErrInvalidPlan
	}
	payerEmail = models.NormalizeEmail(payerEmail)
	if !models.ValidEmail(payerEmail) {
		return "", ErrInvalidEmail
	}

	open, err := srv.store.OpenSubscriptionOf(principal.ID)
	if err != nil && !errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
		return "", err
	}
	// A paused one still holds the plan: it is resumed at the provider, not
	// replaced by a second charge.
	if open != nil && (open.Status == models.SubscriptionAuthorized || open.Status == models.SubscriptionPaused) {
		return "", ErrAlreadySubscribed
	}

	// The same plan asked for again within the minute (a double click, a
	// retry) is the same request to the provider.
	now := srv.now().UTC()
	sum := sha256.Sum256([]byte(reference(principal.ID, plan) + ":" + now.Format("200601021504")))
	key := hex.EncodeToString(sum[:16])
	amount := srv.PriceARS(plan)
	created, err := srv.gateway.CreateSubscription(infrastructure.NewSubscription{
		Key:               key,
		Reason:            "Tablero " + strings.ToUpper(string(plan[:1])) + string(plan[1:]),
		ExternalReference: reference(principal.ID, plan),
		PayerEmail:        payerEmail,
		AmountARS:         amount,
		BackURL:           backURL,
	})
	if err != nil {
		return "", err
	}

	err = srv.store.UpsertSubscription(&models.Subscription{
		Id: created.Id, User: principal.ID, Plan: plan, Status: models.SubscriptionPending,
		AmountARS: amount, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return "", err
	}
	return created.InitPoint, nil
}

// Sync brings the stored subscription and its user's plan in line with what
// the provider says. Nothing the webhook carried is used.
func (srv *Service) Sync(preapprovalID string) (*models.Subscription, error) {
	if !srv.Enabled() {
		return nil, ErrDisabled
	}

	stored, err := srv.store.FindSubscription(preapprovalID)
	if errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
		return nil, ErrUnknownSubscription
	}
	if err != nil {
		return nil, err
	}

	remote, err := srv.gateway.GetSubscription(preapprovalID)
	if err != nil {
		return nil, err
	}
	if referenceUser(remote.ExternalReference) != stored.User {
		return nil, ErrForeignSubscription
	}

	stored.Status = models.ParseSubscriptionStatus(remote.Status)
	if remote.AmountARS > 0 {
		stored.AmountARS = remote.AmountARS
	}
	stored.NextPayment = remote.NextPayment
	stored.UpdatedAt = srv.now().UTC()
	if err := srv.store.UpsertSubscription(stored); err != nil {
		return nil, err
	}

	if stored.Status == models.SubscriptionAuthorized {
		// A superseded subscription that is still authorized at the provider
		// must not overwrite the plan of the user's newer one.
		latest, err := srv.store.LatestSubscription(stored.User)
		if err != nil && !errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
			return nil, err
		}
		if latest == nil || latest.Id == stored.Id {
			if err := srv.grant(stored); err != nil {
				return nil, err
			}
		}
	}
	// Paused or canceled: the plan stays until the date it already runs to,
	// after which it reads as free. Nothing to write.
	return stored, nil
}

// grant puts the subscription's user on its plan until a grace period after
// the next charge.
func (srv *Service) grant(sub *models.Subscription) error {
	id, err := uuid.Parse(sub.User)
	if err != nil {
		return ErrForeignSubscription
	}

	// Never already over: a next charge that is behind us (an upgrade
	// charged right now, a late report) still earns the grace.
	now := srv.now().UTC()
	until := now.Add(30*24*time.Hour + GRACE)
	if sub.NextPayment != nil {
		until = sub.NextPayment.UTC()
		if until.Before(now) {
			until = now
		}
		until = until.Add(GRACE)
	}
	return srv.users.SetUserPlan(id, sub.Plan, &until)
}

// SyncInvoice reacts to a charge: once it was paid, a downgrade waiting for
// it takes effect, then the subscription is synced.
func (srv *Service) SyncInvoice(invoiceID string) (*models.Subscription, error) {
	if !srv.Enabled() {
		return nil, ErrDisabled
	}

	invoice, err := srv.gateway.GetInvoice(invoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.SubscriptionId == "" {
		return nil, ErrUnknownSubscription
	}

	if invoice.Paid {
		stored, err := srv.store.FindSubscription(invoice.SubscriptionId)
		if errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
			return nil, ErrUnknownSubscription
		}
		if err != nil {
			return nil, err
		}
		if stored.PendingPlan != "" {
			stored.Plan, stored.PendingPlan = stored.PendingPlan, ""
			stored.UpdatedAt = srv.now().UTC()
			if err := srv.store.UpsertSubscription(stored); err != nil {
				return nil, err
			}
		}
	}

	return srv.Sync(invoice.SubscriptionId)
}

// ChangePlan moves the caller's authorized subscription to another paid plan.
// An upgrade applies now; a downgrade is charged from the next payment on and
// applies once that one clears. No proration.
func (srv *Service) ChangePlan(principal models.Principal, plan models.Plan) (*models.SubscriptionView, error) {
	if !srv.Enabled() {
		return nil, ErrDisabled
	}
	if principal.Anonymous() || principal.Token || !paid(plan) {
		return nil, ErrInvalidPlan
	}

	sub, err := srv.store.OpenSubscriptionOf(principal.ID)
	if errors.Is(err, infrastructure.ErrSubscriptionNotFound) || (err == nil && sub.Status != models.SubscriptionAuthorized) {
		return nil, ErrNoSubscription
	}
	if err != nil {
		return nil, err
	}

	switch {
	case plan == sub.Plan && sub.PendingPlan == "":
		return srv.view(principal.ID, sub)
	case plan == sub.Plan:
		// Back to what it already is: drop the downgrade.
		if err := srv.gateway.UpdateAmount(sub.Id, srv.PriceARS(plan)); err != nil {
			return nil, err
		}
		sub.PendingPlan = ""
	case plan.Rank() > sub.Plan.Rank():
		if err := srv.gateway.UpdateAmount(sub.Id, srv.PriceARS(plan)); err != nil {
			return nil, err
		}
		sub.Plan, sub.PendingPlan = plan, ""
	default:
		if err := srv.gateway.UpdateAmount(sub.Id, srv.PriceARS(plan)); err != nil {
			return nil, err
		}
		sub.PendingPlan = plan
	}

	sub.AmountARS = srv.PriceARS(plan)
	sub.UpdatedAt = srv.now().UTC()
	if err := srv.store.UpsertSubscription(sub); err != nil {
		return nil, err
	}
	if sub.PendingPlan == "" {
		if err := srv.grant(sub); err != nil {
			return nil, err
		}
	}
	return srv.view(principal.ID, sub)
}

// Cancel ends the caller's subscription; the plan lasts until the date it
// already runs to.
func (srv *Service) Cancel(principal models.Principal) error {
	if !srv.Enabled() {
		return ErrDisabled
	}

	sub, err := srv.store.LatestSubscription(principal.ID)
	if errors.Is(err, infrastructure.ErrSubscriptionNotFound) || (err == nil && !sub.Status.Open()) {
		return ErrNoSubscription
	}
	if err != nil {
		return err
	}

	if err := srv.gateway.Cancel(sub.Id); err != nil {
		return err
	}
	sub.Status, sub.PendingPlan = models.SubscriptionCanceled, ""
	sub.UpdatedAt = srv.now().UTC()
	return srv.store.UpsertSubscription(sub)
}

// SyncMine syncs the caller's latest subscription, the return page's way of
// not waiting for a webhook.
func (srv *Service) SyncMine(principal models.Principal) (*models.SubscriptionView, error) {
	sub, err := srv.store.LatestSubscription(principal.ID)
	if errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
		return srv.view(principal.ID, nil)
	}
	if err != nil {
		return nil, err
	}

	if srv.Enabled() {
		if sub, err = srv.Sync(sub.Id); err != nil {
			return nil, err
		}
	}
	return srv.view(principal.ID, sub)
}

// Subscription is what the caller is told about their plan and charge.
func (srv *Service) Subscription(principal models.Principal) (*models.SubscriptionView, error) {
	sub, err := srv.store.LatestSubscription(principal.ID)
	if errors.Is(err, infrastructure.ErrSubscriptionNotFound) {
		return srv.view(principal.ID, nil)
	}
	if err != nil {
		return nil, err
	}
	return srv.view(principal.ID, sub)
}

func (srv *Service) view(userID string, sub *models.Subscription) (*models.SubscriptionView, error) {
	view := &models.SubscriptionView{Plan: models.PlanFree, Status: models.SubscriptionNone}

	id, err := uuid.Parse(userID)
	if err != nil {
		return view, nil
	}
	user, err := srv.users.FindUser(id)
	if errors.Is(err, infrastructure.ErrUserNotFound) {
		return view, nil
	}
	if err != nil {
		return nil, err
	}

	view.Plan = user.EffectivePlan(srv.now())
	if view.Plan != models.PlanFree && !user.Admin {
		view.Expires = user.PlanExpires
	}
	if sub != nil {
		view.Status, view.PendingPlan = sub.Status, sub.PendingPlan
		view.AmountARS, view.NextPayment = sub.AmountARS, sub.NextPayment
	}
	return view, nil
}

// Reconcile syncs every subscription that is not canceled, so a webhook that
// never arrived is made up for. It reports how many it synced and how many
// failed; a failure does not stop the rest.
func (srv *Service) Reconcile() (synced, failed int, err error) {
	if !srv.Enabled() {
		return 0, 0, ErrDisabled
	}

	subs, err := srv.store.OpenSubscriptions()
	if err != nil {
		return 0, 0, err
	}
	for _, sub := range subs {
		if _, err := srv.Sync(sub.Id); err != nil {
			failed++
			log.Printf("billing: reconcile %s: %v", sub.Id, err)
			continue
		}
		synced++
	}
	return synced, failed, nil
}
