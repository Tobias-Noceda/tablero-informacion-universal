package postits

import (
	"net/http"

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
		boardGroup.POST("/", ctrl.CreatePostIt)
		boardGroup.GET("/:id", ctrl.ExecutePostIt)
		boardGroup.DELETE("/:id", ctrl.DeletePostIt)

		boardGroup.GET("/:id/settings", ctrl.GetPostIt)
		boardGroup.PATCH("/:id/settings", ctrl.EditPostIt)
		boardGroup.PATCH("/:id/position", ctrl.MovePostIt)
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

	postIt, err := ctrl.service.CreatePostIt(&models.PostIts{
		Board:     req.Board,
		WellKnown: req.WellKnown,
		Params:    req.Params,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
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

	postIt, err := ctrl.service.GetPostIt(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
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
// @Success      204  "No Content"
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

	err = ctrl.service.DeletePostIt(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
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

	postIt, err := ctrl.service.GetPostIt(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	data, err := ctrl.service.ExecutePostIt(postIt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Writer.Header().Set("Cache-Control", "private, max-age=10")
	c.JSON(http.StatusOK, data)
}

// EditPostIt godoc
// @Summary      Update a post-it settings in the board
// @Description  Update a post-it.
// @Tags         post-its
// @Accept       json
// @Param        id       path      string                  true  "Board UUID" format(uuid)
// @Param        request  body      UpdateBoardNameRequest  true  "Board name update payload"
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

	var req MovePostItRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.MovePostIt(id, models.Position{X: req.X, Y: req.Y})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
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

	err = ctrl.service.MovePostIt(id, models.Position{X: req.X, Y: req.Y})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
