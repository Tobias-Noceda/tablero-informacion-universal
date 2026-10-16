package main

import (
	"fmt"
	netmail "net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/mail"
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

	DEFAULT_ACCESS_TTL  = 15 * time.Minute
	DEFAULT_REFRESH_TTL = 30 * 24 * time.Hour
	AUTH_COOKIE_PATH    = "/api/v1/auth"
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

// identityProvider is Google sign-in, or nil when no client is configured.
func (cfg config) identityProvider() infrastructure.IdentityProvider {
	if cfg.google.ClientID == "" {
		return nil
	}
	return oidc.New(cfg.google)
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
