package health

import (
	"net/http"

	srv "github.com/Secreto31126/tesis/common/services/health"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	health *srv.Service
}

func NewController(health *srv.Service) *Controller {
	return &Controller{health}
}

// RegisterRoutes mounts the health route. It is public: an uptime monitor
// calls it without a session, and it says nothing about anyone's data.
func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	router.GET("/health", ctrl.Health)
}

// Health godoc
// @Summary      Whether the service and its stores answer
// @Description  Public. 200 while Mongo answers (status "degraded" if Redis does not), 503 without Mongo. version is the deployed commit when the platform provides it.
// @Tags         health
// @Produce      json
// @Success      200  {object}  srv.Report
// @Failure      503  {object}  srv.Report
// @Router       /health [get]
func (ctrl *Controller) Health(c *gin.Context) {
	report := ctrl.health.Check(c.Request.Context())

	code := http.StatusOK
	if report.Status == srv.StatusDown {
		code = http.StatusServiceUnavailable
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(code, report)
}
