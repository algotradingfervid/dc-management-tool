package models

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// ColumnDefinition defines a single column in an address list configuration.
type ColumnDefinition struct {
	Name           string `json:"name"`
	Required       bool   `json:"required"`
	Type           string `json:"type,omitempty"`             // text, number, email, phone - defaults to text
	Validation     string `json:"validation,omitempty"`       // regex pattern
	Fixed          bool   `json:"fixed,omitempty"`            // true for ship-to district/mandal fields (not user-editable)
	ShowInTable    *bool  `json:"show_in_table,omitempty"`    // nil = true (visible by default)
	ShowInPrint    *bool  `json:"show_in_print,omitempty"`    // nil = true (visible by default)
	TableSortOrder int    `json:"table_sort_order,omitempty"` // 0 = use array index
	PrintSortOrder int    `json:"print_sort_order,omitempty"` // 0 = use array index
}

// IsVisibleInTable returns whether this column should appear in the address listing table.
func (cd *ColumnDefinition) IsVisibleInTable() bool {
	return cd.ShowInTable == nil || *cd.ShowInTable
}

// IsVisibleInPrint returns whether this column should appear in printed documents/PDFs.
func (cd *ColumnDefinition) IsVisibleInPrint() bool {
	return cd.ShowInPrint == nil || *cd.ShowInPrint
}

// GetTableSortOrder returns the effective table sort order (falls back to a large value for stable sorting).
func (cd *ColumnDefinition) GetTableSortOrder(index int) int {
	if cd.TableSortOrder > 0 {
		return cd.TableSortOrder
	}
	return 1000 + index
}

