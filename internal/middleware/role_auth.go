package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
)

// RequireRole checks that the current user has the specified role.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := auth.GetCurrentUser(c)
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		if user.Role != role {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"error":  "You don't have permission to access this page",
				"status": 403,
				"user":   user,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
