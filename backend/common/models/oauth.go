package models

import "time"

const (
	OAuth2ClientCredentials = "client_credentials"
	OAuth2AuthorizationCode = "authorization_code"
)

// OAuthProvider names a service the platform has registered its own
// application with. Its endpoints and scopes are configuration and live
// here; its client id and secret are a system secret.
type OAuthProvider string

const (
	ProviderGoogle  OAuthProvider = "google"
	ProviderDiscord OAuthProvider = "discord"
)

type OAuthProviderConfig struct {
	AuthURL    string
	TokenURL   string
	Scopes     string
	Credential SystemSecretName
}

var OAuthProviders = map[OAuthProvider]OAuthProviderConfig{
	ProviderGoogle: {
		AuthURL:    "https://accounts.google.com/o/oauth2/v2/auth?access_type=offline&prompt=consent",
		TokenURL:   "https://oauth2.googleapis.com/token",
		Scopes:     "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/calendar.readonly",
		Credential: SystemGoogleOAuthClient,
	},
	ProviderDiscord: {
		AuthURL:    "https://discord.com/oauth2/authorize",
		TokenURL:   "https://discord.com/api/oauth2/token",
		Scopes:     "identify guilds",
		Credential: SystemDiscordOAuthClient,
	},
}

// OAuth2Client is the platform's own application credential at a provider,
// stored as a system secret of kind oauth2_client.
type OAuth2Client struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type OAuthProviderStatus struct {
	Provider   OAuthProvider `json:"provider"`
	Configured bool          `json:"configured"`
}

type OAuth2Material struct {
	Flow         string `json:"flow"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenURL     string `json:"token_url"`
	// Where the user is sent to consent. authorization_code only.
	AuthURL string `json:"auth_url,omitempty"`
	Scopes  string `json:"scopes,omitempty"`

	// Set when the grant was obtained with the platform's own application.
	// The client fields above are then never persisted: they are loaded from
	// the provider registry and the system scope right before use.
	Provider OAuthProvider `json:"provider,omitempty"`

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

// Hydrate fills the application side of a provider grant from the registry
// and the platform's client credential.
func (m *OAuth2Material) Hydrate(config OAuthProviderConfig, client *OAuth2Client) {
	m.ClientID = client.ClientID
	m.ClientSecret = client.ClientSecret
	m.TokenURL = config.TokenURL
	m.AuthURL = config.AuthURL
	m.Scopes = config.Scopes
}

// Persistable is what may be written to disk: a provider grant never carries
// the platform's credentials, a self-managed one carries its own.
func (m *OAuth2Material) Persistable() *OAuth2Material {
	stored := *m
	if stored.Provider != "" {
		stored.ClientID = ""
		stored.ClientSecret = ""
		stored.TokenURL = ""
		stored.AuthURL = ""
		stored.Scopes = ""
	}
	return &stored
}

func (m *OAuth2Material) Header() string {
	kind := m.TokenType
	if kind == "" {
		kind = "Bearer"
	}

	return kind + " " + m.AccessToken
}
