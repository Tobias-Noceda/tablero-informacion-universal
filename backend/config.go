package main

import (
	"fmt"
	"math"
	"net"
	netmail "net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/mail"
	"github.com/Secreto31126/tesis/common/ports/mercadopago"
	"github.com/Secreto31126/tesis/common/ports/oidc"
)

const (
	ENV_PLATFORM_ADMINS      = "PLATFORM_ADMINS"
	ENV_AUTH_COOKIE_SECURE   = "AUTH_COOKIE_SECURE"
	ENV_ACCESS_TTL           = "ACCESS_TTL"
	ENV_REFRESH_TTL          = "REFRESH_TTL"
	ENV_GOOGLE_ISSUER        = "GOOGLE_ISSUER"
	ENV_GOOGLE_CLIENT_ID     = "GOOGLE_SIGNIN_CLIENT_ID"
	ENV_GOOGLE_CLIENT_SECRET = "GOOGLE_SIGNIN_CLIENT_SECRET"
	ENV_RESEND_API_KEY       = "RESEND_API_KEY"
	ENV_MAIL_FROM            = "MAIL_FROM"
	ENV_MP_ACCESS_TOKEN      = "MP_ACCESS_TOKEN"
	ENV_MP_WEBHOOK_SECRET    = "MP_WEBHOOK_SECRET"
	ENV_BILLING_USD_ARS      = "BILLING_USD_ARS"
	// Set by Vercel on every deployment; the health route reports it.
	ENV_VERCEL_COMMIT = "VERCEL_GIT_COMMIT_SHA"
	SHORT_COMMIT      = 7
	ENV_VERCEL        = "VERCEL"
	// Comma-separated IPs or CIDRs whose X-Forwarded-For is believed, or
	// "none" to key on the connection's own address.
	ENV_TRUSTED_PROXIES = "TRUSTED_PROXIES"
	NO_PROXIES          = "none"

	DEFAULT_ACCESS_TTL  = 15 * time.Minute
	DEFAULT_REFRESH_TTL = 30 * 24 * time.Hour
	AUTH_COOKIE_PATH    = "/api/v1/auth"
)

var (
	// privateNetworks is who forwards requests on Docker: the ALB, which
	// reaches the backend over a private docker network. A caller outside
	// them cannot pass off another address.
	privateNetworks = []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1/128", "fc00::/7"}
	// everyNetwork is Vercel's default: every request goes through its
	// proxy, which replaces X-Forwarded-For with the caller's address.
	everyNetwork = []string{"0.0.0.0/0", "::/0"}
)

// config is every knob the service graph takes from the environment besides
// the connection strings the ports read themselves.
type config struct {
	admins       []string
	cookieSecure bool
	accessTTL    time.Duration
	refreshTTL   time.Duration
	google       oidc.Config
	resendAPIKey string
	mailFrom     string

	// Mercado Pago. Without an access token billing is off: plans are
	// listed, none can be bought.
	mpAccessToken   string
	mpWebhookSecret string
	billingUSDARS   float64
	// gateway replaces Mercado Pago when set; only tests do.
	gateway infrastructure.PaymentGateway

	// version is the deployed commit, empty where the platform does not
	// say (Docker).
	version string

	// trustedProxies are the addresses whose X-Forwarded-For names the
	// client the rate limits key on; nil trusts none.
	trustedProxies []string
}

