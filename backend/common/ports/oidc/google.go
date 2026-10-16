package oidc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	DEFAULT_ISSUER  = "https://accounts.google.com"
	REQUEST_TIMEOUT = 10 * time.Second
)

type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
}

// Google signs users in through an OpenID Connect issuer, Google's unless the
// configuration says otherwise. The issuer is configuration, never user input,
// so it is reached with a plain client instead of the SSRF-safe one.
//
// Discovery happens on first use and is retried until it succeeds, so an
// unreachable issuer at boot only disables the sign-in for a while.
type Google struct {
	config Config
	client *http.Client

	mu       sync.Mutex
	provider *gooidc.Provider
	verifier *gooidc.IDTokenVerifier
}

var _ infrastructure.IdentityProvider = (*Google)(nil)

func New(config Config) *Google {
	if config.Issuer == "" {
		config.Issuer = DEFAULT_ISSUER
	}
	return &Google{config: config, client: &http.Client{Timeout: REQUEST_TIMEOUT}}
}

func (g *Google) AuthURL(state, nonce, codeChallenge, redirectURI string) (string, error) {
	oauth, _, err := g.discover(redirectURI)
	if err != nil {
		return "", err
	}

	return oauth.AuthCodeURL(state,
		gooidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}

func (g *Google) Complete(code, redirectURI, verifier, nonce string) (*infrastructure.ExternalIdentity, error) {
	oauth, idTokens, err := g.discover(redirectURI)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.context()
	defer cancel()

	token, err := oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, rejected("code exchange", err)
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, rejected("code exchange", fmt.Errorf("no id_token in the response"))
	}

	idToken, err := idTokens.Verify(ctx, raw)
	if err != nil {
		return nil, rejected("id_token", err)
	}
	if idToken.Nonce != nonce {
		return nil, rejected("id_token", fmt.Errorf("nonce mismatch"))
	}
	if idToken.Subject == "" {
		return nil, rejected("id_token", fmt.Errorf("no subject"))
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, rejected("id_token claims", err)
	}

	return &infrastructure.ExternalIdentity{
		Provider:      models.IdentityGoogle,
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		Picture:       claims.Picture,
	}, nil
}

func (g *Google) discover(redirectURI string) (*oauth2.Config, *gooidc.IDTokenVerifier, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.provider == nil {
		ctx, cancel := g.context()
		defer cancel()

		provider, err := gooidc.NewProvider(ctx, g.config.Issuer)
		if err != nil {
			return nil, nil, rejected("discovery", err)
		}
		g.provider = provider
		g.verifier = provider.VerifierContext(gooidc.ClientContext(context.Background(), g.client), &gooidc.Config{ClientID: g.config.ClientID})
	}

	return &oauth2.Config{
		ClientID:     g.config.ClientID,
		ClientSecret: g.config.ClientSecret,
		Endpoint:     g.provider.Endpoint(),
		RedirectURL:  redirectURI,
		Scopes:       []string{gooidc.ScopeOpenID, "email", "profile"},
	}, g.verifier, nil
}

func (g *Google) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(gooidc.ClientContext(context.Background(), g.client), REQUEST_TIMEOUT)
}

func rejected(step string, err error) error {
	return fmt.Errorf("%w: %s: %v", infrastructure.ErrIdentityProvider, step, err)
}
