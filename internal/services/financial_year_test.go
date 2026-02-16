package services

import (
	"testing"
	"time"
)

func TestGetFinancialYear(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{"January - belongs to prev FY", time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), "2526"},
		{"February", time.Date(2026, time.February, 16, 0, 0, 0, 0, time.UTC), "2526"},
		{"March 31 - last day of FY", time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC), "2526"},
		{"April 1 - first day of new FY", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), "2627"},
		{"June", time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC), "2627"},
		{"December", time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC), "2627"},
		{"January 2027", time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC), "2627"},
		{"Year 2024 April", time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC), "2425"},
		{"Year 2025 March", time.Date(2025, time.March, 31, 0, 0, 0, 0, time.UTC), "2425"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFinancialYear(tt.date)
			if result != tt.expected {
				t.Errorf("GetFinancialYear(%v) = %s; want %s", tt.date, result, tt.expected)
			}
		})
	}
}

func TestGetFinancialYearAllMonths(t *testing.T) {
	// Test every month of 2026 to ensure correct FY assignment
	expectations := map[time.Month]string{
		time.January:   "2526",
		time.February:  "2526",
		time.March:     "2526",
		time.April:     "2627",
		time.May:       "2627",
		time.June:      "2627",
		time.July:      "2627",
		time.August:    "2627",
		time.September: "2627",
		time.October:   "2627",
		time.November:  "2627",
		time.December:  "2627",
	}

	for month, expected := range expectations {
		date := time.Date(2026, month, 15, 0, 0, 0, 0, time.UTC)
		result := GetFinancialYear(date)
		if result != expected {
			t.Errorf("Month %s: GetFinancialYear(%v) = %s; want %s", month, date, result, expected)
		}
	}
}

func TestGetFinancialYearBoundary(t *testing.T) {
	// Exact boundary: March 31 23:59:59 vs April 1 00:00:00
	march31 := time.Date(2026, time.March, 31, 23, 59, 59, 999999999, time.UTC)
	april1 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	if fy := GetFinancialYear(march31); fy != "2526" {
		t.Errorf("March 31 should be FY 2526, got %s", fy)
	}
	if fy := GetFinancialYear(april1); fy != "2627" {
		t.Errorf("April 1 should be FY 2627, got %s", fy)
	}
}

func TestGetFinancialYearStart(t *testing.T) {
	start := GetFinancialYearStart(2025)
	expected := time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("GetFinancialYearStart(2025) = %v; want %v", start, expected)
	}
}

func TestGetFinancialYearEnd(t *testing.T) {
	end := GetFinancialYearEnd(2025)
	expected := time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC)
	if !end.Equal(expected) {
		t.Errorf("GetFinancialYearEnd(2025) = %v; want %v", end, expected)
	}
}

func TestParseFinancialYear(t *testing.T) {
	tests := []struct {
		name        string
		fy          string
		wantStart   int
		wantEnd     int
		expectError bool
	}{
		{"Valid 2526", "2526", 2025, 2026, false},
		{"Valid 2425", "2425", 2024, 2025, false},
		{"Valid 0001", "0001", 2000, 2001, false},
		{"Invalid - too short", "25", 0, 0, true},
		{"Invalid - too long", "25260", 0, 0, true},
		{"Invalid - non-consecutive", "2527", 0, 0, true},
		{"Invalid - letters", "abcd", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseFinancialYear(tt.fy)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for %s, got none", tt.fy)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tt.fy, err)
				return
			}
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("ParseFinancialYear(%s) = (%d, %d); want (%d, %d)", tt.fy, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
