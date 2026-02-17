package middleware

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

func ProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		projectID, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"error":  "Invalid project ID",
				"status": 404,
				"user":   auth.GetCurrentUser(c),
			})
			c.Abort()
			return
		}

		project, err := database.GetProjectByID(projectID)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"error":  "Project not found",
				"status": 404,
				"user":   auth.GetCurrentUser(c),
			})
			c.Abort()
			return
		}

		user := auth.GetCurrentUser(c)
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Admin can access all projects; regular users need assignment or ownership
		if !user.IsAdmin() {
			assigned, err := database.IsUserAssignedToProject(user.ID, project.ID)
			if err != nil || (!assigned && project.CreatedBy != user.ID) {
				c.HTML(http.StatusForbidden, "error.html", gin.H{
					"error":  "You don't have access to this project",
					"status": 403,
					"user":   user,
				})
				c.Abort()
				return
			}
		}

		c.Set("currentProject", project)

		if count, err := database.GetProductCount(project.ID); err == nil {
			c.Set("productCount", count)
		}

		go func() {
			if err := database.UpdateLastProjectID(user.ID, project.ID); err != nil {
				log.Printf("Failed to update last_project_id for user %d: %v", user.ID, err)
			}
		}()

		c.Next()
	}
}
