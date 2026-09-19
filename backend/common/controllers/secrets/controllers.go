package secrets

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.SecretsService
}

func NewController(service *srv.SecretsService) *Controller {
	return &Controller{
		service: service,
	}
}

// scoping tells a handler which scope a route addresses and whether the
// caller has to identify itself. Replacing cognito_id with an authenticated
// subject means changing principal() and nothing else here.
type scoping struct {
	scope             func(*gin.Context) (models.SecretScope, bool)
	principalRequired bool
}

func (s scoping) principal(c *gin.Context, cognitoID string) (models.Principal, bool) {
	if cognitoID == "" && s.principalRequired {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing cognito_id",
		})
		return models.Principal{}, false
	}

	return models.Principal{ID: cognitoID}, true
}

func boardScope(c *gin.Context) (models.SecretScope, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return models.SecretScope{}, false
	}

	return models.BoardScope(id), true
}

func userScope(c *gin.Context) (models.SecretScope, bool) {
	return models.UserScope(c.Param("id")), true
}

func systemScope(*gin.Context) (models.SecretScope, bool) {
	return models.SystemScope, true
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	ctrl.registerScoped(router.Group("/boards/:id"), scoping{scope: boardScope, principalRequired: true})
	ctrl.registerScoped(router.Group("/users/:id"), scoping{scope: userScope, principalRequired: true})

	system := scoping{scope: systemScope}
	systemGroup := router.Group("/system")
	{
		systemGroup.GET("/secrets", ctrl.ListSystemSecrets)
		systemGroup.PUT("/secrets", ctrl.PutSecret(system))
		systemGroup.DELETE("/secrets/:name", ctrl.DeleteSecret(system))
		systemGroup.PUT("/oauth2", ctrl.PutOAuth2(system))
		systemGroup.PUT("/oauth2/clients", ctrl.PutOAuth2Client)
		systemGroup.GET("/keys", ctrl.ListKeys)
	}

	router.GET("/oauth2/callback", ctrl.Callback)
	router.GET("/oauth2/providers", ctrl.Providers)
}

func (ctrl *Controller) registerScoped(group *gin.RouterGroup, s scoping) {
	group.GET("/secrets", ctrl.ListSecrets(s))
	group.PUT("/secrets", ctrl.PutSecret(s))
	group.DELETE("/secrets/:name", ctrl.DeleteSecret(s))

	group.PUT("/oauth2", ctrl.PutOAuth2(s))
	group.GET("/oauth2/authorize", ctrl.Authorize(s))
	group.POST("/oauth2/connect", ctrl.Connect(s))
}

func fail(c *gin.Context, err error) {
	if errors.Is(err, srv.ErrForbidden) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Not found",
		})
		return
	}

	if errors.Is(err, srv.ErrProviderNotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Validation failures are the caller's fault, not the server's.
	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}

// ListSecrets godoc
// @Summary      List the secret names of a scope
// @Description  Returns metadata only. Secret values are never returned by this API.
// @Tags         secrets
// @Produce      json
// @Param        id          path      string  true  "Board UUID or user id"
// @Param        cognito_id  query     string  true  "AWS Cognito User ID"
// @Success      200         {array}   models.SecretMeta
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /boards/{id}/secrets [get]
// @Router       /users/{id}/secrets [get]
func (ctrl *Controller) ListSecrets(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		principal, ok := s.principal(c, c.Query("cognito_id"))
		if !ok {
			return
		}

		metas, err := ctrl.service.List(scope, principal)
		if err != nil {
			fail(c, err)
			return
		}

		c.JSON(http.StatusOK, metas)
	}
}

// ListSystemSecrets godoc
// @Summary      List the platform's secrets against what the code expects
// @Description  Every name the code references, flagged as configured or not, plus any stored extra.
// @Tags         secrets
// @Produce      json
// @Success      200  {array}  models.SystemSecretStatus
// @Router       /system/secrets [get]
func (ctrl *Controller) ListSystemSecrets(c *gin.Context) {
	statuses, err := ctrl.service.ListSystem(models.Principal{ID: c.Query("cognito_id")})
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, statuses)
}

// ListKeys godoc
// @Summary      List data keys
// @Description  One entry per key, active or retired, with the master key version that wraps it. Never key material.
// @Tags         secrets
// @Produce      json
// @Success      200  {array}  models.DataKey
// @Router       /system/keys [get]
func (ctrl *Controller) ListKeys(c *gin.Context) {
	keys, err := ctrl.service.ListKeys(models.Principal{ID: c.Query("cognito_id")})
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, keys)
}

