package postittemplates

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/postittemplates"
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
	Name        string                           `json:"name"`
	Description string                           `json:"description"`
	Title       models.BoardPostItTitle          `json:"title"`
	Resource    string                           `json:"resource"`
	Request     models.Request                   `json:"request"`
	Response    string                           `json:"response"`
	Query       map[string]string                `json:"query"`
	Rate        int                              `json:"rate"`
	Parameters  []models.PostitTemplateParameter `json:"parameters"`
	Outputs     []models.PostitTemplateOutput    `json:"outputs"`
}

func (r Request) definition() models.PostitTemplate {
	return models.PostitTemplate{
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

type scoping func(c *gin.Context) (models.PostitTemplateScope, bool)

func userScope(c *gin.Context) (models.PostitTemplateScope, bool) {
	return models.PostitTemplateScope{Kind: models.TemplateScopeUser, Owner: c.Param("id")}, true
}

func boardScope(c *gin.Context) (models.PostitTemplateScope, bool) {
	return uuidScope(c, models.TemplateScopeBoard)
}

func orgScope(c *gin.Context) (models.PostitTemplateScope, bool) {
	return uuidScope(c, models.TemplateScopeOrg)
}

func uuidScope(c *gin.Context, kind models.PostitTemplateScopeKind) (models.PostitTemplateScope, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return models.PostitTemplateScope{}, false
	}
	return models.PostitTemplateScope{Kind: kind, Owner: id.String()}, true
}

// RegisterRoutes: card types are listed and created under the scope that
// owns them, then read, replaced and deleted by key.
func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	for prefix, s := range map[string]scoping{"/users/:id": userScope, "/boards/:id": boardScope, "/orgs/:id": orgScope} {
		router.GET(prefix+"/postit-templates", ctrl.List(s))
		router.POST(prefix+"/postit-templates", ctrl.Create(s))
	}
	router.GET("/boards/:id/postit-templates/usable", ctrl.ForBoard)

	group := router.Group("/postit-templates")
	{
		group.GET("/:key", ctrl.Get)
		group.PUT("/:key", ctrl.Update)
		group.DELETE("/:key", ctrl.Delete)
	}
}

// List godoc
// @Summary      List a scope's post-it templates
// @Tags         postit-templates
// @Param        id  path  string  true  "User id, board UUID or org UUID"
// @Success      200  {array}  models.PostitTemplate
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /users/{id}/postit-templates [get]
// @Router       /boards/{id}/postit-templates [get]
// @Router       /orgs/{id}/postit-templates [get]
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
// @Summary      Create a post-it template in a scope
// @Description  User scope: only that user. Board scope: board editors. Org scope: org admins.
// @Tags         postit-templates
// @Param        id    path  string   true  "User id, board UUID or org UUID"
// @Param        body  body  Request  true  "Definition"
// @Success      201  {object}  models.PostitTemplate
// @Failure      400  {object}  map[string]string{"error": "invalid_postit_template", "message": "string"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Failure      409  {object}  map[string]string{"error": "limit_reached"}
// @Router       /users/{id}/postit-templates [post]
// @Router       /boards/{id}/postit-templates [post]
// @Router       /orgs/{id}/postit-templates [post]
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
// @Summary      The post-it templates a board needs
// @Description  Those the caller may drop on it (usable: theirs, the board's, its org's) and those its cards were made from (usable: false).
// @Tags         postit-templates
// @Param        id  path  string  true  "Board UUID"
// @Success      200  {array}  srv.BoardPostitTemplate
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /boards/{id}/postit-templates/usable [get]
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
// @Summary      Get a post-it template
// @Tags         postit-templates
// @Param        key  path  string  true  "Key (custom_…)"
// @Success      200  {object}  models.PostitTemplate
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /postit-templates/{key} [get]
func (ctrl *Controller) Get(c *gin.Context) {
	def, err := ctrl.service.Get(middleware.Principal(c), c.Param("key"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, def)
}

// Update godoc
// @Summary      Replace a post-it template
// @Description  Its key and scope stay. Cards already made keep their own copy.
// @Tags         postit-templates
// @Param        key   path  string   true  "Key (custom_…)"
// @Param        body  body  Request  true  "Definition"
// @Success      200  {object}  models.PostitTemplate
// @Failure      400  {object}  map[string]string{"error": "invalid_postit_template", "message": "string"}
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /postit-templates/{key} [put]
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
// @Summary      Delete a post-it template
// @Description  For good. Cards made from it keep running but can no longer be edited.
// @Tags         postit-templates
// @Param        key  path  string  true  "Key (custom_…)"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string{"error": "not_found"}
// @Router       /postit-templates/{key} [delete]
func (ctrl *Controller) Delete(c *gin.Context) {
	if err := ctrl.service.Delete(middleware.Principal(c), c.Param("key")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bind(c *gin.Context) (models.PostitTemplate, bool) {
	var body Request
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": srv.ErrInvalid.Error(), "message": err.Error()})
		return models.PostitTemplate{}, false
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
		log.Printf("post-it templates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
