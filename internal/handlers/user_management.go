package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	usersform "github.com/narendhupati/dc-management-tool/components/htmx/admin/users"
	"github.com/narendhupati/dc-management-tool/components/layouts"
	userspage "github.com/narendhupati/dc-management-tool/components/pages/admin/users"
	errorpage "github.com/narendhupati/dc-management-tool/components/pages/error"
	"github.com/narendhupati/dc-management-tool/components/partials"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/components"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ListUsers(c echo.Context) error {
	user := auth.GetCurrentUser(c)
	users, err := database.GetAllUsers()
	if err != nil {
		slog.Error("Failed to load users", slog.String("error", err.Error()))
		return components.Render(c, http.StatusInternalServerError,
			errorpage.ErrorPage(http.StatusInternalServerError, "Failed to load users", err.Error()))
	}

	allProjects, _ := database.GetAccessibleProjects(user)
	flashType, flashMessage := auth.PopFlash(c.Request())
	csrfToken := csrf.Token(c.Request())
	currentPath := c.Request().URL.Path

	rawBreadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "User Management", URL: "/admin/users"},
	)
	breadcrumbs := toBreadcrumbItems(rawBreadcrumbs)
	_ = breadcrumbs // available for future use

	pageContent := userspage.List(user, users, allProjects, flashType, flashMessage, csrfToken)
	sidebar := partials.Sidebar(user, nil, allProjects, currentPath)
	topbar := partials.Topbar(user, nil, allProjects, flashType, flashMessage)

	return components.RenderOK(c,
		layouts.MainWithContent("User Management", sidebar, topbar, flashMessage, flashType, pageContent))
}

func ShowCreateUserForm(c echo.Context) error {
	allProjects, _ := database.GetAllProjects()
	csrfField := string(csrf.TemplateField(c.Request()))

	return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
		IsEdit:      false,
		FormUser:    &models.User{Role: "user", IsActive: true},
		Errors:      map[string]string{},
		AllProjects: allProjects,
		AssignedIDs: nil,
		CsrfField:   csrfField,
	}))
}

func CreateUserHandler(c echo.Context) error {
	allProjects, _ := database.GetAllProjects()
	csrfField := string(csrf.TemplateField(c.Request()))

	newUser := &models.User{
		Username: c.FormValue("username"),
		FullName: c.FormValue("full_name"),
		Email:    c.FormValue("email"),
		Role:     c.FormValue("role"),
		IsActive: true,
	}

	errors := helpers.ValidateStruct(newUser)

	password := c.FormValue("password")
	if password == "" {
		errors["password"] = "Password is required"
	} else if len(password) < 6 {
		errors["password"] = "Password must be at least 6 characters"
	}

	if len(errors) > 0 {
		return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
			IsEdit:      false,
			FormUser:    newUser,
			Errors:      errors,
			AllProjects: allProjects,
			AssignedIDs: nil,
			CsrfField:   csrfField,
		}))
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("Failed to hash password for new user", slog.String("username", newUser.Username), slog.String("error", err.Error()))
		errors["general"] = "Failed to hash password"
		return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
			IsEdit:      false,
			FormUser:    newUser,
			Errors:      errors,
			AllProjects: allProjects,
			AssignedIDs: nil,
			CsrfField:   csrfField,
		}))
	}
	newUser.PasswordHash = hash

	if err := database.CreateUser(newUser); err != nil {
		slog.Error("Failed to create user", slog.String("username", newUser.Username), slog.String("error", err.Error()))
		errors["general"] = "Failed to create user: " + err.Error()
		return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
			IsEdit:      false,
			FormUser:    newUser,
			Errors:      errors,
			AllProjects: allProjects,
			AssignedIDs: nil,
			CsrfField:   csrfField,
		}))
	}

	// Assign projects.
	projectIDs := c.Request().PostForm["project_ids"]
	for _, pidStr := range projectIDs {
		if pid, err := strconv.Atoi(pidStr); err == nil {
			_ = database.AssignUserToProject(newUser.ID, pid)
		}
	}

	auth.SetFlash(c.Request(), "success", "User created successfully")
	c.Response().Header().Set("HX-Redirect", "/admin/users")
	return c.String(http.StatusOK, "")
}

