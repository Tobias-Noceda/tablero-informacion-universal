package main

import (
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/ports/oidc"
)

func TestLoadConfig_Defaults(t *testing.T) {
	for _, env := range []string{ENV_PLATFORM_ADMINS, ENV_AUTH_COOKIE_SECURE, ENV_ACCESS_TTL, ENV_REFRESH_TTL, ENV_GOOGLE_ISSUER, ENV_GOOGLE_CLIENT_ID, ENV_GOOGLE_CLIENT_SECRET} {
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
