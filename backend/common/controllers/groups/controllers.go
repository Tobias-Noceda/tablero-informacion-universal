package groups

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/groups"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.GroupService
}

func NewController(service *srv.GroupService) *Controller {
	return &Controller{service}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	groups := router.Group("/groups")
	{
		groups.POST("", ctrl.CreateGroup)
		groups.GET("", ctrl.ListMine)
		groups.GET("/:id", ctrl.GetGroup)
		groups.DELETE("/:id", ctrl.DeleteGroup)
		groups.POST("/:id/members", ctrl.AddMember)
		groups.DELETE("/:id/members", ctrl.RemoveMember)
	}
}

// A group the caller may not touch answers 404: whether it exists is not
// disclosed, the same rule the vault applies to scopes.
func fail(c *gin.Context, err error) {
	if errors.Is(err, srv.ErrForbidden) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Group not found",
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}

func groupID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return uuid.Nil, false
	}

	return id, true
}

func caller(c *gin.Context) (models.Principal, bool) {
	cognitoID := c.Query("cognito_id")
	if cognitoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing cognito_id",
		})
		return models.Principal{}, false
	}

	return models.Principal{ID: cognitoID}, true
}

// CreateGroup godoc
// @Summary      Create a group
// @Description  The caller becomes its owner. A group owns secrets its members may use on any board.
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      CreateGroupRequest  true  "Group payload"
// @Success      201      {object}  models.Group
// @Failure      400      {object}  map[string]string
// @Router       /groups [post]
func (ctrl *Controller) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	group, err := ctrl.service.Create(models.Principal{ID: req.CognitoID}, req.Name)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// ListMine godoc
// @Summary      List the groups the caller belongs to
// @Tags         groups
// @Produce      json
// @Param        cognito_id  query     string  true  "AWS Cognito User ID"
// @Success      200         {array}   models.Group
// @Failure      400         {object}  map[string]string
// @Router       /groups [get]
func (ctrl *Controller) ListMine(c *gin.Context) {
	principal, ok := caller(c)
	if !ok {
		return
	}

	groups, err := ctrl.service.ListMine(principal)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetGroup godoc
// @Summary      Get a group
// @Tags         groups
// @Produce      json
// @Param        id          path      string  true  "Group UUID" format(uuid)
// @Param        cognito_id  query     string  true  "AWS Cognito User ID"
// @Success      200         {object}  models.Group
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /groups/{id} [get]
func (ctrl *Controller) GetGroup(c *gin.Context) {
	id, ok := groupID(c)
	if !ok {
		return
	}

	principal, ok := caller(c)
	if !ok {
		return
	}

	group, err := ctrl.service.Get(principal, id)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup godoc
// @Summary      Delete a group and every secret it owned
// @Tags         groups
// @Param        id          path      string  true  "Group UUID" format(uuid)
// @Param        cognito_id  query     string  true  "AWS Cognito User ID"
// @Success      204         "No Content"
// @Failure      400         {object}  map[string]string
// @Failure      404         {object}  map[string]string
// @Router       /groups/{id} [delete]
func (ctrl *Controller) DeleteGroup(c *gin.Context) {
	id, ok := groupID(c)
	if !ok {
		return
	}

	principal, ok := caller(c)
	if !ok {
		return
	}

	if err := ctrl.service.Delete(principal, id); err != nil {
		fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AddMember godoc
// @Summary      Add a member to a group
// @Description  Owner only.
// @Tags         groups
// @Accept       json
// @Param        id       path      string         true  "Group UUID" format(uuid)
// @Param        request  body      MemberRequest  true  "Member payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /groups/{id}/members [post]
func (ctrl *Controller) AddMember(c *gin.Context) {
	ctrl.changeMembers(c, ctrl.service.AddMember)
}

// RemoveMember godoc
// @Summary      Remove a member from a group
// @Description  Owner only. The group's secrets stay with the group.
// @Tags         groups
// @Accept       json
// @Param        id       path      string         true  "Group UUID" format(uuid)
// @Param        request  body      MemberRequest  true  "Member payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /groups/{id}/members [delete]
func (ctrl *Controller) RemoveMember(c *gin.Context) {
	ctrl.changeMembers(c, ctrl.service.RemoveMember)
}

func (ctrl *Controller) changeMembers(c *gin.Context, change func(models.Principal, uuid.UUID, string) error) {
	id, ok := groupID(c)
	if !ok {
		return
	}

	var req MemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := change(models.Principal{ID: req.CognitoID}, id, req.Member); err != nil {
		fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
