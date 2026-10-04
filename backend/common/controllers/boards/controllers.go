package boards

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	b_srv "github.com/Secreto31126/tesis/common/services/boards"
	r_srv "github.com/Secreto31126/tesis/common/services/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service  *b_srv.BoardService
	realtime *r_srv.RealTimeService
	logger   *slog.Logger
}

func NewController(boards *b_srv.BoardService, realtime *r_srv.RealTimeService) *Controller {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return &Controller{boards, realtime, logger}
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
		boardGroup.DELETE("/:id/strands/:strand", ctrl.DisconnectPostIts)

		boardGroup.PATCH("/:id/name", ctrl.UpdateBoardName)

		boardGroup.PUT("/:id/online", ctrl.ConnectClient)
		boardGroup.DELETE("/:id/online", ctrl.DisconnectClient)
	}
}

// A board the caller may not touch answers 404: whether it exists is not
// disclosed, the same rule groups and the vault follow.
func (ctrl *Controller) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, b_srv.ErrForbidden):
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
	case errors.Is(err, b_srv.ErrOwnerIsNotACollaborator):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		ctrl.logger.Error("board request failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func uuidParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
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

// GetUserBoards godoc
// @Summary      List the caller's boards
// @Description  Every board the caller owns or collaborates on.
// @Tags         boards
// @Produce      json
// @Success      200  {array}   models.Board
// @Failure      401  {object}  map[string]string{"error": "string"}
// @Router       /boards [get]
func (ctrl *Controller) GetUserBoards(c *gin.Context) {
	boards, err := ctrl.service.GetUserBoards(middleware.Principal(c))
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, boards)
}

// GetBoard godoc
// @Summary      Get a specific board
// @Description  Members only.
// @Tags         boards
// @Produce      json
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      200  {object}  models.Board
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      404  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id} [get]
func (ctrl *Controller) GetBoard(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	board, err := ctrl.service.GetBoard(middleware.Principal(c), id)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, board)
}

// CreateBoard godoc
// @Summary      Create a new board
// @Description  The caller becomes its owner.
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        request  body      CreateBoardRequest  true  "Board creation payload"
// @Success      201      {object}  models.Board
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Router       /boards [post]
func (ctrl *Controller) CreateBoard(c *gin.Context) {
	var req CreateBoardRequest
	if !bind(c, &req) {
		return
	}

	board, err := ctrl.service.CreateBoard(middleware.Principal(c), req.Name)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, board)
}

// DeleteBoard godoc
// @Summary      Delete a board
// @Description  Owner only. Permanently deletes a board and its contents.
// @Tags         boards
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      404  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id} [delete]
func (ctrl *Controller) DeleteBoard(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	if err := ctrl.service.DeleteBoard(middleware.Principal(c), id); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetBoardPostIts godoc
// @Summary      Get post-its for a board
// @Description  Members only.
// @Tags         boards, post-its
// @Produce      json
// @Param        id   path      string  true  "Board UUID" format(uuid)
// @Success      200  {array}   models.PostIt
// @Failure      400  {object}  map[string]string{"error": "string"}
// @Failure      404  {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/post-its [get]
func (ctrl *Controller) GetBoardPostIts(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	postIts, err := ctrl.service.GetBoardPostIts(middleware.Principal(c), id)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, postIts)
}

// AddCollaborator godoc
// @Summary      Add a collaborator
// @Description  Owner only.
// @Tags         boards, collaborators
// @Accept       json
// @Param        id       path      string               true  "Board UUID" format(uuid)
// @Param        request  body      CollaboratorRequest  true  "The user to add"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/collaborators [post]
func (ctrl *Controller) AddCollaborator(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req CollaboratorRequest
	if !bind(c, &req) {
		return
	}

	if err := ctrl.service.AddCollaboratorToBoard(middleware.Principal(c), id, req.User); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveCollaborator godoc
// @Summary      Remove a collaborator
// @Description  Owner only, or the collaborator themselves leaving the board.
// @Tags         boards, collaborators
// @Accept       json
// @Param        id       path      string               true  "Board UUID" format(uuid)
// @Param        request  body      CollaboratorRequest  true  "The user to remove"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/collaborators [delete]
func (ctrl *Controller) RemoveCollaborator(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req CollaboratorRequest
	if !bind(c, &req) {
		return
	}

	if err := ctrl.service.RemoveCollaboratorFromBoard(middleware.Principal(c), id, req.User); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateBoardName godoc
// @Summary      Update board name
// @Description  Owner only.
// @Tags         boards
// @Accept       json
// @Param        id       path      string                  true  "Board UUID" format(uuid)
// @Param        request  body      UpdateBoardNameRequest  true  "Board name update payload"
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/name [patch]
func (ctrl *Controller) UpdateBoardName(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req UpdateBoardNameRequest
	if !bind(c, &req) {
		return
	}

	if err := ctrl.service.UpdateBoardName(middleware.Principal(c), id, req.Name); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ConnectPostIts godoc
// @Summary      Connect two post-its
// @Description  Members only. Creates a strand between a source and a target post-it.
// @Tags         boards, strands
// @Accept       json
// @Param        id       path      string         true  "Board UUID" format(uuid)
// @Param        request  body      StrandRequest  true  "Strand payload"
// @Success      201      {object}  models.Strand
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/strands [post]
func (ctrl *Controller) ConnectPostIts(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req StrandRequest
	if !bind(c, &req) {
		return
	}

	strand, err := ctrl.service.ConnectPostIts(middleware.Principal(c), id, req.Source, req.Target)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, strand)
}

// DisconnectPostIts godoc
// @Summary      Disconnect two post-its
// @Description  Members only. Removes a strand.
// @Tags         boards, strands
// @Param        id       path      string         true  "Board UUID"  format(uuid)
// @Param        strand   path      string         true  "Strand UUID" format(uuid)
// @Success      204      "No Content"
// @Failure      400      {object}  map[string]string{"error": "string"}
// @Failure      404      {object}  map[string]string{"error": "string"}
// @Router       /boards/{id}/strands/{strand} [delete]
func (ctrl *Controller) DisconnectPostIts(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	strand, ok := uuidParam(c, "strand")
	if !ok {
		return
	}

	if err := ctrl.service.DisconnectPostIts(middleware.Principal(c), id, strand); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func peer(c *gin.Context) (uuid.UUID, bool) {
	client, err := uuid.Parse(c.Query("peer"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return uuid.Nil, false
	}
	return client, true
}

func (ctrl *Controller) ConnectClient(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	client, ok := peer(c)
	if !ok {
		return
	}

	list, err := ctrl.realtime.AddClientOnline(middleware.Principal(c), id, client)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, list)
}

func (ctrl *Controller) DisconnectClient(c *gin.Context) {
	id, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	client, ok := peer(c)
	if !ok {
		return
	}

	if err := ctrl.realtime.RemoveClientOnline(middleware.Principal(c), id, client); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
