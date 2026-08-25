package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
)

// DEFAULT_TTL applies when a provider omits expires_in, which the spec allows.
const DEFAULT_TTL = 5 * time.Minute

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type Client struct{}

func New() *Client {
	return &Client{}
}

func (c *Client) Fetch(material *models.OAuth2Material) error {
	form := url.Values{}

	switch material.Flow {
	case models.OAuth2ClientCredentials:
		form.Set("grant_type", models.OAuth2ClientCredentials)
		if material.Scopes != "" {
			form.Set("scope", material.Scopes)
		}
	case models.OAuth2AuthorizationCode:
		if material.RefreshToken == "" {
			return fmt.Errorf("Credential has not been authorized yet")
		}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", material.RefreshToken)
	default:
		return fmt.Errorf("Unsupported OAuth2 flow")
	}

	res, err := c.post(material, form)
	if err != nil {
		return err
	}

	material.AccessToken = res.AccessToken
	material.TokenType = res.TokenType

	ttl := time.Duration(res.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = DEFAULT_TTL
	}
	material.ExpiresAt = time.Now().Add(ttl)

	if res.RefreshToken != "" {
		material.RefreshToken = res.RefreshToken
	}

	return nil
}

// Exchange trades an authorization code for the first token pair.
func (c *Client) Exchange(material *models.OAuth2Material, code, redirectURI, verifier string) error {
	form := url.Values{}
	form.Set("grant_type", models.OAuth2AuthorizationCode)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}

	res, err := c.post(material, form)
	if err != nil {
		return err
	}

	if res.RefreshToken == "" {
		return fmt.Errorf("Provider returned no refresh token")
	}

	material.AccessToken = res.AccessToken
	material.TokenType = res.TokenType
	material.RefreshToken = res.RefreshToken

	ttl := time.Duration(res.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = DEFAULT_TTL
	}
	material.ExpiresAt = time.Now().Add(ttl)

	return nil
}

func (c *Client) post(material *models.OAuth2Material, form url.Values) (*tokenResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), safehttp.REQUEST_TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, material.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// RFC 6749 requires every token endpoint to accept client_secret_basic,
	// so it is the one form that works everywhere.
	req.SetBasicAuth(url.QueryEscape(material.ClientID), url.QueryEscape(material.ClientSecret))

	res, err := safehttp.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body := io.LimitReader(res.Body, safehttp.MAX_PAYLOAD_SIZE)

	if res.StatusCode != http.StatusOK {
		// The body of a failed token request tends to echo the credential back.
		return nil, fmt.Errorf("Token endpoint returned %d", res.StatusCode)
	}

	var parsed tokenResponse
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return nil, err
	}

	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("Token endpoint returned no access token")
	}

	return &parsed, nil
}
