package models

import (
	"strings"
	"time"
)

// SubscriptionStatus is where a subscription stands at the payment provider.
// "none" is only ever answered for a user who has never subscribed.
type SubscriptionStatus string

const (
	SubscriptionNone       SubscriptionStatus = "none"
	SubscriptionPending    SubscriptionStatus = "pending"
	SubscriptionAuthorized SubscriptionStatus = "authorized"
	SubscriptionPaused     SubscriptionStatus = "paused"
	SubscriptionCanceled   SubscriptionStatus = "canceled"
)

// ParseSubscriptionStatus reads the provider's word for a status, which
// spells canceled both ways. Anything unknown is pending: it grants nothing.
func ParseSubscriptionStatus(raw string) SubscriptionStatus {
	switch strings.ToLower(raw) {
	case "authorized":
		return SubscriptionAuthorized
	case "paused":
		return SubscriptionPaused
	case "canceled", "cancelled":
		return SubscriptionCanceled
	default:
		return SubscriptionPending
	}
}

// Open is whether the subscription can still charge: everything but canceled.
func (s SubscriptionStatus) Open() bool {
	return s != SubscriptionCanceled
}

// Subscription is a user's monthly charge at the payment provider. Its Id is
// the provider's preapproval id. Plan is what the charge pays for;
// PendingPlan is a downgrade that applies from the next charge.
type Subscription struct {
	Id          string             `bson:"_id" json:"id"`
	User        string             `bson:"user" json:"user"`
	Plan        Plan               `bson:"plan" json:"plan"`
	PendingPlan Plan               `bson:"pending_plan,omitempty" json:"pending_plan,omitempty"`
	Status      SubscriptionStatus `bson:"status" json:"status"`
	AmountARS   int                `bson:"amount_ars" json:"amount_ars"`
	NextPayment *time.Time         `bson:"next_payment,omitempty" json:"next_payment,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// SubscriptionView is what a user is told about their plan and charge.
type SubscriptionView struct {
	Plan        Plan               `json:"plan"`
	Status      SubscriptionStatus `json:"status"`
	PendingPlan Plan               `json:"pending_plan,omitempty"`
	AmountARS   int                `json:"amount_ars,omitempty"`
	NextPayment *time.Time         `json:"next_payment,omitempty"`
	Expires     *time.Time         `json:"expires,omitempty"`
}
