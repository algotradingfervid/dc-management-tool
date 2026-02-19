package models

import (
	"time"
)

type DCTemplate struct {
	ID        int       `json:"id"`
	ProjectID int       `json:"project_id"`
	Name      string    `json:"name" validate:"required,max=100"`
	Purpose   string    `json:"purpose" validate:"max=500"`
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
