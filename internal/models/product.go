package models

import (
	"time"
)

type Product struct {
	ID              int       `json:"id"`
	ProjectID       int       `json:"project_id"`
	ItemName        string    `json:"item_name" validate:"required,max=255"`
	ItemDescription string    `json:"item_description" validate:"required,max=1000"`
	HSNCode         string    `json:"hsn_code"`
	UoM             string    `json:"uom" validate:"required,max=50"`
	BrandModel      string    `json:"brand_model" validate:"required,max=255"`
	PerUnitPrice    float64   `json:"per_unit_price" validate:"required,gt=0"`
	GSTPercentage   float64   `json:"gst_percentage" validate:"gte=0,lte=100"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (p *Product) PriceWithGST() float64 {
	return p.PerUnitPrice * (1 + p.GSTPercentage/100)
}

// ProductPage holds paginated product results.
type ProductPage struct {
	Products    []*Product `json:"products"`
	CurrentPage int        `json:"current_page"`
	PerPage     int        `json:"per_page"`
	TotalCount  int        `json:"total_count"`
	TotalPages  int        `json:"total_pages"`
	Search      string     `json:"search"`
	SortBy      string     `json:"sort_by"`
	SortDir     string     `json:"sort_dir"`
}

// ProductImportResult holds the result of a bulk product import.
type ProductImportResult struct {
	TotalRows  int                  `json:"total_rows"`
	Successful int                  `json:"successful"`
	Failed     int                  `json:"failed"`
	Errors     []ProductImportError `json:"errors,omitempty"`
}

// ProductImportError describes a validation error for a specific row.
type ProductImportError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Error string `json:"error"`
}
