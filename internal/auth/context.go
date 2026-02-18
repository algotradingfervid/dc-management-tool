package auth

import (
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

const contextUserKey = "current_user"

func GetCurrentUser(c echo.Context) *models.User {
	user := c.Get(contextUserKey)
	if user != nil {
		if u, ok := user.(*models.User); ok {
			return u
		}
	}
	return nil
}

func SetCurrentUser(c echo.Context, user *models.User) {
	c.Set(contextUserKey, user)
}
