package middleware

import (
	"net/http"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/gin-gonic/gin"
)

const principalKey = "principal"

// RequireAuth turns a bearer token into the request's principal. Everything
// behind it can trust Principal(c) to be a real, non-anonymous user.
func RequireAuth(verifier infrastructure.TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
		token = strings.TrimSpace(token)
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			unauthorized(c)
			return
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			unauthorized(c)
			return
		}

		c.Set(principalKey, models.Principal{ID: claims.Subject, Admin: claims.Admin})
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Principal(c).Admin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// SameOrigin guards the cookie-authenticated routes: browsers label every
// cross-site request, so a labelled one is refused before it does anything.
func SameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.GetHeader("Sec-Fetch-Site") {
		case "", "same-origin", "none":
			c.Next()
		default:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		}
	}
}

func Principal(c *gin.Context) models.Principal {
	principal, _ := c.Get(principalKey)
	if p, ok := principal.(models.Principal); ok {
		return p
	}
	return models.Principal{}
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
