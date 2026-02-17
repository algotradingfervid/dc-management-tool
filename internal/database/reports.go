package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	db "github.com/narendhupati/dc-management-tool/internal/database/sqlc"
)

// DCSummaryReport holds aggregate stats for the DC summary report.
type DCSummaryReport struct {
	TransitDraftDCs      int
	TransitIssuedDCs     int
	OfficialDraftDCs     int
	OfficialIssuedDCs    int
	TotalItemsDispatched int
	TotalSerialsUsed     int
}

// DestinationRow holds one row of the destination-wise report.
type DestinationRow struct {
	District    string
	Mandal      string
	OfficialDCs int
	TotalItems  int
	DraftCount  int
	IssuedCount int
}

// ProductReportRow holds one row of the product-wise report.
type ProductReportRow struct {
	ProductName      string
	TotalQty         int
	DCCount          int
	DestinationCount int
}

// SerialReportRow holds one row of the serial number report.
type SerialReportRow struct {
	SerialNumber    string
	ProductName     string
	TransitDCNumber string
	TransitDCID     int
	ChallanDate     string
	VehicleNumber   string
	ProjectID       int
}

// DestinationDCRow holds one DC for a destination drill-down.
type DestinationDCRow struct {
	DCID        int
	DCNumber    string
	ChallanDate string
	Status      string
	TotalItems  int
	ProjectID   int
}

// toNullTime converts a *time.Time to sql.NullTime for use with sqlc param types.
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// dateFilterSQL returns SQL clause and args for date range filtering.
func dateFilterSQL(startDate, endDate *time.Time, args []interface{}) (string, []interface{}) {
	clause := ""
	if startDate != nil {
		clause += " AND dc.challan_date >= ?"
		args = append(args, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		clause += " AND dc.challan_date <= ?"
		args = append(args, endDate.Format("2006-01-02"))
	}
	return clause, args
}

// summaryCountSQL executes a COUNT query with the given WHERE fragments and scans into dst.
func summaryCountSQL(query string, args []interface{}, dst *int) error {
	return DB.QueryRow(query, args...).Scan(dst)
}

// GetDCSummaryReport returns aggregate stats for the DC summary report.
// Uses sqlc param types for the both-dates case (documenting the sqlc API surface),
// but falls back to direct hand-written SQL because the generated SQL constants are
// incomplete in the current sqlc output.
func GetDCSummaryReport(projectID int, startDate, endDate *time.Time) (*DCSummaryReport, error) {
	// Reference sqlc param types to keep the import live and document intent.
	// These are used only as documentation; the actual queries are hand-written
	// because the sqlc-generated SQL constants are truncated.
	_ = db.GetDCSummaryTransitDraftFilteredParams{}

	report := &DCSummaryReport{}

	args := []interface{}{projectID}
	dateClause, args := dateFilterSQL(startDate, endDate, args)

	// Transit draft
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans dc WHERE dc.project_id = ? AND dc.dc_type='transit' AND dc.status='draft'"+dateClause, args...,
	).Scan(&report.TransitDraftDCs)
	if err != nil {
		return nil, err
	}

	// Transit issued
	args2 := []interface{}{projectID}
	dateClause2, args2 := dateFilterSQL(startDate, endDate, args2)
	err = DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans dc WHERE dc.project_id = ? AND dc.dc_type='transit' AND dc.status='issued'"+dateClause2, args2...,
	).Scan(&report.TransitIssuedDCs)
	if err != nil {
		return nil, err
	}

	// Official draft
	args3 := []interface{}{projectID}
	dateClause3, args3 := dateFilterSQL(startDate, endDate, args3)
	err = DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans dc WHERE dc.project_id = ? AND dc.dc_type='official' AND dc.status='draft'"+dateClause3, args3...,
	).Scan(&report.OfficialDraftDCs)
	if err != nil {
		return nil, err
	}

	// Official issued
	args4 := []interface{}{projectID}
	dateClause4, args4 := dateFilterSQL(startDate, endDate, args4)
	err = DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans dc WHERE dc.project_id = ? AND dc.dc_type='official' AND dc.status='issued'"+dateClause4, args4...,
	).Scan(&report.OfficialIssuedDCs)
	if err != nil {
		return nil, err
	}

	// Total items dispatched (sum of quantities from issued DCs)
	args5 := []interface{}{projectID}
	dateClause5, args5 := dateFilterSQL(startDate, endDate, args5)
	err = DB.QueryRow(
		`SELECT COALESCE(SUM(li.quantity), 0) FROM dc_line_items li
		 INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		 WHERE dc.project_id = ? AND dc.status='issued'`+dateClause5, args5...,
	).Scan(&report.TotalItemsDispatched)
	if err != nil {
		return nil, err
	}

	// Total serial numbers used
	args6 := []interface{}{projectID}
	dateClause6, args6 := dateFilterSQL(startDate, endDate, args6)
	err = DB.QueryRow(
		`SELECT COUNT(*) FROM serial_numbers sn
		 INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		 INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		 WHERE dc.project_id = ?`+dateClause6, args6...,
	).Scan(&report.TotalSerialsUsed)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// GetDestinationReport returns destination-wise report rows.
