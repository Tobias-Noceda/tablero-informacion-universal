package postits

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/postits"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.PostItsService
}

func NewController(service *srv.PostItsService) *Controller {
	return &Controller{
		service: service,
	}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	boardGroup := router.Group("/post-its")
	{
		boardGroup.POST("", ctrl.CreatePostIt)
		boardGroup.GET("/:id", ctrl.ExecutePostIt)
		boardGroup.DELETE("/:id", ctrl.DeletePostIt)

		boardGroup.GET("/:id/settings", ctrl.GetPostIt)
		boardGroup.PATCH("/:id/settings", ctrl.EditPostIt)
		boardGroup.PATCH("/:id/position", ctrl.MovePostIt)
		boardGroup.PATCH("/:id/title", ctrl.EditTitlePostIt)
		boardGroup.PATCH("/:id/style", ctrl.StylePostIt)
	}
}

// CreatePostIt godoc
// @Summary      Create a post-it
// @Description  Create a new post-it from scratch or following a Well-Known.
// @Tags         post-its
// @Param        body {CreatePostItRequest}  "Post-it data"
// @Success      201  {object}	models.PostIt
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /post-its/ [post]
func (ctrl *Controller) CreatePostIt(c *gin.Context) {
	var req CreatePostItRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	newPostIt := &models.PostIts{
		Board:     req.Board,
		WellKnown: req.WellKnown,
		Params:    req.Params,
		Resource:  req.Resource.URL,
		Request:   req.Request,
		Response:  req.Response,
		Query:     req.Query,
		Rate:      req.Rate,
		Envs:      req.Envs,
		Bindings:  req.Bindings,
	}

	postIt, err := ctrl.service.CreatePostIt(middleware.Principal(c), req.Title, newPostIt)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, postIt)
}

// GetPostIt godoc
// @Summary      Get a post-it settings
// @Description  Retrieve a post-it settings by UUID.
// @Tags         post-its
// @Param        id   path      string  true  "Post-it UUID" format(uuid)
// @Success      200  {object}	models.PostIt
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id}/settings [get]
func (ctrl *Controller) GetPostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	postIt, err := ctrl.service.GetPostIt(middleware.Principal(c), id)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, postIt)
}

// DeletePostIt godoc
// @Summary      Delete a post-it
// @Description  Permanently deletes a post-it and its contents by UUID.
// @Tags         post-its
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      200  {array}   models.Strand The deleted strands
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id} [delete]
func (ctrl *Controller) DeletePostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	strands, err := ctrl.service.DeletePostIt(middleware.Principal(c), id)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, strands)
}

// ExecutePostIt godoc
// @Summary      Execute a post-it to fetch its data
// @Description  Execute a post-it.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string                  true  "Post-it UUID" format(uuid)
// @Success      200      {object}  any
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id} [get]
func (ctrl *Controller) ExecutePostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	postIt, err := ctrl.service.GetPostIt(middleware.Principal(c), id)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	// ?fresh=1 skips the cached result (the card detail's refresh button).
	fresh := c.Query("fresh") == "1"
	execute := ctrl.service.ExecutePostIt
	if fresh {
		execute = ctrl.service.RefreshPostIt
	}

	data, err := execute(postIt)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	if fresh {
		c.Writer.Header().Set("Cache-Control", "no-store")
	} else {
		c.Writer.Header().Set("Cache-Control", "private, max-age=10")
	}
	c.JSON(http.StatusOK, data)
}

// EditPostIt godoc
// @Summary      Update a post-it settings
// @Description  Update the editable settings of a post-it. Only the provided fields are changed.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string               true  "Post-it UUID" format(uuid)
// @Param        request  body      UpdatePostItRequest  true  "Post-it settings update payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id}/settings [patch]
func (ctrl *Controller) EditPostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req UpdatePostItRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	set := map[string]any{}
	if req.Params != nil {
		set["params"] = req.Params
	}
	if req.Bindings != nil {
		set["bindings"] = req.Bindings
	}
	if req.Query != nil {
		set["query"] = req.Query
	}
	if req.Response != nil {
		set["response"] = *req.Response
	}
	if req.Rate != nil {
		set["rate"] = *req.Rate
	}
	if req.Resource != nil {
		set["resource"] = *req.Resource
	}
	if req.Request != nil {
		set["request"] = *req.Request
	}

	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no fields to update",
		})
		return
	}

	err = ctrl.service.UpdatePostIt(middleware.Principal(c), id, set)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// MovePostIt godoc
// @Summary      Move a post-it in the board
// @Description  Changes position of a post-it.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string             true  "Post-It UUID" format(uuid)
// @Param        request  body      MovePostItRequest  true  "Post-It name update payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id}/position [put]
func (ctrl *Controller) MovePostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req MovePostItRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.MovePostIt(middleware.Principal(c), id, models.Position{X: *req.X, Y: *req.Y})
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// EditTitlePostIt godoc
// @Summary      Edit a post-it title
// @Description  Changes the rendered title of a post-it.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string                    true  "Post-It UUID" format(uuid)
// @Param        request  body      UpdatePostItTitleRequest  true  "Post-It name update payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id}/title [patch]
func (ctrl *Controller) EditTitlePostIt(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req UpdatePostItTitleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.EditTitlePostIt(middleware.Principal(c), id, req.Title)
	if err != nil {
		c.JSON(statusOf(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// StylePostIt godoc
// @Summary      Restyle a post-it
// @Description  Editors only. Changes the fields sent (color from the strands' palette, border none|solid|dashed|dotted, width 120-1600 and height 60-1200 pixels); an empty value, or 0 for a size, goes back to the default.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string                 true  "Post-It UUID" format(uuid)
// @Param        request  body      models.CardStylePatch  true  "Fields to change"
// @Success      200      {object}  models.CardStyle
// @Failure      400      {object}  map[string]string{"error": "string"}  "invalid_card_style"
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /post-its/{id}/style [patch]
func (ctrl *Controller) StylePostIt(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var patch models.CardStylePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_card_style"})
		return
	}

	style, err := ctrl.service.StylePostIt(middleware.Principal(c), id, patch)
	switch {
	case errors.Is(err, srv.ErrInvalidCardStyle):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_card_style"})
		return
	case err != nil:
		c.JSON(statusOf(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, style)
}

func statusOf(err error) int {
	switch {
	case errors.Is(err, srv.ErrNotAMember):
		return http.StatusNotFound
	case errors.Is(err, srv.ErrCredentialUnavailable),
		errors.Is(err, infrastructure.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, srv.ErrInvalidBinding),
		errors.Is(err, srv.ErrUnknownWellKnown),
		errors.Is(err, srv.ErrInvalidOneOff),
		errors.Is(err, srv.ErrTemplateFixed),
		errors.Is(err, models.ErrWellKnownMissingVariable),
		errors.Is(err, infrastructure.ErrUnknownCredential):
		return http.StatusBadRequest
	case errors.Is(err, srv.ErrSystemSecretMissing):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
