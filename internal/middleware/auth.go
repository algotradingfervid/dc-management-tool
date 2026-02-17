package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.GetUserID(c.Request)
		if userID == 0 {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := database.GetUserByID(userID)
		if err != nil {
			log.Printf("Failed to load user from session: %v", err)
			auth.DestroySession(c.Request)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Check if user is deactivated
		if !user.IsActive {
			auth.DestroySession(c.Request)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		auth.SetCurrentUser(c, user)
		c.Next()
	}
}
