package database

import (
	"database/sql"
	"time"
)

// DashboardStats holds aggregate statistics for the dashboard
type DashboardStats struct {
	TotalProjects int
	TotalDCs      int
	TransitDCs    int
	OfficialDCs   int
	IssuedDCs     int
	DraftDCs      int
	DCsThisMonth  int
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

// ProjectDCCount holds per-project DC breakdown
type ProjectDCCount struct {
	ProjectID   int
	ProjectName string
	TransitDCs  int
	OfficialDCs int
	TotalDCs    int
}

// GetDashboardStats returns aggregate dashboard statistics with optional date filtering
func GetDashboardStats(startDate, endDate *time.Time) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Total projects (always unfiltered)
	err := DB.QueryRow("SELECT COUNT(*) FROM projects").Scan(&stats.TotalProjects)
	if err != nil {
		return nil, err
	}

	// Build date filter clause
	dateFilter := ""
	var args []interface{}
	if startDate != nil {
		dateFilter += " AND challan_date >= ?"
		args = append(args, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		dateFilter += " AND challan_date <= ?"
		args = append(args, endDate.Format("2006-01-02"))
	}

	// Total DCs (with date filter)
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE 1=1"+dateFilter, args...).Scan(&stats.TotalDCs)
	if err != nil {
		return nil, err
	}

	// Transit DCs
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE dc_type='transit'"+dateFilter, args...).Scan(&stats.TransitDCs)
	if err != nil {
		return nil, err
	}

	// Official DCs
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE dc_type='official'"+dateFilter, args...).Scan(&stats.OfficialDCs)
	if err != nil {
		return nil, err
	}

	// Issued DCs
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE status='issued'"+dateFilter, args...).Scan(&stats.IssuedDCs)
	if err != nil {
		return nil, err
	}

	// Draft DCs (always unfiltered - global count)
	err = DB.QueryRow("SELECT COUNT(*) FROM delivery_challans WHERE status='draft'").Scan(&stats.DraftDCs)
	if err != nil {
		return nil, err
	}

	// DCs this month (always current month, unfiltered)
	now := time.Now()
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	err = DB.QueryRow(
		"SELECT COUNT(*) FROM delivery_challans WHERE challan_date >= ? AND challan_date <= ?",
		firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"),
	).Scan(&stats.DCsThisMonth)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRecentDCs returns the most recent delivery challans
func GetRecentDCs(limit int) ([]RecentDC, error) {
	rows, err := DB.Query(`
		SELECT
			dc.id, dc.dc_number, dc.dc_type,
			p.name, dc.project_id,
			COALESCE(dc.challan_date, ''),
			dc.status,
			COALESCE(dc.created_at, '')
		FROM delivery_challans dc
		LEFT JOIN projects p ON dc.project_id = p.id
		ORDER BY dc.created_at DESC
		LIMIT ?
	`, limit)
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

// GetProjectDCCounts returns per-project DC breakdown
func GetProjectDCCounts() ([]ProjectDCCount, error) {
	rows, err := DB.Query(`
		SELECT
			p.id, p.name,
			SUM(CASE WHEN dc.dc_type = 'transit' THEN 1 ELSE 0 END),
			SUM(CASE WHEN dc.dc_type = 'official' THEN 1 ELSE 0 END),
			COUNT(dc.id)
		FROM projects p
		LEFT JOIN delivery_challans dc ON dc.project_id = p.id
		GROUP BY p.id, p.name
		ORDER BY COUNT(dc.id) DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProjectDCCount
	for rows.Next() {
		var pc ProjectDCCount
		err := rows.Scan(&pc.ProjectID, &pc.ProjectName, &pc.TransitDCs, &pc.OfficialDCs, &pc.TotalDCs)
		if err != nil {
			return nil, err
		}
		results = append(results, pc)
	}
	return results, nil
}
