package billing

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/billing"
	"github.com/gin-gonic/gin"
)

// CURRENCY is what a subscription is charged in.
const CURRENCY = srv.CURRENCY

// RETURN_PATH is the frontend page the payer comes back to from the provider.
const RETURN_PATH = "/billing/return"

type Controller struct {
	plans   infrastructure.Entitlements
	billing *srv.Service
	gateway infrastructure.PaymentGateway
}

// NewController serves the billing routes. gateway is only asked to check
// webhook signatures and may be nil, which turns the webhook off.
func NewController(plans infrastructure.Entitlements, billing *srv.Service, gateway infrastructure.PaymentGateway) *Controller {
	return &Controller{plans, billing, gateway}
}

// RegisterPublicRoutes mounts what needs no session: the plans on offer and
// the provider's webhook, which proves itself by its signature.
func (ctrl *Controller) RegisterPublicRoutes(router gin.IRouter) {
	router.GET("/billing/plans", ctrl.Plans)
	router.POST("/billing/webhook", ctrl.Webhook)
}

// RegisterRoutes mounts what is about the caller, behind RequireAuth. None
// of it is open to personal access tokens.
func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	router.GET("/billing/usage", ctrl.Usage)
	router.GET("/billing/subscription", ctrl.Subscription)
	router.POST("/billing/subscribe", ctrl.Subscribe)
	router.POST("/billing/sync", ctrl.Sync)
	router.PUT("/billing/subscription", ctrl.ChangePlan)
	router.DELETE("/billing/subscription", ctrl.Cancel)
}

type PlanOffer = srv.PlanOffer
type PlansResponse = srv.PlansResponse

// Plans godoc
// @Summary      List the plans on offer
// @Description  Public. What each plan allows and costs. checkout is false while no payment provider is configured: plans cannot be bought.
// @Tags         billing
// @Produce      json
// @Success      200  {object}  PlansResponse
// @Router       /billing/plans [get]
func (ctrl *Controller) Plans(c *gin.Context) {
	c.JSON(http.StatusOK, ctrl.billing.Plans())
}

// Usage godoc
// @Summary      What the caller has used of their plan
// @Description  The organizations and groups they own, the secrets billed to them and their own post-it templates.
// @Tags         billing
// @Produce      json
// @Success      200  {object}  models.Usage
// @Failure      401  {object}  map[string]string{"error": "string"}
// @Router       /billing/usage [get]
func (ctrl *Controller) Usage(c *gin.Context) {
	usage, err := ctrl.plans.Usage(middleware.Principal(c).ID)
	if err != nil {
		log.Printf("billing: usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// fail answers a billing error. Whatever the provider said stays in the log.
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrInvalidPlan), errors.Is(err, srv.ErrInvalidEmail):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, srv.ErrAlreadySubscribed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, srv.ErrNoSubscription), errors.Is(err, srv.ErrUnknownSubscription), errors.Is(err, srv.ErrForeignSubscription):
		c.JSON(http.StatusNotFound, gin.H{"error": srv.ErrNoSubscription.Error()})
	case errors.Is(err, srv.ErrDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": srv.ErrDisabled.Error()})
	default:
		log.Printf("billing: %s %s: %v", c.Request.Method, c.FullPath(), err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "billing_failed"})
	}
}

// Subscription godoc
// @Summary      The caller's plan and subscription
// @Description  plan is the one in force now; status is none for someone who never subscribed.
// @Tags         billing
// @Produce      json
// @Success      200  {object}  models.SubscriptionView
// @Router       /billing/subscription [get]
func (ctrl *Controller) Subscription(c *gin.Context) {
	view, err := ctrl.billing.Subscription(middleware.Principal(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type SubscribeRequest struct {
	Plan       models.Plan `json:"plan"`
	PayerEmail string      `json:"payer_email"`
}

type SubscribeResponse struct {
	InitPoint string `json:"init_point"`
}

// Subscribe godoc
// @Summary      Start a monthly subscription
// @Description  Answers where the payer approves it. The plan changes once the provider reports it authorized.
// @Tags         billing
// @Accept       json
// @Produce      json
// @Param        request  body      SubscribeRequest  true  "pro or max, and the payer's Mercado Pago email"
// @Success      200      {object}  SubscribeResponse
// @Failure      400      {object}  map[string]string{"error": "invalid_plan | invalid_email"}
// @Failure      409      {object}  map[string]string{"error": "already_subscribed"}
// @Failure      503      {object}  map[string]string{"error": "billing_disabled"}
// @Router       /billing/subscribe [post]
func (ctrl *Controller) Subscribe(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalidPlan.Error()})
		return
	}

	point, err := ctrl.billing.Subscribe(middleware.Principal(c), req.Plan, req.PayerEmail, origin(c)+RETURN_PATH)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, SubscribeResponse{InitPoint: point})
}

