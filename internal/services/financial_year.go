package services

import (
	"fmt"
	"time"
)

// GetFinancialYear returns the Indian financial year for a given date.
// Indian FY runs from April 1 to March 31.
// Returns compact format: "YYYYYY" (e.g., "2526" for FY 2025-26).
func GetFinancialYear(date time.Time) string {
	year := date.Year()
	month := date.Month()

	var startYear int
	if month >= time.April {
		startYear = year
	} else {
		startYear = year - 1
	}

	endYear := startYear + 1
	return fmt.Sprintf("%02d%02d", startYear%100, endYear%100)
}

// GetCurrentFinancialYear returns the financial year for the current date.
func GetCurrentFinancialYear() string {
	return GetFinancialYear(time.Now())
}

// GetFinancialYearStart returns the start date (April 1) of the given FY start year.
func GetFinancialYearStart(startYear int) time.Time {
	return time.Date(startYear, time.April, 1, 0, 0, 0, 0, time.UTC)
}

// GetFinancialYearEnd returns the end date (March 31) of the given FY start year.
func GetFinancialYearEnd(startYear int) time.Time {
	return time.Date(startYear+1, time.March, 31, 23, 59, 59, 0, time.UTC)
}

// ParseFinancialYear parses a compact FY string (e.g., "2526") into start and end years.
func ParseFinancialYear(fyString string) (startYear, endYear int, err error) {
	if len(fyString) != 4 {
		return 0, 0, fmt.Errorf("invalid financial year format: %s (expected 4 digits like '2526')", fyString)
	}

	var sy, ey int
	_, err = fmt.Sscanf(fyString, "%02d%02d", &sy, &ey)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid financial year format: %s", fyString)
	}

	startYear = 2000 + sy
	endYear = 2000 + ey

	if endYear != startYear+1 {
		return 0, 0, fmt.Errorf("invalid financial year: end year must be start year + 1, got %s", fyString)
	}

	return startYear, endYear, nil
}
