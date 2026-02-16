package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

const contextUserKey = "current_user"

func GetCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get(contextUserKey); exists {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

func SetCurrentUser(c *gin.Context, user *models.User) {
	c.Set(contextUserKey, user)
}