// Sync godoc
// @Summary      Ask the provider where the caller's subscription stands
// @Description  What the return page calls, so the plan does not wait for a webhook.
// @Tags         billing
// @Produce      json
// @Success      200  {object}  models.SubscriptionView
// @Router       /billing/sync [post]
func (ctrl *Controller) Sync(c *gin.Context) {
	view, err := ctrl.billing.SyncMine(middleware.Principal(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

type ChangePlanRequest struct {
	Plan models.Plan `json:"plan"`
}

// ChangePlan godoc
// @Summary      Move the subscription to another plan
// @Description  An upgrade applies now; a downgrade from the next payment on.
// @Tags         billing
// @Accept       json
// @Produce      json
// @Param        request  body      ChangePlanRequest  true  "pro or max"
// @Success      200      {object}  models.SubscriptionView
// @Failure      400      {object}  map[string]string{"error": "invalid_plan"}
// @Failure      404      {object}  map[string]string{"error": "no_subscription"}
// @Router       /billing/subscription [put]
func (ctrl *Controller) ChangePlan(c *gin.Context) {
	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalidPlan.Error()})
		return
	}

	view, err := ctrl.billing.ChangePlan(middleware.Principal(c), req.Plan)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

// Cancel godoc
// @Summary      Cancel the subscription
// @Description  The plan stays until the date it already runs to.
// @Tags         billing
// @Success      204
// @Failure      404  {object}  map[string]string{"error": "no_subscription"}
// @Router       /billing/subscription [delete]
func (ctrl *Controller) Cancel(c *gin.Context) {
	if err := ctrl.billing.Cancel(middleware.Principal(c)); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Topics of the provider's notifications that change what a user has.
const (
	topicSubscription = "subscription_preapproval"
	topicInvoice      = "subscription_authorized_payment"
)

// Webhook godoc
// @Summary      The payment provider's notifications
// @Description  Public, proven by the x-signature header. Handled before answering, since nothing can be relied on to run after the response. The body is never trusted: the subscription or charge is fetched again.
// @Tags         billing
// @Success      200
// @Failure      400  {object}  map[string]string{"error": "missing_id"}
// @Failure      401  {object}  map[string]string{"error": "invalid_signature"}
// @Router       /billing/webhook [post]
func (ctrl *Controller) Webhook(c *gin.Context) {
	if ctrl.gateway == nil || !ctrl.billing.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": srv.ErrDisabled.Error()})
		return
	}

	// A notification is a few bytes; the cap only stops abuse.
	raw, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<16))
	var body struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(raw, &body)

	// Only the id in the query is signed, so only it is used: the body's is
	// anyone's to write.
	id := c.Query("data.id")
	if !ctrl.gateway.ValidSignature(c.GetHeader("X-Signature"), c.GetHeader("X-Request-Id"), id) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_signature"})
		return
	}
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_id"})
		return
	}

	topic := c.Query("type")
	if topic == "" {
		topic = c.Query("topic")
	}
	if topic == "" {
		topic = body.Type
	}

	var err error
	switch topic {
	case topicSubscription:
		_, err = ctrl.billing.Sync(id)
	case topicInvoice:
		_, err = ctrl.billing.SyncInvoice(id)
	default:
		c.Status(http.StatusOK)
		return
	}

	switch {
	case err == nil:
	case errors.Is(err, srv.ErrUnknownSubscription), errors.Is(err, srv.ErrForeignSubscription):
		// Not ours or not ours to act on: retrying will not change that.
		log.Printf("billing: webhook %s %s ignored: %v", topic, id, err)
	default:
		// The provider retries a failure, so say it was one.
		log.Printf("billing: webhook %s %s: %v", topic, id, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "billing_failed"})
		return
	}
	c.Status(http.StatusOK)
}

// origin is where the browser is, which is where the payer comes back to: the
// proxy tells us the scheme, the Host header the rest.
func origin(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	return scheme + "://" + c.Request.Host
}
