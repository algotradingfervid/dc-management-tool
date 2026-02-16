package models

import (
	"encoding/json"
	"strings"
	"time"
)

// ColumnDefinition defines a single column in an address list configuration.
type ColumnDefinition struct {
	Name       string `json:"name"`
	Required   bool   `json:"required"`
	Type       string `json:"type,omitempty"`       // text, number, email, phone - defaults to text
	Validation string `json:"validation,omitempty"` // regex pattern
}

// AddressListConfig stores the column configuration for a project's address list.
type AddressListConfig struct {
	ID                int                `json:"id"`
	ProjectID         int                `json:"project_id"`
	AddressType       string             `json:"address_type"` // bill_to, ship_to
	ColumnDefinitions []ColumnDefinition `json:"-"`
	ColumnJSON        string             `json:"-"` // raw JSON from DB
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// ParseColumns parses the JSON column definitions string into the struct.
func (c *AddressListConfig) ParseColumns() error {
	if c.ColumnJSON == "" {
		c.ColumnDefinitions = []ColumnDefinition{}
		return nil
	}
	return json.Unmarshal([]byte(c.ColumnJSON), &c.ColumnDefinitions)
}

// ColumnsToJSON serializes ColumnDefinitions to JSON string.
func (c *AddressListConfig) ColumnsToJSON() (string, error) {
	b, err := json.Marshal(c.ColumnDefinitions)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ValidateColumns validates the column definitions.
func (c *AddressListConfig) ValidateColumns() map[string]string {
	errors := make(map[string]string)

	if len(c.ColumnDefinitions) == 0 {
		errors["columns"] = "At least one column is required"
		return errors
	}

	seen := make(map[string]bool)
	for i, col := range c.ColumnDefinitions {
		name := strings.TrimSpace(col.Name)
		if name == "" {
			errors["columns"] = "All columns must have a name"
			return errors
		}
		lower := strings.ToLower(name)
		if seen[lower] {
			errors["columns"] = "Column names must be unique (duplicate: " + name + ")"
			return errors
		}
		seen[lower] = true

		// Default type to text
		if c.ColumnDefinitions[i].Type == "" {
			c.ColumnDefinitions[i].Type = "text"
		}
	}

	return errors
}

// DefaultBillToColumns returns default column definitions for bill-to addresses.
func DefaultBillToColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "Company Name", Required: true, Type: "text"},
		{Name: "GSTIN", Required: true, Type: "text"},
		{Name: "Address Line 1", Required: true, Type: "text"},
		{Name: "City", Required: true, Type: "text"},
		{Name: "State", Required: true, Type: "text"},
		{Name: "PIN Code", Required: true, Type: "text"},
	}
}
