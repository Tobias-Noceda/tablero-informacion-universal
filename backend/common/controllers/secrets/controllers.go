package secrets

import (
	"errors"
	"net/http"

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

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	boardGroup := router.Group("/boards")
	{
		boardGroup.GET("/:id/secrets", ctrl.ListSecrets)
		boardGroup.PUT("/:id/secrets", ctrl.PutSecret)
		boardGroup.DELETE("/:id/secrets/:name", ctrl.DeleteSecret)
	}
}

func boardID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return uuid.Nil, false
	}

	return id, true
}

func fail(c *gin.Context, err error) {
	if errors.Is(err, srv.ErrForbidden) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Board not found",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

// ListSecrets godoc
// @Summary      List a board's secret names
// @Description  Returns metadata only. Secret values are never returned by this API.
// @Tags         secrets
// @Produce      json
// @Param        id          path      string  true  "Board UUID"
// @Param        cognito_id  query     string  true  "AWS Cognito User ID"
// @Success      200         {array}   models.SecretMeta
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /boards/{id}/secrets [get]
func (ctrl *Controller) ListSecrets(c *gin.Context) {
	id, ok := boardID(c)
	if !ok {
		return
	}

	cognitoID, exists := c.GetQuery("cognito_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing cognito_id",
		})
		return
	}

	metas, err := ctrl.service.List(id, cognitoID)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, metas)
}

// PutSecret godoc
// @Summary      Create or replace a board secret
// @Description  Owner only. The value is encrypted before storage and cannot be read back.
// @Tags         secrets
// @Accept       json
// @Param        id       path  string            true  "Board UUID"
// @Param        request  body  PutSecretRequest  true  "Secret payload"
// @Success      204
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /boards/{id}/secrets [put]
func (ctrl *Controller) PutSecret(c *gin.Context) {
	id, ok := boardID(c)
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

	if err := ctrl.service.Put(id, req.CognitoID, req.Name, req.Kind, req.Value); err != nil {
		if errors.Is(err, srv.ErrForbidden) {
			fail(c, err)
			return
		}

		// Validation failures are the caller's fault, not the server's.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteSecret godoc
// @Summary      Delete a board secret
// @Description  Owner only.
// @Tags         secrets
// @Param        id          path   string  true  "Board UUID"
// @Param        name        path   string  true  "Secret name"
// @Param        cognito_id  query  string  true  "AWS Cognito User ID"
// @Success      204
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /boards/{id}/secrets/{name} [delete]
func (ctrl *Controller) DeleteSecret(c *gin.Context) {
	id, ok := boardID(c)
	if !ok {
		return
	}

	cognitoID, exists := c.GetQuery("cognito_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing cognito_id",
		})
		return
	}

	if err := ctrl.service.Delete(id, cognitoID, c.Param("name")); err != nil {
		fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