func loadConfig() (config, error) {
	cfg := config{
		admins:       splitList(os.Getenv(ENV_PLATFORM_ADMINS)),
		cookieSecure: true,
		accessTTL:    DEFAULT_ACCESS_TTL,
		refreshTTL:   DEFAULT_REFRESH_TTL,
		google: oidc.Config{
			Issuer:       os.Getenv(ENV_GOOGLE_ISSUER),
			ClientID:     os.Getenv(ENV_GOOGLE_CLIENT_ID),
			ClientSecret: os.Getenv(ENV_GOOGLE_CLIENT_SECRET),
		},
		resendAPIKey: strings.TrimSpace(os.Getenv(ENV_RESEND_API_KEY)),
		mailFrom:     strings.TrimSpace(os.Getenv(ENV_MAIL_FROM)),

		mpAccessToken:   strings.TrimSpace(os.Getenv(ENV_MP_ACCESS_TOKEN)),
		mpWebhookSecret: strings.TrimSpace(os.Getenv(ENV_MP_WEBHOOK_SECRET)),

		version: shortCommit(os.Getenv(ENV_VERCEL_COMMIT)),
	}

	proxies, err := trustedProxies(os.Getenv(ENV_TRUSTED_PROXIES), os.Getenv(ENV_VERCEL) != "")
	if err != nil {
		return cfg, err
	}
	cfg.trustedProxies = proxies

	if raw := strings.TrimSpace(os.Getenv(ENV_BILLING_USD_ARS)); raw != "" {
		rate, err := strconv.ParseFloat(raw, 64)
		if err != nil || rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
			return cfg, fmt.Errorf("%s: expected a positive number of pesos per dollar, got %q", ENV_BILLING_USD_ARS, raw)
		}
		cfg.billingUSDARS = rate
	}

	if cfg.mpAccessToken != "" && (cfg.mpWebhookSecret == "" || cfg.billingUSDARS <= 0) {
		return cfg, fmt.Errorf("%s needs %s and %s: the secret that signs its webhooks and the exchange rate subscriptions are priced at", ENV_MP_ACCESS_TOKEN, ENV_MP_WEBHOOK_SECRET, ENV_BILLING_USD_ARS)
	}

	if (cfg.google.ClientID == "") != (cfg.google.ClientSecret == "") {
		return cfg, fmt.Errorf("%s and %s go together: set both to enable Google sign-in, or neither", ENV_GOOGLE_CLIENT_ID, ENV_GOOGLE_CLIENT_SECRET)
	}

	if cfg.resendAPIKey != "" && !validSender(cfg.mailFrom) {
		return cfg, fmt.Errorf("%s needs %s: an address on a domain verified in Resend, like \"Tablero <no-reply@example.com>\"", ENV_RESEND_API_KEY, ENV_MAIL_FROM)
	}

	if raw := os.Getenv(ENV_AUTH_COOKIE_SECURE); raw != "" {
		secure, err := strconv.ParseBool(raw)
		if err != nil {
			return cfg, fmt.Errorf("%s: %w", ENV_AUTH_COOKIE_SECURE, err)
		}
		cfg.cookieSecure = secure
	}

	for _, ttl := range []struct {
		env  string
		into *time.Duration
	}{{ENV_ACCESS_TTL, &cfg.accessTTL}, {ENV_REFRESH_TTL, &cfg.refreshTTL}} {
		raw := os.Getenv(ttl.env)
		if raw == "" {
			continue
		}
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return cfg, fmt.Errorf("%s: expected a positive duration, got %q", ttl.env, raw)
		}
		*ttl.into = parsed
	}

	return cfg, nil
}

// trustedProxies reads TRUSTED_PROXIES, defaulting to what the deployment
// puts in front of the backend.
func trustedProxies(raw string, onVercel bool) ([]string, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == NO_PROXIES:
		return nil, nil
	case raw == "" && onVercel:
		return everyNetwork, nil
	case raw == "":
		return privateNetworks, nil
	}

	proxies := splitList(raw)
	for _, proxy := range proxies {
		if _, _, err := net.ParseCIDR(proxy); err != nil && net.ParseIP(proxy) == nil {
			return nil, fmt.Errorf("%s: %q is neither an IP nor a CIDR", ENV_TRUSTED_PROXIES, proxy)
		}
	}
	return proxies, nil
}

func shortCommit(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > SHORT_COMMIT {
		return sha[:SHORT_COMMIT]
	}
	return sha
}

// identityProvider is Google sign-in, or nil when no client is configured.
func (cfg config) identityProvider() infrastructure.IdentityProvider {
	if cfg.google.ClientID == "" {
		return nil
	}
	return oidc.New(cfg.google)
}

// paymentGateway is Mercado Pago, or nil (billing off) without an access token.
func (cfg config) paymentGateway() infrastructure.PaymentGateway {
	if cfg.gateway != nil {
		return cfg.gateway
	}
	if cfg.mpAccessToken == "" {
		return nil
	}
	return mercadopago.New(cfg.mpAccessToken, cfg.mpWebhookSecret)
}

// mailSender is Resend, or the log when no API key is configured.
func (cfg config) mailSender() mail.Sender {
	if cfg.resendAPIKey == "" {
		return mail.LogSender{}
	}
	return mail.NewResendSender(cfg.resendAPIKey, cfg.mailFrom)
}

func validSender(from string) bool {
	parsed, err := netmail.ParseAddress(from)
	return err == nil && models.ValidEmail(parsed.Address)
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
