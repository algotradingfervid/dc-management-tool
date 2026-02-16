package models

import (
	"regexp"
	"strings"
	"time"
)

type Product struct {
	ID              int       `json:"id"`
	ProjectID       int       `json:"project_id"`
	ItemName        string    `json:"item_name"`
	ItemDescription string    `json:"item_description"`
	HSNCode         string    `json:"hsn_code"`
	UoM             string    `json:"uom"`
	BrandModel      string    `json:"brand_model"`
	PerUnitPrice    float64   `json:"per_unit_price"`
	GSTPercentage   float64   `json:"gst_percentage"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (p *Product) PriceWithGST() float64 {
	return p.PerUnitPrice * (1 + p.GSTPercentage/100)
}

var hsnCodeRegex = regexp.MustCompile(`^\d{6,8}$`)

func (p *Product) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(p.ItemName) == "" {
		errors["item_name"] = "Item name is required"
	}

	if strings.TrimSpace(p.ItemDescription) == "" {
		errors["item_description"] = "Item description is required"
	}

	if strings.TrimSpace(p.HSNCode) != "" && !hsnCodeRegex.MatchString(strings.TrimSpace(p.HSNCode)) {
		errors["hsn_code"] = "HSN code must be 6-8 digits"
	}

	if strings.TrimSpace(p.UoM) == "" {
		errors["uom"] = "Unit of measurement is required"
	}

	if strings.TrimSpace(p.BrandModel) == "" {
		errors["brand_model"] = "Brand/Model is required"
	}

	if p.PerUnitPrice <= 0 {
		errors["per_unit_price"] = "Price must be greater than 0"
	}

	if p.GSTPercentage < 0 || p.GSTPercentage > 100 {
		errors["gst_percentage"] = "GST percentage must be between 0 and 100"
	}

	return errors
}
