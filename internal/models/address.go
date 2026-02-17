package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Address stores a single address with flexible JSON data and optional fixed fields.
type Address struct {
	ID           int               `json:"id"`
	ConfigID     int               `json:"config_id"`
	Data         map[string]string `json:"-"`
	DataJSON     string            `json:"-"` // raw JSON from DB
	DistrictName string            `json:"district_name"` // fixed field for ship-to
	MandalName   string            `json:"mandal_name"`   // fixed field for ship-to
	MandalCode   string            `json:"mandal_code"`   // fixed field for ship-to
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ParseData deserializes the JSON data field.
func (a *Address) ParseData() error {
	if a.DataJSON == "" {
		a.Data = make(map[string]string)
		return nil
	}
	return json.Unmarshal([]byte(a.DataJSON), &a.Data)
}

// DataToJSON serializes the Data map to JSON.
func (a *Address) DataToJSON() (string, error) {
	b, err := json.Marshal(a.Data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DisplayName returns a human-readable label built from all address fields.
func (a *Address) DisplayName() string {
	var parts []string

	// Fixed fields first
	if a.DistrictName != "" {
		parts = append(parts, a.DistrictName)
	}
	if a.MandalName != "" {
		parts = append(parts, a.MandalName)
	}

	// Dynamic fields in sorted key order for consistency
	keys := make([]string, 0, len(a.Data))
	for k := range a.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := strings.TrimSpace(a.Data[k])
		if v != "" {
			parts = append(parts, v)
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Address #%d", a.ID)
	}
	return strings.Join(parts, " | ")
}

// AddressPage represents a paginated list of addresses.
type AddressPage struct {
	Addresses   []*Address `json:"addresses"`
	CurrentPage int        `json:"current_page"`
	PerPage     int        `json:"per_page"`
	TotalCount  int        `json:"total_count"`
	TotalPages  int        `json:"total_pages"`
}

// UploadResult holds the result of a bulk address upload.
type UploadResult struct {
	TotalRows  int           `json:"total_rows"`
	Successful int           `json:"successful"`
	Failed     int           `json:"failed"`
	Mode       string        `json:"mode"`
	Errors     []UploadError `json:"errors,omitempty"`
}

// UploadError describes a validation error for a specific row.
type UploadError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Error string `json:"error"`
}
