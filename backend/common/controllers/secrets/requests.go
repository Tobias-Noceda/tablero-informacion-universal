package secrets

import "github.com/Secreto31126/tesis/common/models"

type PutSecretRequest struct {
	CognitoID string            `json:"cognito_id" binding:"required"`
	Name      string            `json:"name" binding:"required"`
	Kind      models.SecretKind `json:"kind" binding:"required"`
	Value string `json:"value" binding:"required"`
}
