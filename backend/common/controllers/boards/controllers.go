package boards

import (
	"net/http"

	srv "github.com/Secreto31126/tesis/common/services/boards"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service *srv.BoardService
}

func NewController(service *srv.BoardService) *Controller {
	return &Controller{
		service: service,
	}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	boardGroup := router.Group("/boards")
	{
		boardGroup.GET("", ctrl.GetUserBoards)
		boardGroup.GET("/:id", ctrl.GetBoard)
		boardGroup.POST("", ctrl.CreateBoard)
		boardGroup.DELETE("/:id", ctrl.DeleteBoard)

		boardGroup.GET("/:id/post-its", ctrl.GetBoardPostIts)

		boardGroup.POST("/:id/collaborators", ctrl.AddCollaborator)
		boardGroup.DELETE("/:id/collaborators", ctrl.RemoveCollaborator)

		boardGroup.POST("/:id/strands", ctrl.ConnectPostIts)
		boardGroup.DELETE("/:id/strands", ctrl.DisconnectPostIts)

		boardGroup.PATCH("/:id/name", ctrl.UpdateBoardName)
	}
}

// GetUserBoards godoc
// @Summary      Get all boards for a user
// @Description  Retrieves a list of all boards associated with a specific Cognito user ID.
// @Tags         boards
// @Produce      json
// @Param        cognito_id  path      string  true  "AWS Cognito User ID"
// @Success      200         {array}   models.Board
// @Failure      500         {object}  map[string]string{"error": "string"}
// @Router       /boards/user/{cognito_id} [get]
func (ctrl *Controller) GetUserBoards(c *gin.Context) {
	cognitoID, exists := c.GetQuery("cognito_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Missing cognito_id",
		})
		return
	}

	boards, err := ctrl.service.GetUserBoards(cognitoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, boards)
}

// GetBoard godoc
// @Summary      Get a specific board
// @Description  Retrieves the details of a single board by its UUID.
// @Tags         boards
// @Produce      json
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      200  {object}  models.Board
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id} [get]
func (ctrl *Controller) GetBoard(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	board, err := ctrl.service.GetBoard(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, board)
}

// CreateBoard godoc
// @Summary      Create a new board
// @Description  Creates a new board with the specified name and owner.
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        request  body      CreateBoardRequest  true  "Board creation payload"
// @Success      201      {object}  models.Board
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards [post]
func (ctrl *Controller) CreateBoard(c *gin.Context) {
	var req CreateBoardRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	board, err := ctrl.service.CreateBoard(req.Name, req.Owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, board)
}

// DeleteBoard godoc
// @Summary      Delete a board
// @Description  Permanently deletes a board and its contents by UUID.
// @Tags         boards
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id} [delete]
func (ctrl *Controller) DeleteBoard(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	err = ctrl.service.DeleteBoard(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetBoardPostIts godoc
// @Summary      Get post-its for a board
// @Description  Retrieves all post-it notes associated with a specific board UUID.
// @Tags         boards, post-its
// @Produce      json
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      200  {array}   models.PostIt
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      500  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/post-its [get]
func (ctrl *Controller) GetBoardPostIts(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	postIts, err := ctrl.service.GetBoardPostIts(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, postIts)
}

// AddCollaborator godoc
// @Summary      Add a collaborator
// @Description  Adds a new user as a collaborator to an existing board.
// @Tags         boards, collaborators
// @Accept       json
// @Param        id       path      string               true  "Board UUID" format(uuid)
// @Param        request  body      CollaboratorRequest  true  "Collaborator payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/collaborators [post]
func (ctrl *Controller) AddCollaborator(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req CollaboratorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.AddCollaboratorToBoard(id, req.CognitoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveCollaborator godoc
// @Summary      Remove a collaborator
// @Description  Removes a user's collaboration access from a board.
// @Tags         boards, collaborators
// @Accept       json
// @Param        id       path      string               true  "Board UUID" format(uuid)
// @Param        request  body      CollaboratorRequest  true  "Collaborator payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/collaborators [delete]
func (ctrl *Controller) RemoveCollaborator(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req CollaboratorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.RemoveCollaboratorFromBoard(id, req.CognitoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateBoardName godoc
// @Summary      Update board name
// @Description  Changes the display name of a specific board.
// @Tags         boards
// @Accept       json
// @Param        id       path      string                  true  "Board UUID" format(uuid)
// @Param        request  body      UpdateBoardNameRequest  true  "Board name update payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/name [put]
func (ctrl *Controller) UpdateBoardName(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req UpdateBoardNameRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.UpdateBoardName(id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ConnectPostIts godoc
// @Summary      Connect two post-its
// @Description  Creates a strand (connection) between a source and target post-it on a board.
// @Tags         boards, strands
// @Accept       json
// @Param        id       path      string         true  "Board UUID" format(uuid)
// @Param        request  body      StrandRequest  true  "Strand payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/strands [post]
func (ctrl *Controller) ConnectPostIts(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req StrandRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.ConnectPostIts(id, req.Source, req.Target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// DisconnectPostIts godoc
// @Summary      Disconnect two post-its
// @Description  Removes the strand (connection) between a source and target post-it on a board.
// @Tags         boards, strands
// @Accept       json
// @Param        id       path      string         true  "Board UUID" format(uuid)
// @Param        request  body      StrandRequest  true  "Strand payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      500      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/strands [delete]
func (ctrl *Controller) DisconnectPostIts(c *gin.Context) {
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid uuid",
		})
		return
	}

	var req StrandRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = ctrl.service.DisconnectPostIts(id, req.Source, req.Target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
