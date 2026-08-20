package realtime

import (
	"net/http"

	srv "github.com/Secreto31126/tesis/common/services/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Controller struct {
	service *srv.RealtimeService
}

func NewController(service *srv.RealtimeService) *Controller {
	return &Controller{service}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	boardGroup := router.Group("/boards")
	{
		boardGroup.GET("/:id/ws", ctrl.GetPostIt)
	}
}

var upgrader = websocket.Upgrader{
	// Allow all origins for development; restrict this in production.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctrl.service.AddClient(id, conn)
	defer ctrl.service.RemoveClient(id, conn)
	defer conn.Close()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
