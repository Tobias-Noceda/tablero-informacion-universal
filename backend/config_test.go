package main

import (
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/ports/mail"
	"github.com/Secreto31126/tesis/common/ports/oidc"
)

func TestLoadConfig_Defaults(t *testing.T) {
	for _, env := range []string{ENV_PLATFORM_ADMINS, ENV_AUTH_COOKIE_SECURE, ENV_ACCESS_TTL, ENV_REFRESH_TTL, ENV_GOOGLE_ISSUER, ENV_GOOGLE_CLIENT_ID, ENV_GOOGLE_CLIENT_SECRET, ENV_RESEND_API_KEY, ENV_MAIL_FROM} {
		t.Setenv(env, "")
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.admins) != 0 || !cfg.cookieSecure || cfg.accessTTL != DEFAULT_ACCESS_TTL || cfg.refreshTTL != DEFAULT_REFRESH_TTL {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.identityProvider() != nil {
		t.Error("google sign-in must be off without a client")
	}
	if _, logs := cfg.mailSender().(mail.LogSender); !logs {
		t.Errorf("without a Resend key mails must go to the log, got %T", cfg.mailSender())
	}
}

func TestLoadConfig_ReadsEverything(t *testing.T) {
	t.Setenv(ENV_PLATFORM_ADMINS, " ana@example.com, ,bob@example.com ")
	t.Setenv(ENV_AUTH_COOKIE_SECURE, "false")
	t.Setenv(ENV_ACCESS_TTL, "5m")
	t.Setenv(ENV_REFRESH_TTL, "48h")
	t.Setenv(ENV_GOOGLE_ISSUER, "http://localhost:8899/default")
	t.Setenv(ENV_GOOGLE_CLIENT_ID, "client")
	t.Setenv(ENV_GOOGLE_CLIENT_SECRET, "secret")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.admins) != 2 || cfg.admins[0] != "ana@example.com" || cfg.cookieSecure || cfg.accessTTL != 5*time.Minute || cfg.refreshTTL != 48*time.Hour {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.google != (oidc.Config{Issuer: "http://localhost:8899/default", ClientID: "client", ClientSecret: "secret"}) || cfg.identityProvider() == nil {
		t.Errorf("google = %+v", cfg.google)
	}
}

func TestLoadConfig_RejectsBadValues(t *testing.T) {
	t.Setenv(ENV_AUTH_COOKIE_SECURE, "maybe")
	if _, err := loadConfig(); err == nil {
		t.Error("bad bool accepted")
	}
	t.Setenv(ENV_AUTH_COOKIE_SECURE, "")
	t.Setenv(ENV_ACCESS_TTL, "-1m")
	if _, err := loadConfig(); err == nil {
		t.Error("negative ttl accepted")
	}
}

func TestLoadConfig_GoogleClientNeedsBothHalves(t *testing.T) {
	for _, half := range []string{ENV_GOOGLE_CLIENT_ID, ENV_GOOGLE_CLIENT_SECRET} {
		t.Setenv(ENV_GOOGLE_CLIENT_ID, "")
		t.Setenv(ENV_GOOGLE_CLIENT_SECRET, "")
		t.Setenv(half, "only-this")
		if _, err := loadConfig(); err == nil {
			t.Errorf("only %s set was accepted", half)
		}
	}
}

func TestLoadConfig_SendsThroughResendWithAKey(t *testing.T) {
	t.Setenv(ENV_RESEND_API_KEY, "re_key")
	t.Setenv(ENV_MAIL_FROM, "Tablero <no-reply@tiu.example>")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, resend := cfg.mailSender().(*mail.ResendSender); !resend {
		t.Errorf("sender = %T", cfg.mailSender())
	}
}

// A key without a sender address would only fail at the first sign-up, so it
// stops the boot instead.
func TestLoadConfig_ResendNeedsAValidSender(t *testing.T) {
	t.Setenv(ENV_RESEND_API_KEY, "re_key")
	for _, from := range []string{"", "not an address", "Tablero <no-reply@>"} {
		t.Setenv(ENV_MAIL_FROM, from)
		if _, err := loadConfig(); err == nil {
			t.Errorf("%s=%q was accepted", ENV_MAIL_FROM, from)
		}
	}
}

func TestLoadConfig_Billing(t *testing.T) {
	clear := func() {
		for _, env := range []string{ENV_MP_ACCESS_TOKEN, ENV_MP_WEBHOOK_SECRET, ENV_BILLING_USD_ARS} {
			t.Setenv(env, "")
		}
	}

	clear()
	cfg, err := loadConfig()
	if err != nil || cfg.paymentGateway() != nil {
		t.Errorf("without a token billing must be off: %v %v", err, cfg.paymentGateway())
	}

	for name, env := range map[string]map[string]string{
		"token alone":          {ENV_MP_ACCESS_TOKEN: "tok"},
		"token without rate":   {ENV_MP_ACCESS_TOKEN: "tok", ENV_MP_WEBHOOK_SECRET: "s"},
		"token without secret": {ENV_MP_ACCESS_TOKEN: "tok", ENV_BILLING_USD_ARS: "1570"},
		"rate is not a number": {ENV_BILLING_USD_ARS: "much"},
		"rate is not positive": {ENV_BILLING_USD_ARS: "0"},
	} {
		clear()
		for k, v := range env {
			t.Setenv(k, v)
		}
		if _, err := loadConfig(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	clear()
	t.Setenv(ENV_MP_ACCESS_TOKEN, "tok")
	t.Setenv(ENV_MP_WEBHOOK_SECRET, "s")
	t.Setenv(ENV_BILLING_USD_ARS, " 1570.5 ")
	cfg, err = loadConfig()
	if err != nil || cfg.billingUSDARS != 1570.5 || cfg.paymentGateway() == nil {
		t.Errorf("complete: %+v %v", cfg, err)
	}

	// A rate and a secret without a token only price the plans for show.
	clear()
	t.Setenv(ENV_BILLING_USD_ARS, "1570")
	if cfg, err := loadConfig(); err != nil || cfg.paymentGateway() != nil || cfg.billingUSDARS != 1570 {
		t.Errorf("rate only: %+v %v", cfg, err)
	}
}
