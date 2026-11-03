package wellknowns

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/wellknowns"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *srv.Service
}

func NewController(service *srv.Service) *Controller {
	return &Controller{service}
}

// RegisterRoutes: any signed-in user reads the catalogue, only platform
// admins change it (the service checks again).
func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	group := router.Group("/well-knowns")
	{
		group.GET("", ctrl.List)
		group.GET("/:key", ctrl.Get)

		admin := group.Group("", middleware.RequireAdmin())
		admin.PUT("/:key", ctrl.Put)
		admin.DELETE("/:key", ctrl.Delete)
		admin.POST("/:key/restore", ctrl.Restore)
	}
}

// List godoc
// @Summary      List the well-known cards
// @Description  Every live definition, by order. Platform credentials are only shown to admins.
// @Tags         well-knowns
// @Success      200  {array}  models.WellKnown
// @Router       /well-knowns [get]
func (ctrl *Controller) List(c *gin.Context) {
	wks, err := ctrl.service.List(middleware.Principal(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, wks)
}

// Get godoc
// @Summary      Get a well-known card
// @Tags         well-knowns
// @Param        key  path  string  true  "Well-known key"
// @Success      200  {object}  models.WellKnown
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /well-knowns/{key} [get]
func (ctrl *Controller) Get(c *gin.Context) {
	wk, err := ctrl.service.Get(middleware.Principal(c), c.Param("key"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, wk)
}

// Put godoc
// @Summary      Create or replace a well-known card (admins)
// @Description  The key comes from the path. Revives a deleted one. Cards already made keep their own copy.
// @Tags         well-knowns
// @Param        key   path  string            true  "Well-known key"
// @Param        body  body  models.WellKnown  true  "Definition"
// @Success      200  {object}  models.WellKnown
// @Failure      400  {object}  map[string]string{"error": "invalid_well_known", "message": "string"}
// @Failure      403  {object}  map[string]string{"error": "forbidden"}
// @Router       /well-knowns/{key} [put]
func (ctrl *Controller) Put(c *gin.Context) {
	var body models.WellKnown
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalid.Error(), "message": err.Error()})
		return
	}

	wk, err := ctrl.service.Put(middleware.Principal(c), c.Param("key"), body)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, wk)
}

// Delete godoc
// @Summary      Delete a well-known card (admins)
// @Description  Hidden from the picker and new cards; existing cards keep running. Undo with restore.
// @Tags         well-knowns
// @Param        key  path  string  true  "Well-known key"
// @Success      204  "No Content"
// @Failure      403  {object}  map[string]string{"error": "forbidden"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /well-knowns/{key} [delete]
func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(middleware.Principal(c), c.Param("key")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Restore godoc
// @Summary      Restore a deleted well-known card (admins)
// @Tags         well-knowns
// @Param        key  path  string  true  "Well-known key"
// @Success      200  {object}  models.WellKnown
// @Failure      403  {object}  map[string]string{"error": "forbidden"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Failure      409  {object}  map[string]string{"error": "not_deleted"}
// @Router       /well-knowns/{key}/restore [post]
func (ctrl *Controller) Restore(c *gin.Context) {
	wk, err := ctrl.service.Restore(middleware.Principal(c), c.Param("key"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, wk)
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrInvalid):
		message, _ := strings.CutPrefix(err.Error(), srv.ErrInvalid.Error()+": ")
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalid.Error(), "message": message})
	case errors.Is(err, srv.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, srv.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": srv.ErrNotFound.Error()})
	case errors.Is(err, srv.ErrNotDeleted):
		c.JSON(http.StatusConflict, gin.H{"error": srv.ErrNotDeleted.Error()})
	default:
		log.Printf("well-knowns: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
