package tokens

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	srv "github.com/Secreto31126/tesis/common/services/tokens"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.TokenService
}

func NewController(service *srv.TokenService) *Controller {
	return &Controller{service}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	tokens := router.Group("/users/:id/tokens")
	{
		tokens.GET("", ctrl.ListTokens)
		tokens.POST("", ctrl.CreateToken)
		tokens.DELETE("/:token", ctrl.RevokeToken)
	}
}

type CreateTokenRequest struct {
	Name string `json:"name" binding:"required"`
	// Days is the lifetime: 7, 15 or 30. Omitted means 15.
	Days int `json:"days"`
}

// ListTokens godoc
// @Summary      List one's personal access tokens
// @Description  Never includes the tokens themselves, only how to tell them apart.
// @Tags         tokens
// @Param        id  path  string  true  "Own user id"
// @Success      200  {array}  models.AccessToken
// @Failure      404  "Not one's own id"
// @Router       /users/{id}/tokens [get]
func (ctrl *Controller) ListTokens(c *gin.Context) {
	if !self(c) {
		return
	}

	tokens, err := ctrl.service.List(middleware.Principal(c))
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// CreateToken godoc
// @Summary      Create a personal access token
// @Description  Read-only, for developer tools such as an MCP client. The token is in this response only.
// @Tags         tokens
// @Accept       json
// @Param        id    path  string              true  "Own user id"
// @Param        body  body  CreateTokenRequest  true  "Name and lifetime in days"
// @Success      201
// @Failure      400  "invalid_name or invalid_expiry"
// @Failure      404  "Not one's own id"
// @Failure      409  "too_many_tokens"
// @Failure      429  "rate_limited"
// @Router       /users/{id}/tokens [post]
func (ctrl *Controller) CreateToken(c *gin.Context) {
	if !self(c) {
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalidName.Error()})
		return
	}

	created, err := ctrl.service.Create(middleware.Principal(c), req.Name, req.Days)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// RevokeToken godoc
// @Summary      Revoke a personal access token
// @Description  Takes effect on the token's next request.
// @Tags         tokens
// @Param        id     path  string  true  "Own user id"
// @Param        token  path  string  true  "Token id"
// @Success      204
// @Failure      404
// @Router       /users/{id}/tokens/{token} [delete]
func (ctrl *Controller) RevokeToken(c *gin.Context) {
	if !self(c) {
		return
	}

	id, err := uuid.Parse(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := ctrl.service.Revoke(middleware.Principal(c), id); err != nil {
		fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// self answers 404 unless the path names the caller: tokens are nobody
// else's business.
func self(c *gin.Context) bool {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil || id.String() != middleware.Principal(c).ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return false
	}
	return true
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrInvalidName), errors.Is(err, srv.ErrInvalidExpiry):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, srv.ErrTooManyTokens):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, infrastructure.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
	case errors.Is(err, srv.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "token_not_allowed"})
	case errors.Is(err, infrastructure.ErrAccessTokenNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
