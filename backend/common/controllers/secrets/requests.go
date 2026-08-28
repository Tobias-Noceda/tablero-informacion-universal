package secrets

import "github.com/Secreto31126/tesis/common/models"

type PutSecretRequest struct {
	CognitoID string            `json:"cognito_id" binding:"required"`
	Name      string            `json:"name" binding:"required"`
	Kind      models.SecretKind `json:"kind" binding:"required"`
	Value     string            `json:"value" binding:"required"`
}

type PutOAuth2Request struct {
	CognitoID string `json:"cognito_id" binding:"required"`
	Name      string `json:"name" binding:"required"`

	Flow     string `json:"flow" binding:"required"`
	ClientID string `json:"client_id" binding:"required"`
	// Write-only, like any other secret value.
	ClientSecret string `json:"client_secret" binding:"required"`
	TokenURL     string `json:"token_url" binding:"required"`
	AuthURL      string `json:"auth_url"`
	Scopes       string `json:"scopes"`
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
