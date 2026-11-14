// Package mercadopago is the payment gateway over Mercado Pago's official Go
// SDK (github.com/mercadopago/sdk-go): subscriptions are "preapprovals". Every
// call goes through the SSRF-safe client and never repeats an error body,
// which can carry payer data.
package mercadopago

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/invoice"
	"github.com/mercadopago/sdk-go/pkg/mperror"
	"github.com/mercadopago/sdk-go/pkg/preapproval"
	"github.com/mercadopago/sdk-go/pkg/requester"
	"github.com/mercadopago/sdk-go/pkg/requestoptions"
	"github.com/mercadopago/sdk-go/pkg/webhook"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
)

const CURRENCY = "ARS"

// MaxSignatureAge is how far from now a webhook's ts may be, either way, so a
// captured call cannot be replayed.
const MaxSignatureAge = 5 * time.Minute

type Gateway struct {
	webhookSecret string
	preapprovals  preapproval.Client
	invoices      invoice.Client
}

var _ infrastructure.PaymentGateway = (*Gateway)(nil)

// New is the gateway for the seller whose access token is token. Webhooks are
// checked against webhookSecret.
func New(token, webhookSecret string) *Gateway {
	return newGateway(token, webhookSecret, safehttp.Client())
}

// newGateway builds the gateway over any requester (tests point it at a local
// server, which the safe client refuses).
//
// config.WithTimeout is deliberately not used: it replaces the injected
// requester with the SDK's default one, dropping the SSRF protection. Every
// call has its own context deadline instead.
func newGateway(token, webhookSecret string, r requester.Requester) *Gateway {
	cfg, err := config.New(token, config.WithHTTPClient(r))
	if err != nil {
		// Only a nil requester fails, and none is ever passed.
		panic(err)
	}
	return &Gateway{webhookSecret: webhookSecret, preapprovals: preapproval.NewClient(cfg), invoices: invoice.NewClient(cfg)}
}

func ctxWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), safehttp.REQUEST_TIMEOUT)
}

// opaque turns an SDK error into a message that never carries a body.
func opaque(err error) error {
	var respErr *mperror.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode != 0 {
		return fmt.Errorf("Mercado Pago returned %d", respErr.StatusCode)
	}
	return fmt.Errorf("Mercado Pago request failed")
}

func subscription(p *preapproval.Response) *infrastructure.GatewaySubscription {
	sub := &infrastructure.GatewaySubscription{
		Id:                p.ID,
		Status:            p.Status,
		ExternalReference: p.ExternalReference,
		AmountARS:         int(p.AutoRecurring.TransactionAmount + 0.5),
		InitPoint:         p.InitPoint,
	}
	if !p.NextPaymentDate.IsZero() {
		next := p.NextPaymentDate
		sub.NextPayment = &next
	}
	return sub
}

func (g *Gateway) CreateSubscription(req infrastructure.NewSubscription) (*infrastructure.GatewaySubscription, error) {
	ctx, cancel := ctxWithTimeout()
	defer cancel()
	if req.Key != "" {
		ctx = requestoptions.WithIdempotencyKey(ctx, req.Key)
	}

	out, err := g.preapprovals.Create(ctx, preapproval.Request{
		Reason:            req.Reason,
		ExternalReference: req.ExternalReference,
		PayerEmail:        req.PayerEmail,
		BackURL:           req.BackURL,
		Status:            "pending",
		AutoRecurring: &preapproval.AutoRecurringRequest{
			Frequency:         1,
			FrequencyType:     "months",
			TransactionAmount: float64(req.AmountARS),
			CurrencyID:        CURRENCY,
		},
	})
	if err != nil {
		return nil, opaque(err)
	}
	if out == nil || out.ID == "" || out.InitPoint == "" {
		return nil, fmt.Errorf("Mercado Pago returned no subscription")
	}
	return subscription(out), nil
}

func (g *Gateway) GetSubscription(id string) (*infrastructure.GatewaySubscription, error) {
	ctx, cancel := ctxWithTimeout()
	defer cancel()

	out, err := g.preapprovals.Get(ctx, id)
	if err != nil {
		return nil, opaque(err)
	}
	if out == nil || out.ID == "" {
		return nil, fmt.Errorf("Mercado Pago returned no subscription")
	}
	return subscription(out), nil
}

func (g *Gateway) GetInvoice(id string) (*infrastructure.GatewayInvoice, error) {
	ctx, cancel := ctxWithTimeout()
	defer cancel()

	out, err := g.invoices.Get(ctx, id)
	if err != nil {
		return nil, opaque(err)
	}
	return &infrastructure.GatewayInvoice{
		Id:             strconv.Itoa(out.ID),
		SubscriptionId: out.PreapprovalID,
		Paid:           out.Payment.Status == "approved",
	}, nil
}

func (g *Gateway) UpdateAmount(id string, amountARS int) error {
	ctx, cancel := ctxWithTimeout()
	defer cancel()

	_, err := g.preapprovals.Update(ctx, id, preapproval.UpdateRequest{
		AutoRecurring: &preapproval.AutoRecurringUpdateRequest{TransactionAmount: float64(amountARS)},
	})
	if err != nil {
		return opaque(err)
	}
	return nil
}

func (g *Gateway) Cancel(id string) error {
	ctx, cancel := ctxWithTimeout()
	defer cancel()

	if _, err := g.preapprovals.Update(ctx, id, preapproval.UpdateRequest{Status: "cancelled"}); err != nil {
		return opaque(err)
	}
	return nil
}

// ValidSignature checks the x-signature header of a webhook call against the
// gateway's secret. It also refuses a call signed more than MaxSignatureAge
// from now, either way. Without a secret nothing is valid.
func (g *Gateway) ValidSignature(xSignature, xRequestID, dataID string) bool {
	return validSignature(xSignature, xRequestID, dataID, g.webhookSecret, time.Now)
}

func validSignature(xSignature, xRequestID, dataID, secret string, now func() time.Time) bool {
	if secret == "" {
		return false
	}
	// Mercado Pago signs the data id lowercased; the SDK takes it as given.
	return webhook.ValidateSignature(xSignature, xRequestID, strings.ToLower(dataID), secret,
		webhook.WithTolerance(MaxSignatureAge), webhook.WithNow(now)) == nil
}
