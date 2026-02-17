package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/narendhupati/dc-management-tool/internal/models"
)

// GetOrCreateAddressConfig gets the config for a project/type, creating a default if none exists.
func GetOrCreateAddressConfig(projectID int, addressType string) (*models.AddressListConfig, error) {
	config := &models.AddressListConfig{}

	err := DB.QueryRow(
		`SELECT id, project_id, address_type, column_definitions, created_at, updated_at
		 FROM address_list_configs WHERE project_id = ? AND address_type = ?`,
		projectID, addressType,
	).Scan(&config.ID, &config.ProjectID, &config.AddressType, &config.ColumnJSON, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default config based on address type
		var defaultCols []models.ColumnDefinition
		switch addressType {
		case "ship_to":
			defaultCols = models.DefaultShipToColumns()
		case "bill_from":
			defaultCols = models.DefaultBillFromColumns()
		case "dispatch_from":
			defaultCols = models.DefaultDispatchFromColumns()
		default:
			defaultCols = models.DefaultBillToColumns()
		}
		colJSON, _ := json.Marshal(defaultCols)

		result, err := DB.Exec(
			`INSERT INTO address_list_configs (project_id, address_type, column_definitions) VALUES (?, ?, ?)`,
			projectID, addressType, string(colJSON),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		id, _ := result.LastInsertId()
		config.ID = int(id)
		config.ProjectID = projectID
		config.AddressType = addressType
		config.ColumnJSON = string(colJSON)
	} else if err != nil {
		return nil, err
	}

	if err := config.ParseColumns(); err != nil {
		return nil, fmt.Errorf("failed to parse columns: %w", err)
	}

	return config, nil
}

// UpdateAddressConfig updates the column definitions for a config.
func UpdateAddressConfig(configID int, columnJSON string) error {
	_, err := DB.Exec(
		`UPDATE address_list_configs SET column_definitions = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		columnJSON, configID,
	)
	return err
}

// BulkInsertAddresses inserts multiple addresses in a transaction.
func BulkInsertAddresses(configID int, addresses []*models.Address) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO addresses (config_id, address_data, district_name, mandal_name, mandal_code) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, addr := range addresses {
		dataJSON, err := addr.DataToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize address data: %w", err)
		}
		_, err = stmt.Exec(configID, dataJSON, addr.DistrictName, addr.MandalName, addr.MandalCode)
		if err != nil {
			return fmt.Errorf("failed to insert address: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteAllAddresses deletes all addresses for a config.
func DeleteAllAddresses(configID int) error {
	_, err := DB.Exec(`DELETE FROM addresses WHERE config_id = ?`, configID)
	return err
}

// ListAddresses returns paginated addresses for a config.
func ListAddresses(configID, page, perPage int, search string) (*models.AddressPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	// Count total
	var countQuery string
	var countArgs []interface{}
	if search != "" {
		searchPattern := "%" + search + "%"
		countQuery = `SELECT COUNT(*) FROM addresses WHERE config_id = ? AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?)`
		countArgs = []interface{}{configID, searchPattern, searchPattern, searchPattern, searchPattern}
	} else {
		countQuery = `SELECT COUNT(*) FROM addresses WHERE config_id = ?`
		countArgs = []interface{}{configID}
	}

	var totalCount int
	if err := DB.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Fetch addresses
	var dataQuery string
	var dataArgs []interface{}
	if search != "" {
		searchPattern := "%" + search + "%"
		dataQuery = `SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
			FROM addresses WHERE config_id = ? AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?)
			ORDER BY id DESC LIMIT ? OFFSET ?`
		dataArgs = []interface{}{configID, searchPattern, searchPattern, searchPattern, searchPattern, perPage, offset}
	} else {
		dataQuery = `SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
			FROM addresses WHERE config_id = ?
			ORDER BY id DESC LIMIT ? OFFSET ?`
		dataArgs = []interface{}{configID, perPage, offset}
	}

	rows, err := DB.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []*models.Address
	for rows.Next() {
		a := &models.Address{}
		if err := rows.Scan(&a.ID, &a.ConfigID, &a.DataJSON, &a.DistrictName, &a.MandalName, &a.MandalCode, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if err := a.ParseData(); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return &models.AddressPage{
		Addresses:   addresses,
		CurrentPage: page,
		PerPage:     perPage,
		TotalCount:  totalCount,
		TotalPages:  totalPages,
	}, nil
}

// DeleteAddress deletes a single address by ID.
func DeleteAddress(addressID, configID int) error {
	result, err := DB.Exec(`DELETE FROM addresses WHERE id = ? AND config_id = ?`, addressID, configID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("address not found")
	}
	return nil
}

// CountAddresses returns the total number of addresses for a config.
func CountAddresses(configID int) (int, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM addresses WHERE config_id = ?`, configID).Scan(&count)
	return count, err
}

