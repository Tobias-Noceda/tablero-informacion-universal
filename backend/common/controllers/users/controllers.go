package users

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	srv "github.com/Secreto31126/tesis/common/services/users"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Controller struct {
	service  *srv.UserService
	verifier infrastructure.TokenVerifier
}

func NewController(service *srv.UserService, verifier infrastructure.TokenVerifier) *Controller {
	return &Controller{service, verifier}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	users := router.Group("/users", middleware.RequireAuth(ctrl.verifier))
	{
		users.GET("/:id", ctrl.GetUser)
		users.PATCH("/:id", ctrl.UpdateUser)
	}
}

type UpdateUserRequest struct {
	Name string `json:"name" binding:"required"`
}

// GetUser godoc
// @Summary      Read a user
// @Description  One's own record comes back in full; anyone else's as a public summary.
// @Tags         users
// @Param        id  path  string  true  "User id"
// @Success      200
// @Failure      404
// @Router       /users/{id} [get]
func (ctrl *Controller) GetUser(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	principal := middleware.Principal(c)

	if principal.ID == id.String() {
		profile, err := ctrl.service.Profile(principal, id)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusOK, profile)
		return
	}

	summary, err := ctrl.service.Summary(principal, id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// UpdateUser godoc
// @Summary      Rename oneself
// @Tags         users
// @Accept       json
// @Param        id    path  string             true  "User id"
// @Param        body  body  UpdateUserRequest  true  "New name"
// @Success      200
// @Failure      404  "Not one's own id"
// @Router       /users/{id} [patch]
func (ctrl *Controller) UpdateUser(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := ctrl.service.Rename(middleware.Principal(c), id, req.Name)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func userID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return uuid.Nil, false
	}
	return id, true
}

// A user the caller may not touch answers 404, like every other resource.
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrForbidden), errors.Is(err, infrastructure.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	case errors.Is(err, srv.ErrInvalidName):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
