package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func ShowLogin(c *gin.Context) {
	if userID := auth.GetUserID(c.Request); userID != 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	flashType, flashMsg := auth.PopFlash(c.Request)

	c.HTML(http.StatusOK, "login.html", gin.H{
		"csrfField": csrf.TemplateField(c.Request),
		"flashType": flashType,
		"flashMsg":  flashMsg,
	})
}

func ProcessLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		auth.SetFlash(c.Request, "error", "Username and password are required")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	user, err := database.GetUserByUsername(username)
	if err != nil {
		log.Printf("Login failed for username %s: %v", username, err)
		auth.SetFlash(c.Request, "error", "Invalid username or password")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, password) {
		log.Printf("Invalid password for username %s", username)
		auth.SetFlash(c.Request, "error", "Invalid username or password")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if err := auth.RenewToken(c.Request); err != nil {
		log.Printf("Failed to renew session token: %v", err)
	}

	auth.SetUserID(c.Request, user.ID)
	auth.SetFlash(c.Request, "success", "Login successful")

	log.Printf("User %s logged in successfully", username)
	c.Redirect(http.StatusFound, "/")
}

func Logout(c *gin.Context) {
	userID := auth.GetUserID(c.Request)

	if err := auth.DestroySession(c.Request); err != nil {
		log.Printf("Failed to destroy session: %v", err)
	}

	log.Printf("User ID %d logged out", userID)
	c.Redirect(http.StatusFound, "/login")
}