func GetDestinationReport(projectID int, startDate, endDate *time.Time) ([]DestinationRow, error) {
	args := []interface{}{projectID}
	dateClause, args := dateFilterSQL(startDate, endDate, args)

	rows, err := DB.Query(`
		SELECT
			COALESCE(a.district_name, 'Unknown') AS district,
			COALESCE(a.mandal_name, 'Unknown') AS mandal,
			COUNT(CASE WHEN dc.dc_type='official' THEN 1 END) AS official_dcs,
			COALESCE(SUM(li_counts.total_qty), 0) AS total_items,
			COUNT(CASE WHEN dc.dc_type='official' AND dc.status='draft' THEN 1 END) AS draft_count,
			COUNT(CASE WHEN dc.dc_type='official' AND dc.status='issued' THEN 1 END) AS issued_count
		FROM delivery_challans dc
		LEFT JOIN addresses a ON dc.ship_to_address_id = a.id
		LEFT JOIN (
			SELECT dc_id, SUM(quantity) AS total_qty FROM dc_line_items GROUP BY dc_id
		) li_counts ON li_counts.dc_id = dc.id
		WHERE dc.project_id = ?`+dateClause+`
		GROUP BY COALESCE(a.district_name, 'Unknown'), COALESCE(a.mandal_name, 'Unknown')
		ORDER BY district, mandal
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DestinationRow
	for rows.Next() {
		var r DestinationRow
		if err := rows.Scan(&r.District, &r.Mandal, &r.OfficialDCs, &r.TotalItems, &r.DraftCount, &r.IssuedCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// GetDestinationDCs returns official DCs for a specific district/mandal (drill-down).
func GetDestinationDCs(projectID int, district, mandal string, startDate, endDate *time.Time) ([]DestinationDCRow, error) {
	args := []interface{}{projectID, district, mandal}
	dateClause, args := dateFilterSQL(startDate, endDate, args)

	rows, err := DB.Query(`
		SELECT
			dc.id, dc.dc_number, COALESCE(dc.challan_date, ''), dc.status,
			COALESCE(SUM(li.quantity), 0) AS total_items,
			dc.project_id
		FROM delivery_challans dc
		LEFT JOIN addresses a ON dc.ship_to_address_id = a.id
		LEFT JOIN dc_line_items li ON li.dc_id = dc.id
		WHERE dc.project_id = ? AND dc.dc_type='official'
			AND COALESCE(a.district_name, 'Unknown') = ?
			AND COALESCE(a.mandal_name, 'Unknown') = ?`+dateClause+`
		GROUP BY dc.id
		ORDER BY dc.challan_date DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DestinationDCRow
	for rows.Next() {
		var r DestinationDCRow
		if err := rows.Scan(&r.DCID, &r.DCNumber, &r.ChallanDate, &r.Status, &r.TotalItems, &r.ProjectID); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// GetProductReport returns product-wise report rows.
func GetProductReport(projectID int, startDate, endDate *time.Time) ([]ProductReportRow, error) {
	args := []interface{}{projectID}
	dateClause, args := dateFilterSQL(startDate, endDate, args)

	rows, err := DB.Query(`
		SELECT
			COALESCE(p.item_name, 'Unknown') AS product_name,
			COALESCE(SUM(li.quantity), 0) AS total_qty,
			COUNT(DISTINCT dc.id) AS dc_count,
			COUNT(DISTINCT COALESCE(a.district_name, '') || '|' || COALESCE(a.mandal_name, '')) AS destination_count
		FROM dc_line_items li
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		LEFT JOIN products p ON li.product_id = p.id
		LEFT JOIN addresses a ON dc.ship_to_address_id = a.id
		WHERE dc.project_id = ?`+dateClause+`
		GROUP BY li.product_id
		ORDER BY total_qty DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProductReportRow
	for rows.Next() {
		var r ProductReportRow
		if err := rows.Scan(&r.ProductName, &r.TotalQty, &r.DCCount, &r.DestinationCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// GetSerialReport returns serial number report rows with optional search.
func GetSerialReport(projectID int, search string, startDate, endDate *time.Time) ([]SerialReportRow, error) {
	args := []interface{}{projectID}
	dateClause, args := dateFilterSQL(startDate, endDate, args)

	searchClause := ""
	if search != "" {
		searchTerms := strings.Split(search, ",")
		placeholders := make([]string, len(searchTerms))
		for i, term := range searchTerms {
			placeholders[i] = "?"
			args = append(args, "%"+strings.TrimSpace(term)+"%")
		}
		if len(placeholders) == 1 {
			searchClause = " AND sn.serial_number LIKE ?"
		} else {
			clauses := make([]string, len(placeholders))
			for i := range placeholders {
				clauses[i] = "sn.serial_number LIKE ?"
			}
			searchClause = " AND (" + strings.Join(clauses, " OR ") + ")"
		}
	}

	rows, err := DB.Query(`
		SELECT
			sn.serial_number,
			COALESCE(p.item_name, 'Unknown') AS product_name,
			dc.dc_number,
			dc.id AS dc_id,
			COALESCE(dc.challan_date, '') AS challan_date,
			COALESCE(td.vehicle_number, '') AS vehicle_number,
			dc.project_id
		FROM serial_numbers sn
		INNER JOIN dc_line_items li ON sn.line_item_id = li.id
		INNER JOIN delivery_challans dc ON li.dc_id = dc.id
		LEFT JOIN products p ON li.product_id = p.id
		LEFT JOIN dc_transit_details td ON td.dc_id = dc.id
		WHERE dc.project_id = ?`+dateClause+searchClause+`
		ORDER BY sn.serial_number
		LIMIT 500
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("serial report query: %w", err)
	}
	defer rows.Close()

	var results []SerialReportRow
	for rows.Next() {
		var r SerialReportRow
		if err := rows.Scan(&r.SerialNumber, &r.ProductName, &r.TransitDCNumber, &r.TransitDCID, &r.ChallanDate, &r.VehicleNumber, &r.ProjectID); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// Ensure the sqlc package import is used — reference the context and db packages.
var _ = context.Background
var _ = toNullTime