// GetPrintSortOrder returns the effective print sort order (falls back to a large value for stable sorting).
func (cd *ColumnDefinition) GetPrintSortOrder(index int) int {
	if cd.PrintSortOrder > 0 {
		return cd.PrintSortOrder
	}
	return 1000 + index
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
// For ship_to configs, it ensures fixed columns are always present by merging
// defaults for any that are missing from the saved JSON.
func (c *AddressListConfig) ParseColumns() error {
	if c.ColumnJSON == "" {
		c.ColumnDefinitions = []ColumnDefinition{}
	} else if err := json.Unmarshal([]byte(c.ColumnJSON), &c.ColumnDefinitions); err != nil {
		return err
	}

	// Ensure configs with fixed columns always include them
	if fixed := FixedColumnsForType(c.AddressType); len(fixed) > 0 {
		c.ensureFixedColumns(fixed)
	}
	return nil
}

// FixedColumnsForType returns the fixed columns for a given address type.
// Returns nil for address types that have no fixed columns.
// To add fixed columns to a new address type, add a case here.
func FixedColumnsForType(addressType string) []ColumnDefinition {
	switch addressType {
	case "ship_to":
		return FixedShipToColumns()
	default:
		return nil
	}
}

// ensureFixedColumns checks that all fixed columns are present in ColumnDefinitions.
// Missing fixed columns are prepended with their defaults.
// Also removes any non-fixed duplicates that share a name with a fixed column.
func (c *AddressListConfig) ensureFixedColumns(requiredFixed []ColumnDefinition) {
	fixedNames := make(map[string]bool)
	for _, fcol := range requiredFixed {
		fixedNames[fcol.Name] = true
	}

	// Find existing fixed columns and remove non-fixed duplicates
	existing := make(map[string]bool)
	var cleaned []ColumnDefinition
	for _, col := range c.ColumnDefinitions {
		if col.Fixed {
			existing[col.Name] = true
			cleaned = append(cleaned, col)
		} else if fixedNames[col.Name] {
			// Skip non-fixed duplicates of fixed column names
			continue
		} else {
			cleaned = append(cleaned, col)
		}
	}

	// Prepend any missing fixed columns
	var missing []ColumnDefinition
	for _, fcol := range requiredFixed {
		if !existing[fcol.Name] {
			missing = append(missing, fcol)
		}
	}

	if len(missing) > 0 {
		cleaned = append(missing, cleaned...)
	}
	c.ColumnDefinitions = cleaned
}

// ColumnsToJSON serializes ColumnDefinitions to JSON string.
func (c *AddressListConfig) ColumnsToJSON() (string, error) {
	b, err := json.Marshal(c.ColumnDefinitions)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// TableVisibleColumns returns columns visible in the table, sorted by TableSortOrder.
func (c *AddressListConfig) TableVisibleColumns() []ColumnDefinition {
	var visible []ColumnDefinition
	for i, col := range c.ColumnDefinitions {
		if col.IsVisibleInTable() {
			cp := col
			if cp.TableSortOrder == 0 {
				cp.TableSortOrder = 1000 + i
			}
			visible = append(visible, cp)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		return visible[i].TableSortOrder < visible[j].TableSortOrder
	})
	return visible
}

// DynamicTableVisibleColumns returns only non-fixed columns visible in the table, sorted by TableSortOrder.
func (c *AddressListConfig) DynamicTableVisibleColumns() []ColumnDefinition {
	var visible []ColumnDefinition
	for i, col := range c.ColumnDefinitions {
		if !col.Fixed && col.IsVisibleInTable() {
			cp := col
			if cp.TableSortOrder == 0 {
				cp.TableSortOrder = 1000 + i
			}
			visible = append(visible, cp)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		return visible[i].TableSortOrder < visible[j].TableSortOrder
	})
	return visible
}

// PrintVisibleColumns returns columns visible in print/PDF output, sorted by PrintSortOrder.
func (c *AddressListConfig) PrintVisibleColumns() []ColumnDefinition {
	var visible []ColumnDefinition
	for i, col := range c.ColumnDefinitions {
		if col.IsVisibleInPrint() {
			cp := col
			if cp.PrintSortOrder == 0 {
				cp.PrintSortOrder = 1000 + i
			}
			visible = append(visible, cp)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool {
		return visible[i].PrintSortOrder < visible[j].PrintSortOrder
	})
	return visible
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

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool { return &b }

// FixedShipToColumns returns the fixed columns for ship-to addresses.
// These are always present and cannot be removed by users.
func FixedShipToColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "District Name", Required: true, Type: "text", Fixed: true, ShowInTable: boolPtr(true), ShowInPrint: boolPtr(true), TableSortOrder: 1, PrintSortOrder: 1},
		{Name: "Mandal/ULB Name", Required: true, Type: "text", Fixed: true, ShowInTable: boolPtr(true), ShowInPrint: boolPtr(true), TableSortOrder: 2, PrintSortOrder: 2},
		{Name: "Mandal Code", Required: true, Type: "text", Fixed: true, ShowInTable: boolPtr(true), ShowInPrint: boolPtr(true), TableSortOrder: 3, PrintSortOrder: 3},
	}
}

// FixedColumns returns the fixed column definitions from the config.
// If no fixed columns are saved, returns defaults from FixedShipToColumns().
func (c *AddressListConfig) FixedColumns() []ColumnDefinition {
	var fixed []ColumnDefinition
	for _, col := range c.ColumnDefinitions {
		if col.Fixed {
			fixed = append(fixed, col)
		}
	}
	if len(fixed) == 0 && c.AddressType == "ship_to" {
		return FixedShipToColumns()
	}
	return fixed
}

// DynamicColumns returns only the non-fixed column definitions.
func (c *AddressListConfig) DynamicColumns() []ColumnDefinition {
	var dynamic []ColumnDefinition
	for _, col := range c.ColumnDefinitions {
		if !col.Fixed {
			dynamic = append(dynamic, col)
		}
	}
	return dynamic
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

// DefaultShipToColumns returns default column definitions for ship-to addresses.
// Note: District Name, Mandal/ULB Name, and Mandal Code are fixed columns
// stored in dedicated DB fields. The columns here are additional dynamic columns.
func DefaultShipToColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "Location", Required: true, Type: "text"},
		{Name: "Location ID", Required: true, Type: "text"},
		{Name: "SRO", Required: false, Type: "text"},
		{Name: "Secretariat Name", Required: true, Type: "text"},
		{Name: "Secretariat Code", Required: true, Type: "text"},
	}
}

// DefaultBillFromColumns returns default column definitions for bill-from addresses.
func DefaultBillFromColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "Company Name", Required: true, Type: "text"},
		{Name: "Address Line 1", Required: true, Type: "text"},
		{Name: "Address Line 2", Required: false, Type: "text"},
		{Name: "City", Required: true, Type: "text"},
		{Name: "State", Required: true, Type: "text"},
		{Name: "PIN Code", Required: false, Type: "text"},
		{Name: "GSTIN", Required: false, Type: "text"},
		{Name: "Email", Required: false, Type: "text"},
		{Name: "CIN No.", Required: false, Type: "text"},
		{Name: "PAN", Required: false, Type: "text"},
	}
}

// DefaultDispatchFromColumns returns default column definitions for dispatch-from addresses.
func DefaultDispatchFromColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "Company Name", Required: true, Type: "text"},
		{Name: "Address Line 1", Required: true, Type: "text"},
		{Name: "Address Line 2", Required: false, Type: "text"},
		{Name: "City", Required: true, Type: "text"},
		{Name: "State", Required: true, Type: "text"},
		{Name: "PIN Code", Required: false, Type: "text"},
		{Name: "Contact Person", Required: false, Type: "text"},
		{Name: "Phone", Required: false, Type: "text"},
	}
}
