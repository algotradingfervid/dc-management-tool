package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// DC type constants.
const (
	DCTypeTransit  = "transit"
	DCTypeOfficial = "official"
)

// dcTypeCode maps DC type to its code in the DC number.
var dcTypeCode = map[string]string{
	DCTypeTransit:  "TDC",
	DCTypeOfficial: "ODC",
}

// dcCodeToType maps DC number code back to DC type.
var dcCodeToType = map[string]string{
	"TDC": DCTypeTransit,
	"ODC": DCTypeOfficial,
}

// dcNumberPattern validates the DC number format: PREFIX-TDC-2526-001
var dcNumberPattern = regexp.MustCompile(`^[A-Za-z0-9/]+-(TDC|ODC)-\d{4}-\d{3,}$`)

// DCNumberParts represents the parsed components of a DC number.
type DCNumberParts struct {
	Prefix         string
	FinancialYear  string
	DCType         string
	SequenceNumber int
}

// PeekNextDCNumber returns what the next DC number would be WITHOUT incrementing the sequence.
// Use this for display purposes (e.g., showing the number on the create form).
func PeekNextDCNumber(db *sql.DB, projectID int, dcType string) (string, error) {
	if dcType != DCTypeTransit && dcType != DCTypeOfficial {
		return "", fmt.Errorf("invalid DC type: %s (must be 'transit' or 'official')", dcType)
	}

	var dcPrefix string
	err := db.QueryRow("SELECT dc_prefix FROM projects WHERE id = ?", projectID).Scan(&dcPrefix)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("project not found: %d", projectID)
		}
		return "", fmt.Errorf("failed to get project prefix: %w", err)
	}
	if dcPrefix == "" {
		return "", fmt.Errorf("project %d has no DC prefix set", projectID)
	}

	fy := GetFinancialYear(time.Now())

	var nextSeq int
	err = db.QueryRow(`
		SELECT next_sequence FROM dc_number_sequences
		WHERE project_id = ? AND dc_type = ? AND financial_year = ?`,
		projectID, dcType, fy,
	).Scan(&nextSeq)
	if err == sql.ErrNoRows {
		nextSeq = 1
	} else if err != nil {
		return "", fmt.Errorf("failed to read sequence: %w", err)
	}

	return FormatDCNumber(dcPrefix, fy, dcType, nextSeq), nil
}

// GenerateDCNumber generates a unique DC number for a delivery challan.
// Format: {Prefix}-{TDC|ODC}-{YYYYYY}-{NNN}
// Example: SCP-TDC-2425-001
// WARNING: This increments the sequence. Only call when actually creating a DC.
func GenerateDCNumber(db *sql.DB, projectID int, dcType string) (string, error) {
	return GenerateDCNumberForDate(db, projectID, dcType, time.Now())
}

// GenerateDCNumberForDate generates a DC number using a specific date for FY calculation.
// This is useful for testing and for generating DCs with a specific date.
func GenerateDCNumberForDate(db *sql.DB, projectID int, dcType string, date time.Time) (string, error) {
	if dcType != DCTypeTransit && dcType != DCTypeOfficial {
		return "", fmt.Errorf("invalid DC type: %s (must be 'transit' or 'official')", dcType)
	}

	// Use BEGIN IMMEDIATE for SQLite write transaction to prevent SQLITE_BUSY
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Force immediate write lock
	if _, err := tx.Exec("SELECT 1 FROM dc_number_sequences LIMIT 0"); err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Get project DC prefix
	var dcPrefix string
	err = tx.QueryRow("SELECT dc_prefix FROM projects WHERE id = ?", projectID).Scan(&dcPrefix)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("project not found: %d", projectID)
		}
		return "", fmt.Errorf("failed to get project prefix: %w", err)
	}

	if dcPrefix == "" {
		return "", fmt.Errorf("project %d has no DC prefix set", projectID)
	}

	fy := GetFinancialYear(date)

	sequence, err := getNextSequence(tx, projectID, dcType, fy)
	if err != nil {
		return "", fmt.Errorf("failed to get next sequence: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return FormatDCNumber(dcPrefix, fy, dcType, sequence), nil
}

// getNextSequence retrieves and increments the sequence number atomically within a transaction.
func getNextSequence(tx *sql.Tx, projectID int, dcType, financialYear string) (int, error) {
	// Use INSERT ... ON CONFLICT to atomically get-and-increment
	_, err := tx.Exec(`
		INSERT INTO dc_number_sequences (project_id, dc_type, financial_year, next_sequence)
		VALUES (?, ?, ?, 2)
		ON CONFLICT (project_id, dc_type, financial_year)
		DO UPDATE SET next_sequence = next_sequence + 1, updated_at = CURRENT_TIMESTAMP`,
		projectID, dcType, financialYear,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert sequence: %w", err)
	}

	// Read the current value (which is next_sequence - 1 for existing, or 1 for new)
	var nextSeq int
	err = tx.QueryRow(`
		SELECT next_sequence - 1 FROM dc_number_sequences
		WHERE project_id = ? AND dc_type = ? AND financial_year = ?`,
		projectID, dcType, financialYear,
	).Scan(&nextSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to read sequence: %w", err)
	}

	// For a brand new row, we inserted next_sequence=2, so next_sequence-1=1. Correct.
	// For an existing row, we incremented, so next_sequence-1 = the value we want. Correct.
	return nextSeq, nil
}

// FormatDCNumber formats a DC number from its components.
func FormatDCNumber(prefix, financialYear, dcType string, sequence int) string {
	code := dcTypeCode[dcType]
	return fmt.Sprintf("%s-%s-%s-%03d", prefix, code, financialYear, sequence)
}

// ParseDCNumber parses a DC number string into its components.
func ParseDCNumber(dcNumber string) (*DCNumberParts, error) {
	if !dcNumberPattern.MatchString(dcNumber) {
		return nil, fmt.Errorf("invalid DC number format: %s", dcNumber)
	}

	// Find the last 3 segments by splitting from the right
	// Format: PREFIX-TDC-2526-001 (prefix may contain hyphens or slashes)
	// We know the last 3 segments are: type code, FY, sequence
	parts := splitFromRight(dcNumber, "-", 3)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid DC number format: %s", dcNumber)
	}

	seq, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid sequence number in DC number: %s", dcNumber)
	}

	dcType, ok := dcCodeToType[parts[1]]
	if !ok {
		return nil, fmt.Errorf("invalid DC type code in DC number: %s", parts[1])
	}

	return &DCNumberParts{
		Prefix:         parts[0],
		FinancialYear:  parts[2],
		DCType:         dcType,
		SequenceNumber: seq,
	}, nil
}

// splitFromRight splits a string by separator, taking the last n segments
// and joining everything before them as the first element.
func splitFromRight(s, sep string, n int) []string {
	// Find positions of all separators
	var positions []int
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			positions = append(positions, i)
		}
	}

	if len(positions) < n {
		return nil
	}

	splitPos := positions[len(positions)-n]
	result := []string{s[:splitPos]}
	remaining := s[splitPos+1:]

	for i := 0; i < n-1; i++ {
		idx := 0
		for idx < len(remaining) && remaining[idx] != sep[0] {
			idx++
		}
		result = append(result, remaining[:idx])
		if idx < len(remaining) {
			remaining = remaining[idx+1:]
		}
	}
	result = append(result, remaining)

	return result
}

// IsValidDCNumber checks if a string is a valid DC number format.
func IsValidDCNumber(dcNumber string) bool {
	_, err := ParseDCNumber(dcNumber)
	return err == nil
}