func ShowEditUserForm(c echo.Context) error {
	uid, _ := strconv.Atoi(c.Param("uid"))

	formUser, err := database.GetUserByID(uid)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	allProjects, _ := database.GetAllProjects()
	assignedIDs, _ := database.GetAssignedProjectIDs(uid)
	csrfField := string(csrf.TemplateField(c.Request()))

	return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
		IsEdit:      true,
		FormUser:    formUser,
		Errors:      map[string]string{},
		AllProjects: allProjects,
		AssignedIDs: assignedIDs,
		CsrfField:   csrfField,
	}))
}

func UpdateUserHandler(c echo.Context) error {
	uid, _ := strconv.Atoi(c.Param("uid"))
	allProjects, _ := database.GetAllProjects()
	csrfField := string(csrf.TemplateField(c.Request()))

	formUser, err := database.GetUserByID(uid)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	formUser.FullName = strings.TrimSpace(c.FormValue("full_name"))
	formUser.Email = strings.TrimSpace(c.FormValue("email"))
	formUser.Role = c.FormValue("role")

	errors := helpers.ValidateStruct(formUser)
	if len(errors) > 0 {
		assignedIDs, _ := database.GetAssignedProjectIDs(uid)
		return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
			IsEdit:      true,
			FormUser:    formUser,
			Errors:      errors,
			AllProjects: allProjects,
			AssignedIDs: assignedIDs,
			CsrfField:   csrfField,
		}))
	}

	if err := database.UpdateUser(formUser); err != nil {
		slog.Error("Failed to update user", slog.Int("user_id", uid), slog.String("error", err.Error()))
		errors["general"] = "Failed to update user"
		assignedIDs, _ := database.GetAssignedProjectIDs(uid)
		return components.RenderOK(c, usersform.UserForm(usersform.UserFormProps{
			IsEdit:      true,
			FormUser:    formUser,
			Errors:      errors,
			AllProjects: allProjects,
			AssignedIDs: assignedIDs,
			CsrfField:   csrfField,
		}))
	}

	// Update project assignments: remove all existing, then re-assign.
	projectIDs := c.Request().PostForm["project_ids"]
	existingIDs, _ := database.GetAssignedProjectIDs(uid)
	for _, eid := range existingIDs {
		_ = database.RemoveUserFromProject(uid, eid)
	}
	for _, pidStr := range projectIDs {
		if pid, err := strconv.Atoi(pidStr); err == nil {
			_ = database.AssignUserToProject(uid, pid)
		}
	}

	auth.SetFlash(c.Request(), "success", "User updated successfully")
	c.Response().Header().Set("HX-Redirect", "/admin/users")
	return c.String(http.StatusOK, "")
}

func ToggleUserStatusHandler(c echo.Context) error {
	uid, _ := strconv.Atoi(c.Param("uid"))

	targetUser, err := database.GetUserByID(uid)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	if targetUser.IsActive {
		_ = database.DeactivateUser(uid)
		slog.Info("User deactivated", slog.Int("target_user_id", uid))
		auth.SetFlash(c.Request(), "success", "User deactivated")
	} else {
		_ = database.ActivateUser(uid)
		slog.Info("User activated", slog.Int("target_user_id", uid))
		auth.SetFlash(c.Request(), "success", "User activated")
	}

	c.Response().Header().Set("HX-Redirect", "/admin/users")
	return c.String(http.StatusOK, "")
}

func ResetUserPasswordHandler(c echo.Context) error {
	uid, _ := strconv.Atoi(c.Param("uid"))

	_, err := database.GetUserByID(uid)
	if err != nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	password := c.FormValue("password")
	if password == "" || len(password) < 6 {
		auth.SetFlash(c.Request(), "error", "Password must be at least 6 characters")
		c.Response().Header().Set("HX-Redirect", "/admin/users")
		return c.String(http.StatusOK, "")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("Failed to hash password for reset", slog.Int("target_user_id", uid), slog.String("error", err.Error()))
		auth.SetFlash(c.Request(), "error", "Failed to hash password")
		c.Response().Header().Set("HX-Redirect", "/admin/users")
		return c.String(http.StatusOK, "")
	}

	_ = database.UpdateUserPassword(uid, hash)
	slog.Info("User password reset", slog.Int("target_user_id", uid))
	auth.SetFlash(c.Request(), "success", "Password reset successfully")
	c.Response().Header().Set("HX-Redirect", "/admin/users")
	return c.String(http.StatusOK, "")
}
