package auth

import "github.com/Secreto31126/tesis/common/models"

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TokenRequest struct {
	Token string `json:"token" binding:"required"`
}

type EmailRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SessionResponse carries the access token; the refresh token only ever
// travels in the cookie.
type SessionResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresIn   int            `json:"expires_in"`
	User        models.Profile `json:"user"`
}
