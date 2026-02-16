package models

import (
	"encoding/json"
	"time"
)

// Address stores a single address with flexible JSON data.
type Address struct {
	ID        int               `json:"id"`
	ConfigID  int               `json:"config_id"`
	Data      map[string]string `json:"-"`
	DataJSON  string            `json:"-"` // raw JSON from DB
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
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
