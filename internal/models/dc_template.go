package models

import (
	"strings"
	"time"
)

type DCTemplate struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	Name      string    `json:"name"`
	Purpose   string    `json:"purpose"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Computed fields
	ProductCount    int `json:"product_count"`
	TransitDCCount  int `json:"transit_dc_count"`
	OfficialDCCount int `json:"official_dc_count"`
	UsageCount      int `json:"usage_count"` // TransitDCCount + OfficialDCCount
}

type DCTemplateProduct struct {
	TemplateID      int `json:"template_id"`
	ProductID       int `json:"product_id"`
	DefaultQuantity int `json:"default_quantity"`
}

// TemplateProductRow represents a product in a template with its default quantity
type TemplateProductRow struct {
	Product
	DefaultQuantity int `json:"default_quantity"`
}

func (t *DCTemplate) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(t.Name) == "" {
		errors["name"] = "Template name is required"
	} else if len(t.Name) > 100 {
		errors["name"] = "Template name must be 100 characters or less"
	}

	if len(t.Purpose) > 500 {
		errors["purpose"] = "Purpose must be 500 characters or less"
	}

	return errors
}
