package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

func ListUsers(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	users, err := database.GetAllUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load users", "status": 500, "user": user,
		})
		return
	}

	flashType, flashMessage := auth.PopFlash(c.Request)

	c.HTML(http.StatusOK, "admin/users/list.html", gin.H{
		"user":         user,
		"users":        users,
		"currentPath":  c.Request.URL.Path,
		"breadcrumbs":  helpers.BuildBreadcrumbs(helpers.Breadcrumb{Title: "User Management", URL: "/admin/users"}),
		"flashType":    flashType,
		"flashMessage": flashMessage,
		"csrfToken":    csrf.Token(c.Request),
		"csrfField":    csrf.TemplateField(c.Request),
	})
}

func ShowCreateUserForm(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	allProjects, _ := database.GetAllProjects()

	c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
		"user":        user,
		"formUser":    &models.User{Role: "user", IsActive: true},
		"allProjects": allProjects,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func CreateUserHandler(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	allProjects, _ := database.GetAllProjects()

	newUser := &models.User{
		Username: c.PostForm("username"),
		FullName: c.PostForm("full_name"),
		Email:    c.PostForm("email"),
		Role:     c.PostForm("role"),
		IsActive: true,
	}

	errors := newUser.ValidateUser()

	password := c.PostForm("password")
	if password == "" {
		errors["password"] = "Password is required"
	} else if len(password) < 6 {
		errors["password"] = "Password must be at least 6 characters"
	}

	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
			"user": user, "formUser": newUser, "errors": errors,
			"allProjects": allProjects, "csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		errors["general"] = "Failed to hash password"
		c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
			"user": user, "formUser": newUser, "errors": errors,
			"allProjects": allProjects, "csrfField": csrf.TemplateField(c.Request),
		})
		return
	}
	newUser.PasswordHash = hash

	if err := database.CreateUser(newUser); err != nil {
		errors["general"] = "Failed to create user: " + err.Error()
		c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
			"user": user, "formUser": newUser, "errors": errors,
			"allProjects": allProjects, "csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	// Assign projects
	projectIDs := c.PostFormArray("project_ids")
	for _, pidStr := range projectIDs {
		if pid, err := strconv.Atoi(pidStr); err == nil {
			database.AssignUserToProject(newUser.ID, pid)
		}
	}

	auth.SetFlash(c.Request, "success", "User created successfully")
	c.Header("HX-Redirect", "/admin/users")
	c.String(http.StatusOK, "")
}

func ShowEditUserForm(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	uid, _ := strconv.Atoi(c.Param("uid"))

	formUser, err := database.GetUserByID(uid)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	allProjects, _ := database.GetAllProjects()
	assignedIDs, _ := database.GetAssignedProjectIDs(uid)

	c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
		"user":        user,
		"formUser":    formUser,
		"isEdit":      true,
		"allProjects": allProjects,
		"assignedIDs": assignedIDs,
		"csrfField":   csrf.TemplateField(c.Request),
	})
}

func UpdateUserHandler(c *gin.Context) {
	user := auth.GetCurrentUser(c)
	uid, _ := strconv.Atoi(c.Param("uid"))
	allProjects, _ := database.GetAllProjects()

	formUser, err := database.GetUserByID(uid)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	formUser.FullName = c.PostForm("full_name")
	formUser.Email = c.PostForm("email")
	formUser.Role = c.PostForm("role")

	errors := formUser.ValidateUser()
	if len(errors) > 0 {
		c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
			"user": user, "formUser": formUser, "isEdit": true, "errors": errors,
			"allProjects": allProjects, "csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	if err := database.UpdateUser(formUser); err != nil {
		errors["general"] = "Failed to update user"
		c.HTML(http.StatusOK, "htmx/admin/users/form.html", gin.H{
			"user": user, "formUser": formUser, "isEdit": true, "errors": errors,
			"allProjects": allProjects, "csrfField": csrf.TemplateField(c.Request),
		})
		return
	}

	// Update project assignments
	projectIDs := c.PostFormArray("project_ids")
	// Remove all existing, re-assign
	existingIDs, _ := database.GetAssignedProjectIDs(uid)
	for _, eid := range existingIDs {
		database.RemoveUserFromProject(uid, eid)
	}
	for _, pidStr := range projectIDs {
		if pid, err := strconv.Atoi(pidStr); err == nil {
			database.AssignUserToProject(uid, pid)
		}
	}

	auth.SetFlash(c.Request, "success", "User updated successfully")
	c.Header("HX-Redirect", "/admin/users")
	c.String(http.StatusOK, "")
}

func ToggleUserStatusHandler(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))

	targetUser, err := database.GetUserByID(uid)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	if targetUser.IsActive {
		database.DeactivateUser(uid)
		auth.SetFlash(c.Request, "success", "User deactivated")
	} else {
		database.ActivateUser(uid)
		auth.SetFlash(c.Request, "success", "User activated")
	}

	c.Header("HX-Redirect", "/admin/users")
	c.String(http.StatusOK, "")
}

func ResetUserPasswordHandler(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))

	_, err := database.GetUserByID(uid)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	password := c.PostForm("password")
	if password == "" || len(password) < 6 {
		auth.SetFlash(c.Request, "error", "Password must be at least 6 characters")
		c.Header("HX-Redirect", "/admin/users")
		c.String(http.StatusOK, "")
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		auth.SetFlash(c.Request, "error", "Failed to hash password")
		c.Header("HX-Redirect", "/admin/users")
		c.String(http.StatusOK, "")
		return
	}

	database.UpdateUserPassword(uid, hash)
	auth.SetFlash(c.Request, "success", "Password reset successfully")
	c.Header("HX-Redirect", "/admin/users")
	c.String(http.StatusOK, "")
}
