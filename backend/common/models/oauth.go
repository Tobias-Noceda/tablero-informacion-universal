package models

import "time"

const (
	OAuth2ClientCredentials = "client_credentials"
	OAuth2AuthorizationCode = "authorization_code"
)

type OAuth2Material struct {
	Flow         string `json:"flow"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenURL     string `json:"token_url"`
	// Where the user is sent to consent. authorization_code only.
	AuthURL string `json:"auth_url,omitempty"`
	Scopes  string `json:"scopes,omitempty"`

	// Only set once a user has completed an authorization_code grant.
	RefreshToken string `json:"refresh_token,omitempty"`

	// Cached result of the last token request.
	AccessToken string    `json:"access_token,omitempty"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// REFRESH_MARGIN renews a token before it actually expires, so a request is
// never sent with a credential that dies in flight.
const REFRESH_MARGIN = 60 * time.Second

func (m *OAuth2Material) NeedsRefresh() bool {
	return m.AccessToken == "" || time.Now().Add(REFRESH_MARGIN).After(m.ExpiresAt)
}

func (m *OAuth2Material) Header() string {
	kind := m.TokenType
	if kind == "" {
		kind = "Bearer"
	}

	return kind + " " + m.AccessToken
}
