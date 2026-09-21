package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ENV_PLATFORM_ADMINS    = "PLATFORM_ADMINS"
	ENV_AUTH_COOKIE_SECURE = "AUTH_COOKIE_SECURE"
	ENV_ACCESS_TTL         = "ACCESS_TTL"
	ENV_REFRESH_TTL        = "REFRESH_TTL"

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
}

func loadConfig() (config, error) {
	cfg := config{
		admins:       splitList(os.Getenv(ENV_PLATFORM_ADMINS)),
		cookieSecure: true,
		accessTTL:    DEFAULT_ACCESS_TTL,
		refreshTTL:   DEFAULT_REFRESH_TTL,
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

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
