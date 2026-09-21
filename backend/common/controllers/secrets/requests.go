package secrets

import "github.com/Secreto31126/tesis/common/models"

// Caller identifies who is making a write. Optional on the wire because the
// system scope has nobody to name yet; the policy decides whether an
// anonymous caller is acceptable for the scope at hand.
type Caller struct {
	CognitoID string `json:"cognito_id"`
}

func (c Caller) Principal() models.Principal {
	return models.Principal{ID: c.CognitoID}
}

type PutSecretRequest struct {
	Caller
	Name  string            `json:"name" binding:"required"`
	Kind  models.SecretKind `json:"kind" binding:"required"`
	Value string            `json:"value" binding:"required"`
}

type PutOAuth2Request struct {
	Caller
	Name string `json:"name" binding:"required"`

	Flow     string `json:"flow" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
	// Write-only, like any other secret value.
	ClientSecret string `json:"client_secret" binding:"required"`
	TokenURL     string `json:"token_url" binding:"required"`
	AuthURL      string `json:"auth_url"`
	Scopes       string `json:"scopes"`
}

type PutOAuth2ClientRequest struct {
	Provider models.OAuthProvider `json:"provider" binding:"required"`
	ClientID string               `json:"client_id" binding:"required"`
	// Write-only, like any other secret value.
	ClientSecret string `json:"client_secret" binding:"required"`
}

type ConnectRequest struct {
	Caller
	Provider    models.OAuthProvider `json:"provider" binding:"required"`
	Name        string               `json:"name" binding:"required"`
	RedirectURI string               `json:"redirect_uri" binding:"required"`
}

func (r *PutOAuth2Request) Material() *models.OAuth2Material {
	return &models.OAuth2Material{
		Flow:         r.Flow,
		ClientID:     r.ClientID,
		ClientSecret: r.ClientSecret,
		TokenURL:     r.TokenURL,
		AuthURL:      r.AuthURL,
		Scopes:       r.Scopes,
	}
}

type SetGrantsRequest struct {
	Caller
	Grants []models.Grant `json:"grants"`
}
