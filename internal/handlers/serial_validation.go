package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/database"
)

// SerialValidationRequest is the JSON body for the validation endpoint.
type SerialValidationRequest struct {
	ProjectID     int    `json:"project_id"`
	ProductID     int    `json:"product_id"`
	SerialNumbers string `json:"serial_numbers"` // newline-separated
	ExcludeDCID   *int   `json:"exclude_dc_id"`
}

// SerialValidationResponse is the JSON response.
type SerialValidationResponse struct {
	Valid            bool                     `json:"valid"`
	DuplicateInDB    []SerialConflictResponse `json:"duplicate_in_db"`
	DuplicateInInput []string                 `json:"duplicate_in_input"`
	TotalCount       int                      `json:"total_count"`
}

// SerialConflictResponse is a single conflict entry.
type SerialConflictResponse struct {
	SerialNumber string `json:"serial_number"`
	DCNumber     string `json:"dc_number"`
	DCStatus     string `json:"dc_status"`
	ProductName  string `json:"product_name"`
}

// ValidateSerialNumbers handles POST /api/serial-numbers/validate.
func ValidateSerialNumbers(c echo.Context) error {
	var req SerialValidationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	if req.ProjectID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "project_id is required"})
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

	// Check against database (per-product if product_id specified)
	var conflicts []database.SerialConflict
	var err error
	if req.ProductID > 0 {
		conflicts, err = database.CheckSerialsInProjectByProduct(req.ProjectID, req.ProductID, unique, req.ExcludeDCID)
	} else {
		conflicts, err = database.CheckSerialsInProject(req.ProjectID, unique, req.ExcludeDCID)
	}
	if err != nil {
		slog.Error("Failed to validate serial numbers",
			slog.Int("project_id", req.ProjectID),
			slog.Int("product_id", req.ProductID),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to validate serial numbers"})
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

	return c.JSON(http.StatusOK, resp)
}

// IssueDCHandler handles POST /projects/:id/dcs/:dcid/issue.
func IssueDCHandler(c echo.Context) error {
	user := auth.GetCurrentUser(c)

	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid DC ID"})
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "DC not found"})
	}

	if dc.Status != "draft" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "DC is already issued"})
	}

	// Validate DC has line items with serial numbers
	lineItems, err := database.GetLineItemsByDCID(dcID)
	if err != nil || len(lineItems) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "DC must have at least one line item to be issued"})
	}

	// Serial numbers are required for transit DCs only
	if dc.DCType == "transit" {
		for _, li := range lineItems { //nolint:gocritic
			serials, _ := database.GetSerialNumbersByLineItemID(li.ID)
			if len(serials) == 0 {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "All line items must have serial numbers before issuing"})
			}
			if len(serials) != li.Quantity {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Serial number count must match quantity for all line items"})
			}
		}
	}

	if err := database.IssueDC(dcID, user.ID); err != nil {
		slog.Error("Failed to issue DC",
			slog.Int("dc_id", dcID),
			slog.Int("project_id", projectID),
			slog.Int("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to issue DC: " + err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "DC issued successfully",
	})
}

// DeleteDCHandler handles DELETE /projects/:id/dcs/:dcid.
func DeleteDCHandler(c echo.Context) error {
	projectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid project ID"})
	}

	dcID, err := strconv.Atoi(c.Param("dcid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid DC ID"})
	}

	dc, err := database.GetDeliveryChallanByID(dcID)
	if err != nil || dc.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "DC not found"})
	}

	if dc.Status != "draft" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Only draft DCs can be deleted"})
	}

	if err := database.DeleteDC(dcID); err != nil {
		slog.Error("Failed to delete DC",
			slog.Int("dc_id", dcID),
			slog.Int("project_id", projectID),
			slog.String("error", err.Error()),
		)
		if strings.Contains(err.Error(), "failed to delete") {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to delete DC"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "DC and all associated serial numbers deleted successfully",
		"dc_number": dc.DCNumber,
	})
}
