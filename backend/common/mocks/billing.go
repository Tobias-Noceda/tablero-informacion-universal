package mocks

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// MemorySubscriptionStore keeps subscriptions in a slice, oldest first.
type MemorySubscriptionStore struct {
	mu   sync.Mutex
	Subs []models.Subscription
}

var _ infrastructure.SubscriptionStore = (*MemorySubscriptionStore)(nil)

func (s *MemorySubscriptionStore) UpsertSubscription(sub *models.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Subs {
		if s.Subs[i].Id == sub.Id {
			s.Subs[i] = *sub
			return nil
		}
	}
	s.Subs = append(s.Subs, *sub)
	return nil
}

func (s *MemorySubscriptionStore) find(match func(models.Subscription) bool) (*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Newest first, like the database sorts them.
	for _, sub := range slices.Backward(s.Subs) {
		if match(sub) {
			found := sub
			return &found, nil
		}
	}
	return nil, infrastructure.ErrSubscriptionNotFound
}

func (s *MemorySubscriptionStore) FindSubscription(id string) (*models.Subscription, error) {
	return s.find(func(sub models.Subscription) bool { return sub.Id == id })
}

func (s *MemorySubscriptionStore) LatestSubscription(user string) (*models.Subscription, error) {
	return s.find(func(sub models.Subscription) bool { return sub.User == user })
}

func (s *MemorySubscriptionStore) OpenSubscriptionOf(user string) (*models.Subscription, error) {
	return s.find(func(sub models.Subscription) bool {
		return sub.User == user && (sub.Status == models.SubscriptionAuthorized || sub.Status == models.SubscriptionPaused)
	})
}

func (s *MemorySubscriptionStore) OpenSubscriptions() ([]models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var open []models.Subscription
	for _, sub := range s.Subs {
		if sub.Status.Open() {
			open = append(open, sub)
		}
	}
	return open, nil
}

// FakeGateway is a payment provider that lives in a map. Tests move a
// subscription along by editing Subs, the way a payer or the provider would.
type FakeGateway struct {
	mu       sync.Mutex
	Subs     map[string]*infrastructure.GatewaySubscription
	Invoices map[string]infrastructure.GatewayInvoice
	Created  []infrastructure.NewSubscription
	Updates  []int
	Canceled []string
	// Signature is what ValidSignature answers.
	Signature bool
	Err       error
}

var _ infrastructure.PaymentGateway = (*FakeGateway)(nil)

func NewFakeGateway() *FakeGateway {
	return &FakeGateway{Subs: map[string]*infrastructure.GatewaySubscription{}, Invoices: map[string]infrastructure.GatewayInvoice{}, Signature: true}
}

func (g *FakeGateway) CreateSubscription(req infrastructure.NewSubscription) (*infrastructure.GatewaySubscription, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Err != nil {
		return nil, g.Err
	}
	g.Created = append(g.Created, req)
	sub := &infrastructure.GatewaySubscription{
		Id: fmt.Sprintf("pre-%d", len(g.Created)), Status: "pending", ExternalReference: req.ExternalReference,
		AmountARS: req.AmountARS, InitPoint: "https://mp.test/checkout/" + req.Key,
	}
	g.Subs[sub.Id] = sub
	return sub, nil
}

func (g *FakeGateway) GetSubscription(id string) (*infrastructure.GatewaySubscription, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Err != nil {
		return nil, g.Err
	}
	sub, ok := g.Subs[id]
	if !ok {
		return nil, fmt.Errorf("Mercado Pago returned 404")
	}
	copied := *sub
	return &copied, nil
}

func (g *FakeGateway) GetInvoice(id string) (*infrastructure.GatewayInvoice, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Err != nil {
		return nil, g.Err
	}
	invoice, ok := g.Invoices[id]
	if !ok {
		return nil, fmt.Errorf("Mercado Pago returned 404")
	}
	return &invoice, nil
}

func (g *FakeGateway) UpdateAmount(id string, amountARS int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Err != nil {
		return g.Err
	}
	g.Subs[id].AmountARS = amountARS
	g.Updates = append(g.Updates, amountARS)
	return nil
}

func (g *FakeGateway) Cancel(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Err != nil {
		return g.Err
	}
	g.Subs[id].Status = "cancelled"
	g.Canceled = append(g.Canceled, id)
	return nil
}

func (g *FakeGateway) ValidSignature(string, string, string) bool {
	return g.Signature
}

// Authorize is the payer approving the subscription, next charge at next.
func (g *FakeGateway) Authorize(id string, next time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Subs[id].Status = "authorized"
	g.Subs[id].NextPayment = &next
}
