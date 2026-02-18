package database

import (
	"database/sql"
	"time"
)

// DashboardStats holds aggregate statistics for the project dashboard
type DashboardStats struct {
	// Entity counts
	TotalProducts        int
	TotalTemplates       int
	TotalBillToAddresses int
	TotalShipToAddresses int
	// DC counts
	TotalDCs     int
	TransitDCs   int
	OfficialDCs  int
	IssuedDCs    int
	DraftDCs     int
	DCsThisMonth int

	// Breakdown by type+status
	TransitDCsDraft   int
	TransitDCsIssued  int
	OfficialDCsDraft  int
	OfficialDCsIssued int

	// Serial numbers
	TotalSerialNumbers int
}

// RecentActivity represents a recent action for the dashboard activity feed
type RecentActivity struct {
	ID          int
	EntityType  string // "dc", "product", "address"
	EntityID    int
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	ProjectID   int
}

// RecentDC represents a recent delivery challan for the dashboard list
type RecentDC struct {
	ID            int
	DCNumber      string
	DCType        string
	ProjectName   string
	ProjectID     int
	ChallanDate   string
	Status        string
	ShipToSummary string
	CreatedAt     string
}

// GetDashboardStats returns aggregate dashboard statistics for a project with optional date filtering.
// Hand-written SQL: GetDashboardStats supports single-bound date ranges (start-only or end-only)
// which cannot be expressed as static sqlc queries; sqlc provides only no-filter and both-bounds variants.
func GetDashboardStats(projectID int, startDate, endDate *time.Time) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Build date filter clause for DCs
	dateFilter := ""
	var dateArgs []interface{}
	if startDate != nil {
		dateFilter += " AND challan_date >= ?"
		dateArgs = append(dateArgs, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		dateFilter += " AND challan_date <= ?"
		dateArgs = append(dateArgs, endDate.Format("2006-01-02"))
	}

	// --- Entity counts (never date-filtered) ---

	_ = DB.QueryRow("SELECT COUNT(*) FROM products WHERE project_id = ?", projectID).Scan(&stats.TotalProducts)
	_ = DB.QueryRow("SELECT COUNT(*) FROM dc_templates WHERE project_id = ?", projectID).Scan(&stats.TotalTemplates)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM addresses a
		JOIN address_list_configs c ON a.config_id = c.id
		WHERE c.project_id = ? AND c.address_type = 'bill_to'`, projectID).Scan(&stats.TotalBillToAddresses)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM addresses a
		JOIN address_list_configs c ON a.config_id = c.id
		WHERE c.project_id = ? AND c.address_type = 'ship_to'`, projectID).Scan(&stats.TotalShipToAddresses)

	// --- DC counts (with optional date filter) ---
	buildArgs := func(extra ...interface{}) []interface{} {
		args := []interface{}{projectID}
		args = append(args, extra...)
		args = append(args, dateArgs...)
		return args
	}

	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ?"+dateFilter, buildArgs()...).Scan(&stats.TotalDCs)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='transit'"+dateFilter, buildArgs()...).Scan(&stats.TransitDCs)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='official'"+dateFilter, buildArgs()...).Scan(&stats.OfficialDCs)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status='issued'"+dateFilter, buildArgs()...).Scan(&stats.IssuedDCs)
	// DraftDCs is intentionally not date-filtered (matches original behavior).
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND status='draft'", projectID).Scan(&stats.DraftDCs)

	// Breakdown by type+status
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='transit' AND status='draft'"+dateFilter, buildArgs()...).Scan(&stats.TransitDCsDraft)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='transit' AND status='issued'"+dateFilter, buildArgs()...).Scan(&stats.TransitDCsIssued)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='official' AND status='draft'"+dateFilter, buildArgs()...).Scan(&stats.OfficialDCsDraft)
	_ = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND dc_type='official' AND status='issued'"+dateFilter, buildArgs()...).Scan(&stats.OfficialDCsIssued)

	// DCs this month (bounds computed in Go)
	now := time.Now()
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	_ = DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE project_id = ? AND challan_date >= ? AND challan_date <= ?",
		projectID, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"),
	).Scan(&stats.DCsThisMonth)

	// Total serial numbers (no date filter)
	_ = DB.QueryRow("SELECT COUNT(*) FROM serial_numbers WHERE project_id = ?", projectID).Scan(&stats.TotalSerialNumbers)

	return stats, nil
}

// GetRecentDCs returns the most recent delivery challans for a project.
// Hand-written SQL: sqlc GetRecentDCsRow has ChallanDate/CreatedAt as time.Time, but these
// columns are stored as TEXT and may be empty strings; scanning into string is safer.
func GetRecentDCs(projectID int, limit int) ([]RecentDC, error) {
	rows, err := DB.Query(`
		SELECT
			dc.id, dc.dc_number, dc.dc_type,
			p.name, dc.project_id,
			COALESCE(dc.challan_date, ''),
			dc.status,
			COALESCE(dc.created_at, '')
		FROM delivery_challans dc
		LEFT JOIN projects p ON dc.project_id = p.id
		WHERE dc.project_id = ?
		ORDER BY dc.created_at DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RecentDC
	for rows.Next() {
		var dc RecentDC
		var projectName sql.NullString
		err := rows.Scan(&dc.ID, &dc.DCNumber, &dc.DCType, &projectName, &dc.ProjectID, &dc.ChallanDate, &dc.Status, &dc.CreatedAt)
		if err != nil {
			return nil, err
		}
		dc.ProjectName = projectName.String
		results = append(results, dc)
	}
	return results, nil
}

// GetRecentActivity returns the most recent activity items for a project dashboard.
// Hand-written SQL: sqlc GetRecentActivityRow.Description is interface{} (computed column);
// scanning created_at as string is safer given the TEXT-stored date format.
func GetRecentActivity(projectID int, limit int) ([]RecentActivity, error) {
	rows, err := DB.Query(`
		SELECT id, 'dc' as entity_type, id as entity_id,
			dc_number as title,
			dc_type || ' DC ' || status as description,
			status,
			created_at,
			project_id
		FROM delivery_challans
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []RecentActivity
	for rows.Next() {
		var a RecentActivity
		var createdStr string
		err := rows.Scan(&a.ID, &a.EntityType, &a.EntityID, &a.Title, &a.Description, &a.Status, &createdStr, &a.ProjectID)
		if err != nil {
			return nil, err
		}
		if t, err := time.Parse("2006-01-02 15:04:05", createdStr); err == nil {
			a.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			a.CreatedAt = t
		}
		activities = append(activities, a)
	}
	return activities, nil
}