// PutSecret godoc
// @Summary      Create or replace a secret
// @Description  Managers only. The value is encrypted before storage and cannot be read back.
// @Tags         secrets
// @Accept       json
// @Param        id       path  string            true  "Board UUID or user id"
// @Param        request  body  PutSecretRequest  true  "Secret payload"
// @Success      204
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /boards/{id}/secrets [put]
// @Router       /users/{id}/secrets [put]
// @Router       /system/secrets [put]
func (ctrl *Controller) PutSecret(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		var req PutSecretRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		principal, ok := s.principal(c, req.CognitoID)
		if !ok {
			return
		}

		if err := ctrl.service.Put(scope, principal, req.Name, req.Kind, req.Value); err != nil {
			fail(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// DeleteSecret godoc
// @Summary      Delete a secret
// @Description  Managers only.
// @Tags         secrets
// @Param        id          path   string  true  "Board UUID or user id"
// @Param        name        path   string  true  "Secret name"
// @Param        cognito_id  query  string  true  "AWS Cognito User ID"
// @Success      204
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /boards/{id}/secrets/{name} [delete]
// @Router       /users/{id}/secrets/{name} [delete]
// @Router       /system/secrets/{name} [delete]
func (ctrl *Controller) DeleteSecret(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		principal, ok := s.principal(c, c.Query("cognito_id"))
		if !ok {
			return
		}

		if err := ctrl.service.Delete(scope, principal, c.Param("name")); err != nil {
			fail(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// PutOAuth2 godoc
// @Summary      Create or replace an OAuth2 credential
// @Description  Managers only. Tokens are never accepted from the caller.
// @Tags         secrets
// @Accept       json
// @Param        id       path  string            true  "Board UUID or user id"
// @Param        request  body  PutOAuth2Request  true  "OAuth2 configuration"
// @Success      204
// @Router       /boards/{id}/oauth2 [put]
// @Router       /users/{id}/oauth2 [put]
// @Router       /system/oauth2 [put]
func (ctrl *Controller) PutOAuth2(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		var req PutOAuth2Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		principal, ok := s.principal(c, req.CognitoID)
		if !ok {
			return
		}

		if err := ctrl.service.PutOAuth2(scope, principal, req.Name, req.Material()); err != nil {
			fail(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

// Authorize godoc
// @Summary      Start an OAuth2 authorization code handshake
// @Description  Managers only. Responds with the provider URL to send the user to.
// @Tags         secrets
// @Produce      json
// @Param        id           path   string  true  "Board UUID or user id"
// @Param        name         query  string  true  "Credential name"
// @Param        cognito_id   query  string  true  "AWS Cognito User ID"
// @Param        redirect_uri query  string  true  "Callback URL registered with the provider"
// @Success      200  {object}  map[string]string
// @Router       /boards/{id}/oauth2/authorize [get]
// @Router       /users/{id}/oauth2/authorize [get]
func (ctrl *Controller) Authorize(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		name, hasName := c.GetQuery("name")
		redirect, hasRedirect := c.GetQuery("redirect_uri")
		if !hasName || !hasRedirect {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Missing name or redirect_uri",
			})
			return
		}

		principal, ok := s.principal(c, c.Query("cognito_id"))
		if !ok {
			return
		}

		target, err := ctrl.service.Authorize(scope, principal, name, redirect)
		if err != nil {
			fail(c, err)
			return
		}

		// Returned rather than redirected so the caller decides how to navigate.
		c.JSON(http.StatusOK, gin.H{"authorization_url": target})
	}
}

// Connect godoc
// @Summary      Connect a provider account using the platform's own application
// @Description  Managers only. Stores an empty grant and responds with the provider URL to send the user to.
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Param        id       path  string          true  "Board UUID or user id"
// @Param        request  body  ConnectRequest  true  "Provider, grant name and callback URL"
// @Success      200  {object}  map[string]string
// @Router       /boards/{id}/oauth2/connect [post]
// @Router       /users/{id}/oauth2/connect [post]
func (ctrl *Controller) Connect(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s.scope(c)
		if !ok {
			return
		}

		var req ConnectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		principal, ok := s.principal(c, req.CognitoID)
		if !ok {
			return
		}

		target, err := ctrl.service.Connect(scope, principal, req.Provider, req.Name, req.RedirectURI)
		if err != nil {
			fail(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"authorization_url": target})
	}
}

// PutOAuth2Client godoc
// @Summary      Register the platform's application at a provider
// @Description  The client id and secret are stored encrypted in the system scope and never returned.
// @Tags         secrets
// @Accept       json
// @Param        request  body  PutOAuth2ClientRequest  true  "Provider application credential"
// @Success      204
// @Router       /system/oauth2/clients [put]
func (ctrl *Controller) PutOAuth2Client(c *gin.Context) {
	var req PutOAuth2ClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := ctrl.service.PutOAuth2Client(models.Principal{ID: c.Query("cognito_id")}, req.Provider, req.ClientID, req.ClientSecret); err != nil {
		fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Providers godoc
// @Summary      List the providers users can connect to
// @Description  Every provider the platform knows, flagged by whether its application credential is provisioned.
// @Tags         secrets
// @Produce      json
// @Success      200  {array}  models.OAuthProviderStatus
// @Router       /oauth2/providers [get]
func (ctrl *Controller) Providers(c *gin.Context) {
	statuses, err := ctrl.service.Providers()
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, statuses)
}

// Callback godoc
// @Summary      Finish an OAuth2 authorization code handshake
// @Description  Called by the provider. Authenticated by the state value alone.
// @Tags         secrets
// @Param        state  query  string  true  "Opaque state issued by Authorize"
// @Param        code   query  string  true  "Authorization code"
// @Success      204
// @Router       /oauth2/callback [get]
func (ctrl *Controller) Callback(c *gin.Context) {
	if providerError, failed := c.GetQuery("error"); failed {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provider denied the authorization: " + providerError,
		})
		return
	}

	if err := ctrl.service.Callback(c.Query("state"), c.Query("code")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
