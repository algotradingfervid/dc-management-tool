package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// DCListItem represents a single DC row in the global listing.
type DCListItem struct {
	ID              int
	DCNumber        string
	DCType          string
	ChallanDate     string
	ProjectID       int
	ProjectName     string
	ShipToSummary   string
	Status          string
	TotalValue      *float64
	LineItemCount   int
	TotalQuantity   int
}

// DCListFilters holds all filter/sort/pagination parameters.
type DCListFilters struct {
	ProjectID string
	DCType    string // "all", "transit", "official"
	Status    string // "all", "draft", "issued"
	DateFrom  string
	DateTo    string
	Search    string
	SortBy    string // "dc_number", "challan_date", "status", "project_name"
	SortOrder string // "asc", "desc"
	Page      int
	PageSize  int
}

// DCListResult holds paginated results.
type DCListResult struct {
	DCs        []DCListItem
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
}

// ProjectOption is a lightweight project for filter dropdowns.
type ProjectOption struct {
	ID   int
	Name string
}

// GetAllDCsFiltered fetches all DCs across projects with filters, sorting, and pagination.
func GetAllDCsFiltered(filters DCListFilters) (*DCListResult, error) {
	whereClauses := []string{}
	args := []interface{}{}

	if filters.ProjectID != "" && filters.ProjectID != "all" {
		whereClauses = append(whereClauses, "dc.project_id = ?")
		args = append(args, filters.ProjectID)
	}

	if filters.DCType != "" && filters.DCType != "all" {
		whereClauses = append(whereClauses, "dc.dc_type = ?")
		args = append(args, filters.DCType)
	}

	if filters.Status != "" && filters.Status != "all" {
		whereClauses = append(whereClauses, "dc.status = ?")
		args = append(args, filters.Status)
	}

	if filters.DateFrom != "" {
		whereClauses = append(whereClauses, "dc.challan_date >= ?")
		args = append(args, filters.DateFrom)
	}

	if filters.DateTo != "" {
		whereClauses = append(whereClauses, "dc.challan_date <= ?")
		args = append(args, filters.DateTo)
	}

	if filters.Search != "" {
		whereClauses = append(whereClauses, "dc.dc_number LIKE ?")
		args = append(args, "%"+filters.Search+"%")
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM delivery_challans dc %s`, whereClause)
	var totalCount int
	if err := DB.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	// Sort
	allowedSorts := map[string]string{
		"dc_number":    "dc.dc_number",
		"challan_date": "dc.challan_date",
		"status":       "dc.status",
		"project_name": "p.name",
	}
	sortCol := "dc.challan_date"
	if col, ok := allowedSorts[filters.SortBy]; ok {
		sortCol = col
	}
	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	orderBy := fmt.Sprintf("%s %s", sortCol, sortOrder)

	// Pagination
	if filters.PageSize <= 0 {
		filters.PageSize = 25
	}
	if filters.Page <= 0 {
		filters.Page = 1
	}
	offset := (filters.Page - 1) * filters.PageSize

	// Main query with JOINs
	query := fmt.Sprintf(`
		SELECT
			dc.id,
			dc.dc_number,
			dc.dc_type,
			COALESCE(dc.challan_date, ''),
			dc.project_id,
			COALESCE(p.name, 'Unknown'),
			COALESCE(
				(SELECT GROUP_CONCAT(val, ', ')
				 FROM (
					SELECT json_each.value as val
					FROM addresses a2, json_each(a2.address_data)
					WHERE a2.id = dc.ship_to_address_id
					LIMIT 2
				 )
				), 'N/A'
			) as ship_to_summary,
			dc.status,
			(SELECT SUM(li.total_amount) FROM dc_line_items li WHERE li.dc_id = dc.id) as total_value,
			(SELECT COUNT(*) FROM dc_line_items li WHERE li.dc_id = dc.id) as line_item_count,
			(SELECT COALESCE(SUM(li.quantity), 0) FROM dc_line_items li WHERE li.dc_id = dc.id) as total_quantity
		FROM delivery_challans dc
		LEFT JOIN projects p ON dc.project_id = p.id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, whereClause, orderBy)

	queryArgs := make([]interface{}, len(args))
	copy(queryArgs, args)
	queryArgs = append(queryArgs, filters.PageSize, offset)

	rows, err := DB.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var dcs []DCListItem
	for rows.Next() {
		var dc DCListItem
		var totalVal sql.NullFloat64
		err := rows.Scan(
			&dc.ID, &dc.DCNumber, &dc.DCType, &dc.ChallanDate,
			&dc.ProjectID, &dc.ProjectName, &dc.ShipToSummary,
			&dc.Status, &totalVal, &dc.LineItemCount, &dc.TotalQuantity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if totalVal.Valid {
			dc.TotalValue = &totalVal.Float64
		}
		dcs = append(dcs, dc)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + filters.PageSize - 1) / filters.PageSize
	}

	return &DCListResult{
		DCs:        dcs,
		TotalCount: totalCount,
		Page:       filters.Page,
		PageSize:   filters.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetAllProjectOptions returns all projects for the filter dropdown.
func GetAllProjectOptions() ([]ProjectOption, error) {
	rows, err := DB.Query("SELECT id, name FROM projects ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ProjectOption
	for rows.Next() {
		var p ProjectOption
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
