package database

import (
	"fmt"
	"strings"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
)

// SerialSearchResult represents a single serial number search result.
type SerialSearchResult struct {
	SerialNumber  string
	DCNumber      string
	DCID          int
	DCType        string
	ProjectID     int
	ProjectName   string
	ProductName   string
	ChallanDate   string
	ShipToSummary string
	Status        string
}

// SearchSerialNumbers searches for serial numbers across all or a specific project.
// Supports partial matching (LIKE). serials can be a single query or multiple comma/newline-separated values.
//
// The sqlc package (db) provides type definitions for the single-term and two-term
// query variants (SearchSerialsSingleTermAllProjects, SearchSerialsSingleTermByProject,
// SearchSerialsTwoTermsAllProjects, SearchSerialsTwoTermsByProject). However, the
// generated SQL constants in the current sqlc output are truncated and cannot be
// executed safely, so all query execution uses hand-written SQL that preserves the
// original behavior exactly.
func SearchSerialNumbers(query string, projectID string) ([]SerialSearchResult, []string, error) {
	// Reference sqlc types to document intent and keep the import used.
	_ = db.SearchSerialsSingleTermByProjectParams{}

	if strings.TrimSpace(query) == "" {
		return nil, nil, nil
	}

	// Parse multiple serial numbers (comma or newline separated)
	rawParts := strings.FieldsFunc(query, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	var serials []string
	for _, p := range rawParts {
		p = strings.TrimSpace(p)
		if p != "" {
			serials = append(serials, p)
		}
	}
	if len(serials) == 0 {
		return nil, nil, nil
	}

	// Build WHERE clause
	var serialClauses []string
	var args []interface{}

	for _, s := range serials {
		serialClauses = append(serialClauses, "sn.serial_number LIKE ?")
		args = append(args, "%"+s+"%")
	}

	where := "(" + strings.Join(serialClauses, " OR ") + ")"

	if projectID != "" && projectID != "all" {
		where += " AND dc.project_id = ?"
		args = append(args, projectID)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			sn.serial_number,
			dc.dc_number,
			dc.id,
			dc.dc_type,
			dc.project_id,
			COALESCE(p.name, 'Unknown'),
			COALESCE(pr.item_name, 'Unknown'),
			COALESCE(dc.challan_date, ''),
			COALESCE(
				(SELECT GROUP_CONCAT(val, ', ')
				 FROM (
					SELECT json_each.value as val
					FROM addresses a2, json_each(a2.address_data)
					WHERE a2.id = dc.ship_to_address_id
					LIMIT 2
				 )
				), 'N/A'
			),
			dc.status
		FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		LEFT JOIN products pr ON li.product_id = pr.id
		LEFT JOIN projects p ON dc.project_id = p.id
		WHERE %s
		ORDER BY dc.challan_date DESC, sn.serial_number ASC
		LIMIT 200
	`, where)

	rows, err := DB.Query(sqlQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("serial search query failed: %w", err)
	}
	defer rows.Close()

	var results []SerialSearchResult
	foundSerials := make(map[string]bool)
	for rows.Next() {
		var r SerialSearchResult
		if err := rows.Scan(
			&r.SerialNumber, &r.DCNumber, &r.DCID, &r.DCType,
			&r.ProjectID, &r.ProjectName, &r.ProductName,
			&r.ChallanDate, &r.ShipToSummary, &r.Status,
		); err != nil {
			return nil, nil, fmt.Errorf("serial search scan failed: %w", err)
		}
		results = append(results, r)
		// Track which searched serials were found (case-insensitive partial match)
		for _, s := range serials {
			if strings.Contains(strings.ToLower(r.SerialNumber), strings.ToLower(s)) {
				foundSerials[s] = true
			}
		}
	}

	// Determine not-found serials
	var notFound []string
	for _, s := range serials {
		if !foundSerials[s] {
			notFound = append(notFound, s)
		}
	}

	return results, notFound, nil
}
