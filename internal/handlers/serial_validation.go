package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

// SerialValidationRequest is the JSON body for the validation endpoint.
type SerialValidationRequest struct {
	ProjectID     int    `json:"project_id"`
	SerialNumbers string `json:"serial_numbers"` // newline-separated
	ExcludeDCID   *int   `json:"exclude_dc_id"`
}

// SerialValidationResponse is the JSON response.
type SerialValidationResponse struct {
	Valid            bool                       `json:"valid"`
	DuplicateInDB    []SerialConflictResponse   `json:"duplicate_in_db"`
	DuplicateInInput []string                   `json:"duplicate_in_input"`
	TotalCount       int                        `json:"total_count"`
}

// SerialConflictResponse is a single conflict entry.
type SerialConflictResponse struct {
	SerialNumber string `json:"serial_number"`
	DCNumber     string `json:"dc_number"`
	DCStatus     string `json:"dc_status"`
	ProductName  string `json:"product_name"`
}

// ValidateSerialNumbers handles POST /api/serial-numbers/validate.
func ValidateSerialNumbers(c *gin.Context) {
	var req SerialValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ProjectID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	// Parse serial numbers
	serials := parseSerialNumbers(req.SerialNumbers)

	// Find duplicates within the input itself
	seen := make(map[string]bool)
	var dupsInInput []string
	dupsSet := make(map[string]bool)
	for _, s := range serials {
		if seen[s] && !dupsSet[s] {
			dupsInInput = append(dupsInInput, s)
			dupsSet[s] = true
		}
		seen[s] = true
	}

	// Deduplicate for DB check
	unique := make([]string, 0, len(seen))
	for s := range seen {
		unique = append(unique, s)
	}

	// Check against database
	conflicts, err := database.CheckSerialsInProject(req.ProjectID, unique, req.ExcludeDCID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate serial numbers"})
		return
	}

	var dbDups []SerialConflictResponse
	for _, conflict := range conflicts {
		dbDups = append(dbDups, SerialConflictResponse{
			SerialNumber: conflict.SerialNumber,
			DCNumber:     conflict.DCNumber,
			DCStatus:     conflict.DCStatus,
			ProductName:  conflict.ProductName,
		})
	}

	resp := SerialValidationResponse{
		Valid:            len(dbDups) == 0 && len(dupsInInput) == 0,
		DuplicateInDB:    dbDups,
		DuplicateInInput: dupsInInput,
		TotalCount:       len(serials),
	}

	c.JSON(http.StatusOK, resp)
}

// IssueDCHandler handles POST /projects/:id/dcs/:dcid/issue.
func IssueDCHandler(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DC ID"})
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
		return
	}

	if dc.Status != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DC is already issued"})
		return
	}

	// Validate DC has line items with serial numbers
	lineItems, err := database.GetLineItemsByDCID(dcID)
	if err != nil || len(lineItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DC must have at least one line item to be issued"})
		return
	}

	for _, li := range lineItems {
		serials, _ := database.GetSerialNumbersByLineItemID(li.ID)
		if len(serials) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All line items must have serial numbers before issuing"})
			return
		}
		if len(serials) != li.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Serial number count must match quantity for all line items"})
			return
		}
	}

	if err := database.IssueDC(dcID, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue DC: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "DC issued successfully",
	})
}

// DeleteDCHandler handles DELETE /projects/:id/dcs/:dcid.
func DeleteDCHandler(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DC ID"})
		return
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		c.JSON(http.StatusNotFound, gin.H{"error": "DC not found"})
		return
	}

	if err := database.DeleteDC(dcID); err != nil {
		if strings.Contains(err.Error(), "failed to delete") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete DC"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "DC and all associated serial numbers deleted successfully",
		"dc_number": dc.DCNumber,
	})
}
