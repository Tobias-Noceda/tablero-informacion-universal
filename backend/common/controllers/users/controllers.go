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
	service *srv.UserService
}

func NewController(service *srv.UserService) *Controller {
	return &Controller{service}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	users := router.Group("/users")
	{
		users.GET("/:id", ctrl.GetUser)
		users.PATCH("/:id", ctrl.UpdateUser)
	}
}

type UpdateUserRequest struct {
	Name   *string `json:"name"`
	Locale *string `json:"locale"`
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
// @Summary      Change one's name or language
// @Description  Only the fields sent change. The language (en, es) is the one mails arrive in.
// @Tags         users
// @Accept       json
// @Param        id    path  string             true  "User id"
// @Param        body  body  UpdateUserRequest  true  "New name and/or locale"
// @Success      200
// @Failure      400  "invalid name, invalid_locale or no fields to update"
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

	profile, err := ctrl.service.Update(middleware.Principal(c), id, srv.Changes{Name: req.Name, Locale: req.Locale})
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
	case errors.Is(err, srv.ErrInvalidName), errors.Is(err, srv.ErrInvalidLocale), errors.Is(err, srv.ErrNothingToUpdate):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}
