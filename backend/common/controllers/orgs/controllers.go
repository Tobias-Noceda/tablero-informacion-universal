package orgs

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	srv "github.com/Secreto31126/tesis/common/services/orgs"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.OrgService
	logger  *slog.Logger
}

func NewController(service *srv.OrgService) *Controller {
	return &Controller{service, slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	orgs := router.Group("/orgs")
	{
		orgs.POST("", ctrl.CreateOrg)
		orgs.GET("", ctrl.ListMine)
		orgs.GET("/:id", ctrl.GetOrg)
		orgs.PATCH("/:id", ctrl.RenameOrg)
		orgs.DELETE("/:id", ctrl.DeleteOrg)
		orgs.PUT("/:id/members", ctrl.SetMember)
		orgs.DELETE("/:id/members/:user", ctrl.RemoveMember)
	}
}

// An organization the caller is not in answers 404: whether it exists is not
// disclosed, the same rule boards and groups follow.
func (ctrl *Controller) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrForbidden):
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
	case errors.Is(err, infrastructure.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
	case errors.Is(err, srv.ErrInvalidName):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_name"})
	case errors.Is(err, srv.ErrInvalidRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_role"})
	case errors.Is(err, srv.ErrLastAdmin):
		c.JSON(http.StatusConflict, gin.H{"error": "last_admin"})
	case errors.Is(err, srv.ErrOrgNotEmpty):
		c.JSON(http.StatusConflict, gin.H{"error": "org_not_empty"})
	case errors.Is(err, srv.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
	default:
		ctrl.logger.Error("organization request failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return uuid.Nil, false
	}
	return id, true
}

func bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

// CreateOrg godoc
// @Summary      Create an organization
// @Description  The caller becomes its first admin.
// @Tags         orgs
// @Accept       json
// @Produce      json
// @Param        request  body      CreateOrgRequest  true  "Organization payload"
// @Success      201      {object}  models.Org
// @Failure      400      {object}  map[string]string
// @Router       /orgs [post]
func (ctrl *Controller) CreateOrg(c *gin.Context) {
	var req CreateOrgRequest
	if !bind(c, &req) {
		return
	}

	org, err := ctrl.service.Create(middleware.Principal(c), req.Name)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, org)
}

// ListMine godoc
// @Summary      List the caller's organizations
// @Description  Each one carries the caller's role.
// @Tags         orgs
// @Produce      json
// @Success      200  {array}   models.Org
// @Router       /orgs [get]
func (ctrl *Controller) ListMine(c *gin.Context) {
	orgs, err := ctrl.service.ListMine(middleware.Principal(c))
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, orgs)
}

// GetOrg godoc
// @Summary      Get an organization and who is in it
// @Description  Members only.
// @Tags         orgs
// @Produce      json
// @Param        id   path      string  true  "Organization UUID" format(uuid)
// @Success      200  {object}  models.OrgDetail
// @Failure      404  {object}  map[string]string
// @Router       /orgs/{id} [get]
func (ctrl *Controller) GetOrg(c *gin.Context) {
	id, ok := orgID(c)
	if !ok {
		return
	}

	org, err := ctrl.service.Get(middleware.Principal(c), id)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, org)
}

// RenameOrg godoc
// @Summary      Rename an organization
// @Description  Admins only.
// @Tags         orgs
// @Accept       json
// @Param        id       path  string            true  "Organization UUID" format(uuid)
// @Param        request  body  RenameOrgRequest  true  "New name"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /orgs/{id} [patch]
func (ctrl *Controller) RenameOrg(c *gin.Context) {
	id, ok := orgID(c)
	if !ok {
		return
	}
	var req RenameOrgRequest
	if !bind(c, &req) {
		return
	}

	if err := ctrl.service.Rename(middleware.Principal(c), id, req.Name); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteOrg godoc
// @Summary      Delete an organization
// @Description  Admins only, once no board or group belongs to it.
// @Tags         orgs
// @Param        id   path  string  true  "Organization UUID" format(uuid)
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "org_not_empty"
// @Router       /orgs/{id} [delete]
func (ctrl *Controller) DeleteOrg(c *gin.Context) {
	id, ok := orgID(c)
	if !ok {
		return
	}

	if err := ctrl.service.Delete(middleware.Principal(c), id); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// SetMember godoc
// @Summary      Add someone to an organization, or change their role
// @Description  Admins only. The last admin cannot be demoted.
// @Tags         orgs
// @Accept       json
// @Produce      json
// @Param        id       path      string            true  "Organization UUID" format(uuid)
// @Param        request  body      SetMemberRequest  true  "Who and as what"
// @Success      200      {object}  models.OrgMemberSummary
// @Failure      400      {object}  map[string]string  "invalid_role"
// @Failure      404      {object}  map[string]string  "user_not_found"
// @Failure      409      {object}  map[string]string  "last_admin"
// @Router       /orgs/{id}/members [put]
func (ctrl *Controller) SetMember(c *gin.Context) {
	id, ok := orgID(c)
	if !ok {
		return
	}
	var req SetMemberRequest
	if !bind(c, &req) {
		return
	}

	member, err := ctrl.service.SetMember(middleware.Principal(c), id, req.Email, req.Role)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, member)
}

// RemoveMember godoc
// @Summary      Remove someone from an organization
// @Description  Admins, or the member themselves. The last admin cannot leave.
// @Tags         orgs
// @Param        id    path  string  true  "Organization UUID" format(uuid)
// @Param        user  path  string  true  "User id"
// @Success      204   "No Content"
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string  "last_admin"
// @Router       /orgs/{id}/members/{user} [delete]
func (ctrl *Controller) RemoveMember(c *gin.Context) {
	id, ok := orgID(c)
	if !ok {
		return
	}

	if err := ctrl.service.RemoveMember(middleware.Principal(c), id, c.Param("user")); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