// CreateAddress inserts a single address.
func CreateAddress(configID int, data map[string]string, districtName, mandalName, mandalCode string) (int, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	result, err := DB.Exec(
		`INSERT INTO addresses (config_id, address_data, district_name, mandal_name, mandal_code) VALUES (?, ?, ?, ?, ?)`,
		configID, string(dataJSON), districtName, mandalName, mandalCode,
	)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return int(id), nil
}

// GetAddress gets a single address by ID.
func GetAddress(addressID int) (*models.Address, error) {
	a := &models.Address{}
	err := DB.QueryRow(
		`SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at FROM addresses WHERE id = ?`,
		addressID,
	).Scan(&a.ID, &a.ConfigID, &a.DataJSON, &a.DistrictName, &a.MandalName, &a.MandalCode, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := a.ParseData(); err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateAddress updates a single address's data.
func UpdateAddress(addressID int, data map[string]string, districtName, mandalName, mandalCode string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = DB.Exec(
		`UPDATE addresses SET address_data = ?, district_name = ?, mandal_name = ?, mandal_code = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		string(dataJSON), districtName, mandalName, mandalCode, addressID,
	)
	return err
}

// ValidateAddressData validates address data against column definitions.
func ValidateAddressData(data map[string]string, columns []models.ColumnDefinition) []string {
	var errs []string
	for _, col := range columns {
		val := strings.TrimSpace(data[col.Name])
		if col.Required && val == "" {
			errs = append(errs, fmt.Sprintf("%s is required", col.Name))
		}
	}
	return errs
}

// SearchAddressesForSelector returns addresses matching a search query for the HTMX selector component.
func SearchAddressesForSelector(configID int, search string, addressType string, limit int) ([]*models.Address, error) {
	if limit < 1 {
		limit = 20
	}

	var query string
	var args []interface{}

	if search != "" {
		searchPattern := "%" + search + "%"
		if addressType == "ship_to" {
			query = `SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
				FROM addresses WHERE config_id = ? AND (address_data LIKE ? OR district_name LIKE ? OR mandal_name LIKE ? OR mandal_code LIKE ?)
				ORDER BY district_name, mandal_name LIMIT ?`
			args = []interface{}{configID, searchPattern, searchPattern, searchPattern, searchPattern, limit}
		} else {
			query = `SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
				FROM addresses WHERE config_id = ? AND address_data LIKE ?
				ORDER BY id LIMIT ?`
			args = []interface{}{configID, searchPattern, limit}
		}
	} else {
		query = `SELECT id, config_id, address_data, district_name, mandal_name, mandal_code, created_at, updated_at
			FROM addresses WHERE config_id = ?
			ORDER BY id LIMIT ?`
		args = []interface{}{configID, limit}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []*models.Address
	for rows.Next() {
		a := &models.Address{}
		if err := rows.Scan(&a.ID, &a.ConfigID, &a.DataJSON, &a.DistrictName, &a.MandalName, &a.MandalCode, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if err := a.ParseData(); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}

	return addresses, nil
}
