package auth

import (
	"net/http"
	"strings"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(AuthenticatedRoleKey) != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		c.Next()
	}
}

func RequireOwnerOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticatedUserID := c.GetString(AuthenticatedUserIDKey)
		if authenticatedUserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if c.GetString(AuthenticatedRoleKey) != models.RoleAdmin &&
			!strings.EqualFold(authenticatedUserID, c.Param("id")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		c.Next()
	}
}
