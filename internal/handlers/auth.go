package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/components/standalone"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func ShowLogin(c echo.Context) error {
	if userID := auth.GetUserID(c.Request()); userID != 0 {
		return redirectAfterLogin(c, userID)
	}

	flashType, flashMsg := auth.PopFlash(c.Request())

	csrfToken := csrf.Token(c.Request())
	errors := map[string]string{}
	if flashType == "error" && flashMsg != "" {
		errors["credentials"] = flashMsg
	}
	return components.RenderOK(c, standalone.Login(csrfToken, errors, ""))
}

func ProcessLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		auth.SetFlash(c.Request(), "error", "Username and password are required")
		return c.Redirect(http.StatusFound, "/login")
	}

	user, err := database.GetUserByUsername(username)
	if err != nil {
		slog.Warn("Login failed: user not found", slog.String("username", username), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Invalid username or password")
		return c.Redirect(http.StatusFound, "/login")
	}

	if !auth.VerifyPassword(user.PasswordHash, password) {
		slog.Warn("Login failed: invalid password", slog.String("username", username))
		auth.SetFlash(c.Request(), "error", "Invalid username or password")
		return c.Redirect(http.StatusFound, "/login")
	}

	if !user.IsActive {
		slog.Warn("Login failed: account deactivated", slog.String("username", username))
		auth.SetFlash(c.Request(), "error", "Your account has been deactivated")
		return c.Redirect(http.StatusFound, "/login")
	}

	if err := auth.RenewToken(c.Request()); err != nil {
		slog.Error("Failed to renew session token", slog.String("error", err.Error()), slog.String("username", username))
	}

	auth.SetUserID(c.Request(), user.ID)
	auth.SetFlash(c.Request(), "success", "Login successful")

	slog.Info("User logged in successfully", slog.String("username", username))
	return redirectAfterLogin(c, user.ID)
}

func RedirectToProject(c echo.Context, userID int) error {
	return redirectAfterLogin(c, userID)
}

func redirectAfterLogin(c echo.Context, userID int) error {
	user, err := database.GetUserByID(userID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/projects/new")
	}

	if user.LastProjectID != nil {
		_, lookupErr := database.GetProjectByID(*user.LastProjectID)
		if lookupErr == nil {
			return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", *user.LastProjectID))
		}
	}

	projects, err := database.GetAccessibleProjects(user)
	if err != nil || len(projects) == 0 {
		if user.IsAdmin() {
			return c.Redirect(http.StatusFound, "/projects/new")
		}
		return c.Redirect(http.StatusFound, "/projects/select")
	}

	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/%d/dashboard", projects[0].ID))
}

func Logout(c echo.Context) error {
	userID := auth.GetUserID(c.Request())

	if err := auth.DestroySession(c.Request()); err != nil {
		slog.Error("Failed to destroy session", slog.String("error", err.Error()), slog.Int("userID", userID))
	}

	slog.Info("User logged out", slog.Int("userID", userID))
	return c.Redirect(http.StatusFound, "/login")
}
