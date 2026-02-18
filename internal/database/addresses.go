package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
	"github.com/narendhupati/dc-management-tool/internal/models"
)

// q returns a sqlc Queries instance backed by the global DB.
func addressQueries() *db.Queries {
	return db.New(DB)
}

// mapAddressRow converts a sqlc GetAddressRow to a models.Address.
func mapAddressRow(r db.GetAddressRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapListAddressesRow converts a sqlc ListAddressesRow to a models.Address.
func mapListAddressesRow(r db.ListAddressesRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapListAddressesWithSearchRow converts a sqlc ListAddressesWithSearchRow to a models.Address.
func mapListAddressesWithSearchRow(r db.ListAddressesWithSearchRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapSearchAddressesForSelectorRow converts a sqlc SearchAddressesForSelectorRow to a models.Address.
func mapSearchAddressesForSelectorRow(r db.SearchAddressesForSelectorRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapSearchAddressesForSelectorSimpleRow converts a sqlc SearchAddressesForSelectorSimpleRow to a models.Address.
func mapSearchAddressesForSelectorSimpleRow(r db.SearchAddressesForSelectorSimpleRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapSearchAddressesNoFilterRow converts a sqlc SearchAddressesNoFilterRow to a models.Address.
func mapSearchAddressesNoFilterRow(r db.SearchAddressesNoFilterRow) *models.Address {
	return &models.Address{
		ID:           int(r.ID),
		ConfigID:     int(r.ConfigID),
		DataJSON:     r.AddressData,
		DistrictName: r.DistrictName,
		MandalName:   r.MandalName,
		MandalCode:   r.MandalCode,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

// mapAddressListConfig converts a sqlc AddressListConfig to a models.AddressListConfig.
func mapAddressListConfig(r db.AddressListConfig) *models.AddressListConfig {
	return &models.AddressListConfig{
		ID:          int(r.ID),
		ProjectID:   int(r.ProjectID),
		AddressType: r.AddressType,
		ColumnJSON:  r.ColumnDefinitions,
	}
}

// GetOrCreateAddressConfig gets the config for a project/type, creating a default if none exists.
func GetOrCreateAddressConfig(projectID int, addressType string) (*models.AddressListConfig, error) {
	ctx := context.Background()
	q := addressQueries()

	row, err := q.GetAddressConfig(ctx, db.GetAddressConfigParams{
		ProjectID:   int64(projectID),
		AddressType: addressType,
	})

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
		colJSON, marshalErr := json.Marshal(defaultCols)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal default columns: %w", marshalErr)
		}

		result, cfgErr := q.CreateAddressConfig(ctx, db.CreateAddressConfigParams{
			ProjectID:         int64(projectID),
			AddressType:       addressType,
			ColumnDefinitions: string(colJSON),
		})
		if cfgErr != nil {
			return nil, fmt.Errorf("failed to create default config: %w", cfgErr)
		}
		id, cfgErr := result.LastInsertId()
		if cfgErr != nil {
			return nil, fmt.Errorf("get insert ID: %w", cfgErr)
		}

		config := &models.AddressListConfig{
			ID:          int(id),
			ProjectID:   projectID,
			AddressType: addressType,
			ColumnJSON:  string(colJSON),
		}
		if parseErr := config.ParseColumns(); parseErr != nil {
			return nil, fmt.Errorf("failed to parse columns: %w", parseErr)
		}
		return config, nil
	} else if err != nil {
		return nil, err
	}

	config := mapAddressListConfig(row)
	if err := config.ParseColumns(); err != nil {
		return nil, fmt.Errorf("failed to parse columns: %w", err)
	}
	return config, nil
}

// UpdateAddressConfig updates the column definitions for a config.
func UpdateAddressConfig(configID int, columnJSON string) error {
	ctx := context.Background()
	return addressQueries().UpdateAddressConfig(ctx, db.UpdateAddressConfigParams{
		ColumnDefinitions: columnJSON,
		ID:                int64(configID),
	})
}

// BulkInsertAddresses inserts multiple addresses in a transaction.
func BulkInsertAddresses(configID int, addresses []*models.Address) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := db.New(tx)
	ctx := context.Background()

	for _, addr := range addresses {
		dataJSON, err := addr.DataToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize address data: %w", err)
		}
		_, err = qtx.InsertAddress(ctx, db.InsertAddressParams{
			ConfigID:     int64(configID),
			AddressData:  dataJSON,
			DistrictName: addr.DistrictName,
			MandalName:   addr.MandalName,
			MandalCode:   addr.MandalCode,
		})
		if err != nil {
			return fmt.Errorf("failed to insert address: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteAllAddresses deletes all addresses for a config.
func DeleteAllAddresses(configID int) error {
	ctx := context.Background()
	return addressQueries().DeleteAllAddresses(ctx, int64(configID))
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

	ctx := context.Background()
	q := addressQueries()

	var totalCount int64
	if search != "" {
		searchPattern := "%" + search + "%"
		count, err := q.CountAddressesWithSearch(ctx, db.CountAddressesWithSearchParams{
			ConfigID:     int64(configID),
			AddressData:  searchPattern,
			DistrictName: searchPattern,
			MandalName:   searchPattern,
			MandalCode:   searchPattern,
		})
		if err != nil {
			return nil, err
		}
		totalCount = count
	} else {
		count, err := q.CountAddresses(ctx, int64(configID))
		if err != nil {
			return nil, err
		}
		totalCount = count
	}

	var addresses []*models.Address

	if search != "" {
		searchPattern := "%" + search + "%"
		rows, err := q.ListAddressesWithSearch(ctx, db.ListAddressesWithSearchParams{
			ConfigID:     int64(configID),
			AddressData:  searchPattern,
			DistrictName: searchPattern,
			MandalName:   searchPattern,
			MandalCode:   searchPattern,
			Limit:        int64(perPage),
			Offset:       int64(offset),
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			a := mapListAddressesWithSearchRow(r)
			if err := a.ParseData(); err != nil {
				return nil, err
			}
			addresses = append(addresses, a)
		}
	} else {
		rows, err := q.ListAddresses(ctx, db.ListAddressesParams{
			ConfigID: int64(configID),
			Limit:    int64(perPage),
			Offset:   int64(offset),
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			a := mapListAddressesRow(r)
			if err := a.ParseData(); err != nil {
				return nil, err
			}
			addresses = append(addresses, a)
		}
	}

	total := int(totalCount)
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return &models.AddressPage{
		Addresses:   addresses,
		CurrentPage: page,
		PerPage:     perPage,
		TotalCount:  total,
		TotalPages:  totalPages,
	}, nil
}

// DeleteAddress deletes a single address by ID.
func DeleteAddress(addressID, configID int) error {
	ctx := context.Background()
	// We need to check rows affected; use hand-written SQL for that since sqlc DeleteAddress uses :exec.
	result, err := DB.ExecContext(ctx, `DELETE FROM addresses WHERE id = ? AND config_id = ?`, addressID, configID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("address not found")
	}
	return nil
}

// CountAddresses returns the total number of addresses for a config.
func CountAddresses(configID int) (int, error) {
	ctx := context.Background()
	count, err := addressQueries().CountAddresses(ctx, int64(configID))
	return int(count), err
}

// CreateAddress inserts a single address.
func CreateAddress(configID int, data map[string]string, districtName, mandalName, mandalCode string) (int, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	ctx := context.Background()
	result, err := addressQueries().InsertAddress(ctx, db.InsertAddressParams{
		ConfigID:     int64(configID),
		AddressData:  string(dataJSON),
		DistrictName: districtName,
		MandalName:   mandalName,
		MandalCode:   mandalCode,
	})
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get insert ID: %w", err)
	}
	return int(id), nil
}

// GetAddress gets a single address by ID.
func GetAddress(addressID int) (*models.Address, error) {
	ctx := context.Background()
	row, err := addressQueries().GetAddress(ctx, int64(addressID))
	if err != nil {
		return nil, err
	}
	a := mapAddressRow(row)
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
	ctx := context.Background()
	return addressQueries().UpdateAddress(ctx, db.UpdateAddressParams{
		AddressData:  string(dataJSON),
		DistrictName: districtName,
		MandalName:   mandalName,
		MandalCode:   mandalCode,
		ID:           int64(addressID),
	})
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

	ctx := context.Background()
	q := addressQueries()

	var addresses []*models.Address

	if search != "" {
		searchPattern := "%" + search + "%"
		if addressType == "ship_to" {
			rows, err := q.SearchAddressesForSelector(ctx, db.SearchAddressesForSelectorParams{
				ConfigID:     int64(configID),
				AddressData:  searchPattern,
				DistrictName: searchPattern,
				MandalName:   searchPattern,
				MandalCode:   searchPattern,
				Limit:        int64(limit),
			})
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				a := mapSearchAddressesForSelectorRow(r)
				if err := a.ParseData(); err != nil {
					return nil, err
				}
				addresses = append(addresses, a)
			}
		} else {
			rows, err := q.SearchAddressesForSelectorSimple(ctx, db.SearchAddressesForSelectorSimpleParams{
				ConfigID:    int64(configID),
				AddressData: searchPattern,
				Limit:       int64(limit),
			})
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				a := mapSearchAddressesForSelectorSimpleRow(r)
				if err := a.ParseData(); err != nil {
					return nil, err
				}
				addresses = append(addresses, a)
			}
		}
	} else {
		rows, err := q.SearchAddressesNoFilter(ctx, db.SearchAddressesNoFilterParams{
			ConfigID: int64(configID),
			Limit:    int64(limit),
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			a := mapSearchAddressesNoFilterRow(r)
			if err := a.ParseData(); err != nil {
				return nil, err
			}
			addresses = append(addresses, a)
		}
	}

	return addresses, nil
}
