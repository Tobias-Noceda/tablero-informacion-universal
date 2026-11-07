package customwellknowns

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/customwellknowns"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.Service
}

func NewController(service *srv.Service) *Controller {
	return &Controller{service}
}

type Request struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Title       models.BoardPostItTitle           `json:"title"`
	Resource    string                            `json:"resource"`
	Request     models.Request                    `json:"request"`
	Response    string                            `json:"response"`
	Query       map[string]string                 `json:"query"`
	Rate        int                               `json:"rate"`
	Parameters  []models.CustomWellKnownParameter `json:"parameters"`
	Outputs     []models.CustomWellKnownOutput    `json:"outputs"`
}

func (r Request) definition() models.CustomWellKnown {
	return models.CustomWellKnown{
		Name:        strings.TrimSpace(r.Name),
		Description: strings.TrimSpace(r.Description),
		Title:       r.Title,
		Resource:    strings.TrimSpace(r.Resource),
		Request:     r.Request,
		Response:    r.Response,
		Query:       r.Query,
		Rate:        r.Rate,
		Parameters:  r.Parameters,
		Outputs:     r.Outputs,
	}
}

type scoping func(c *gin.Context) (models.CustomWellKnownScope, bool)

func userScope(c *gin.Context) (models.CustomWellKnownScope, bool) {
	return models.CustomWellKnownScope{Kind: models.CustomScopeUser, Owner: c.Param("id")}, true
}

func boardScope(c *gin.Context) (models.CustomWellKnownScope, bool) {
	return uuidScope(c, models.CustomScopeBoard)
}

func orgScope(c *gin.Context) (models.CustomWellKnownScope, bool) {
	return uuidScope(c, models.CustomScopeOrg)
}

func uuidScope(c *gin.Context, kind models.CustomWellKnownScopeKind) (models.CustomWellKnownScope, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return models.CustomWellKnownScope{}, false
	}
	return models.CustomWellKnownScope{Kind: kind, Owner: id.String()}, true
}

// RegisterRoutes: card types are listed and created under the scope that
// owns them, then read, replaced and deleted by key.
func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	for prefix, s := range map[string]scoping{"/users/:id": userScope, "/boards/:id": boardScope, "/orgs/:id": orgScope} {
		router.GET(prefix+"/well-knowns", ctrl.List(s))
		router.POST(prefix+"/well-knowns", ctrl.Create(s))
	}
	router.GET("/boards/:id/well-knowns/usable", ctrl.ForBoard)

	group := router.Group("/custom-well-knowns")
	{
		group.GET("/:key", ctrl.Get)
		group.PUT("/:key", ctrl.Update)
		group.DELETE("/:key", ctrl.Delete)
	}
}

// List godoc
// @Summary      List a scope's custom card types
// @Tags         custom-well-knowns
// @Param        id  path  string  true  "User id, board UUID or org UUID"
// @Success      200  {array}  models.CustomWellKnown
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /users/{id}/well-knowns [get]
// @Router       /boards/{id}/well-knowns [get]
// @Router       /orgs/{id}/well-knowns [get]
func (ctrl *Controller) List(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s(c)
		if !ok {
			return
		}
		list, err := ctrl.service.List(middleware.Principal(c), scope)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

// Create godoc
// @Summary      Create a custom card type in a scope
// @Description  User scope: only that user. Board scope: board editors. Org scope: org admins.
// @Tags         custom-well-knowns
// @Param        id    path  string   true  "User id, board UUID or org UUID"
// @Param        body  body  Request  true  "Definition"
// @Success      201  {object}  models.CustomWellKnown
// @Failure      400  {object}  map[string]string{"error": "invalid_well_known", "message": "string"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Failure      409  {object}  map[string]string{"error": "limit_reached"}
// @Router       /users/{id}/well-knowns [post]
// @Router       /boards/{id}/well-knowns [post]
// @Router       /orgs/{id}/well-knowns [post]
func (ctrl *Controller) Create(s scoping) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := s(c)
		if !ok {
			return
		}
		body, ok := bind(c)
		if !ok {
			return
		}
		created, err := ctrl.service.Create(middleware.Principal(c), scope, body)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusCreated, created)
	}
}

// ForBoard godoc
// @Summary      The custom card types a board needs
// @Description  Those the caller may drop on it (usable: theirs, the board's, its org's) and those its cards were made from (usable: false).
// @Tags         custom-well-knowns
// @Param        id  path  string  true  "Board UUID"
// @Success      200  {array}  srv.BoardWellKnown
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /boards/{id}/well-knowns/usable [get]
func (ctrl *Controller) ForBoard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}
	list, err := ctrl.service.ForBoard(middleware.Principal(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

// Get godoc
// @Summary      Get a custom card type
// @Tags         custom-well-knowns
// @Param        key  path  string  true  "Key (custom_…)"
// @Success      200  {object}  models.CustomWellKnown
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /custom-well-knowns/{key} [get]
func (ctrl *Controller) Get(c *gin.Context) {
	def, err := ctrl.service.Get(middleware.Principal(c), c.Param("key"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, def)
}

// Update godoc
// @Summary      Replace a custom card type
// @Description  Its key and scope stay. Cards already made keep their own copy.
// @Tags         custom-well-knowns
// @Param        key   path  string   true  "Key (custom_…)"
// @Param        body  body  Request  true  "Definition"
// @Success      200  {object}  models.CustomWellKnown
// @Failure      400  {object}  map[string]string{"error": "invalid_well_known", "message": "string"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /custom-well-knowns/{key} [put]
func (ctrl *Controller) Update(c *gin.Context) {
	body, ok := bind(c)
	if !ok {
		return
	}
	def, err := ctrl.service.Update(middleware.Principal(c), c.Param("key"), body)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, def)
}

// Delete godoc
// @Summary      Delete a custom card type
// @Description  For good. Cards made from it keep running but can no longer be edited.
// @Tags         custom-well-knowns
// @Param        key  path  string  true  "Key (custom_…)"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /custom-well-knowns/{key} [delete]
func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(middleware.Principal(c), c.Param("key")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bind(c *gin.Context) (models.CustomWellKnown, bool) {
	var body Request
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalid.Error(), "message": err.Error()})
		return models.CustomWellKnown{}, false
	}
	return body.definition(), true
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrInvalid):
		message, _ := strings.CutPrefix(err.Error(), srv.ErrInvalid.Error()+": ")
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalid.Error(), "message": message})
	case errors.Is(err, srv.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": srv.ErrNotFound.Error()})
	case errors.Is(err, srv.ErrLimit):
		c.JSON(http.StatusConflict, gin.H{"error": srv.ErrLimit.Error()})
	default:
		log.Printf("custom well-knowns: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
