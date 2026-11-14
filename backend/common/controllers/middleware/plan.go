package middleware

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/gin-gonic/gin"
)

// PlanLimit answers a request the payer's plan refuses: 402 with the limit
// that was hit, its ceiling and the plan, so a client can say what to do
// about it. It reports whether err was one; every controller asks it first.
func PlanLimit(c *gin.Context, err error) bool {
	var limit *models.LimitError
	if !errors.As(err, &limit) {
		return false
	}

	c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
		"error": limit.Error(),
		"limit": limit.Limit,
		"max":   limit.Max,
		"plan":  limit.Plan,
	})
	return true
}
