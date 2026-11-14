package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

// SubscriptionStore keeps what is known of each user's subscriptions.
type SubscriptionStore interface {
	// UpsertSubscription stores sub under its Id, replacing what was there.
	UpsertSubscription(sub *models.Subscription) error
	FindSubscription(id string) (*models.Subscription, error)
	// LatestSubscription is the user's most recently created one, or
	// ErrSubscriptionNotFound.
	LatestSubscription(user string) (*models.Subscription, error)
	// OpenSubscriptionOf is the user's subscription that is authorized or
	// paused, or ErrSubscriptionNotFound.
	OpenSubscriptionOf(user string) (*models.Subscription, error)
	// OpenSubscriptions lists every subscription that is not canceled.
	OpenSubscriptions() ([]models.Subscription, error)
}

// GatewaySubscription is a recurring charge as the payment provider reports it.
type GatewaySubscription struct {
	Id string
	// Status is the provider's word, canceled spelled either way.
	Status string
	// ExternalReference is what the subscription was created with:
	// "<user>:<plan>".
	ExternalReference string
	AmountARS         int
	NextPayment       *time.Time
	InitPoint         string
}

// GatewayInvoice is one charge of a subscription.
type GatewayInvoice struct {
	Id             string
	SubscriptionId string
	// Paid is whether the money was taken.
	Paid bool
}

// NewSubscription is what the provider needs to start a subscription.
type NewSubscription struct {
	// Key makes a retried request create it once.
	Key               string
	Reason            string
	ExternalReference string
	PayerEmail        string
	AmountARS         int
	BackURL           string
}

// PaymentGateway is the payment provider. Nothing it says about a payment is
// trusted from a webhook body: the service asks it again.
type PaymentGateway interface {
	CreateSubscription(req NewSubscription) (*GatewaySubscription, error)
	GetSubscription(id string) (*GatewaySubscription, error)
	GetInvoice(id string) (*GatewayInvoice, error)
	UpdateAmount(id string, amountARS int) error
	Cancel(id string) error
	// ValidSignature checks the headers and the data id of a webhook call
	// against the provider's shared secret.
	ValidSignature(xSignature, xRequestID, dataID string) bool
}
