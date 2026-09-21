package main

import (
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	for _, env := range []string{ENV_PLATFORM_ADMINS, ENV_AUTH_COOKIE_SECURE, ENV_ACCESS_TTL, ENV_REFRESH_TTL} {
		t.Setenv(env, "")
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.admins) != 0 || !cfg.cookieSecure || cfg.accessTTL != DEFAULT_ACCESS_TTL || cfg.refreshTTL != DEFAULT_REFRESH_TTL {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoadConfig_ReadsEverything(t *testing.T) {
	t.Setenv(ENV_PLATFORM_ADMINS, " ana@example.com, ,bob@example.com ")
	t.Setenv(ENV_AUTH_COOKIE_SECURE, "false")
	t.Setenv(ENV_ACCESS_TTL, "5m")
	t.Setenv(ENV_REFRESH_TTL, "48h")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.admins) != 2 || cfg.admins[0] != "ana@example.com" || cfg.cookieSecure || cfg.accessTTL != 5*time.Minute || cfg.refreshTTL != 48*time.Hour {
		t.Errorf("cfg = %+v", cfg)
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
