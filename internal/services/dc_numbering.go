package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

// FormatConfig holds the configurable DC number format settings from a project.
type FormatConfig struct {
	Format    string // e.g. "{PREFIX}/{PROJECT_CODE}/{FY}/{SEQ}" or "{PREFIX}-{TYPE}-{FY}-{SEQ}"
	Separator string // legacy, not used with token-based format
	Padding   int    // zero-padding for sequence number (default 3)
}

// DefaultFormatConfig returns the default format configuration.
func DefaultFormatConfig() FormatConfig {
	return FormatConfig{
		Format:  "{PREFIX}-{TYPE}-{FY}-{SEQ}",
		Padding: 3,
	}
}

// FormatDCNumberConfigurable formats a DC number using a configurable format pattern.
func FormatDCNumberConfigurable(format, prefix, projectCode, fy, dcType string, sequence, padding int) string {
	if format == "" {
		format = "{PREFIX}-{TYPE}-{FY}-{SEQ}"
	}
	if padding < 1 {
		padding = 3
	}

	code := dcTypeCode[dcType]
	seqStr := fmt.Sprintf("%0*d", padding, sequence)
	fyFormatted := fmt.Sprintf("%s-%s", fy[:2], fy[2:])

	r := strings.NewReplacer(
		"{PREFIX}", prefix,
		"{PROJECT_CODE}", projectCode,
		"{FY}", fyFormatted,
		"{SEQ}", seqStr,
		"{TYPE}", code,
	)
	return r.Replace(format)
}

// PreviewDCNumber generates a preview of what a DC number would look like.
func PreviewDCNumber(format, prefix, projectCode string, padding int) string {
	if format == "" {
		format = "{PREFIX}-{TYPE}-{FY}-{SEQ}"
	}
	if padding < 1 {
		padding = 3
	}
	fy := GetFinancialYear(time.Now())
	return FormatDCNumberConfigurable(format, prefix, projectCode, fy, DCTypeTransit, 1, padding)
}

// PeekNextDCNumber returns what the next DC number would be WITHOUT incrementing the sequence.
func PeekNextDCNumber(db *sql.DB, projectID int, dcType string) (string, error) {
	if dcType != DCTypeTransit && dcType != DCTypeOfficial {
		return "", fmt.Errorf("invalid DC type: %s (must be 'transit' or 'official')", dcType)
	}

	var dcPrefix, dcNumberFormat string
	var seqPadding int
	err := db.QueryRow("SELECT dc_prefix, dc_number_format, seq_padding FROM projects WHERE id = ?", projectID).Scan(&dcPrefix, &dcNumberFormat, &seqPadding)
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

	// Use configurable format if set, otherwise default
	if dcNumberFormat != "" && dcNumberFormat != "{PREFIX}-{TYPE}-{FY}-{SEQ}" {
		return FormatDCNumberConfigurable(dcNumberFormat, dcPrefix, dcPrefix, fy, dcType, nextSeq, seqPadding), nil
	}

	return FormatDCNumber(dcPrefix, fy, dcType, nextSeq), nil
}

// GenerateDCNumber generates a unique DC number for a delivery challan.
func GenerateDCNumber(db *sql.DB, projectID int, dcType string) (string, error) {
	return GenerateDCNumberForDate(db, projectID, dcType, time.Now())
}

// GenerateDCNumberForDate generates a DC number using a specific date for FY calculation.
func GenerateDCNumberForDate(db *sql.DB, projectID int, dcType string, date time.Time) (string, error) {
	if dcType != DCTypeTransit && dcType != DCTypeOfficial {
		return "", fmt.Errorf("invalid DC type: %s (must be 'transit' or 'official')", dcType)
	}

	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT 1 FROM dc_number_sequences LIMIT 0"); err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	var dcPrefix, dcNumberFormat string
	var seqPadding int
	err = tx.QueryRow("SELECT dc_prefix, dc_number_format, seq_padding FROM projects WHERE id = ?", projectID).Scan(&dcPrefix, &dcNumberFormat, &seqPadding)
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

	if dcNumberFormat != "" && dcNumberFormat != "{PREFIX}-{TYPE}-{FY}-{SEQ}" {
		return FormatDCNumberConfigurable(dcNumberFormat, dcPrefix, dcPrefix, fy, dcType, sequence, seqPadding), nil
	}

	return FormatDCNumber(dcPrefix, fy, dcType, sequence), nil
}

// getNextSequence retrieves and increments the sequence number atomically within a transaction.
func getNextSequence(tx *sql.Tx, projectID int, dcType, financialYear string) (int, error) {
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

	var nextSeq int
	err = tx.QueryRow(`
		SELECT next_sequence - 1 FROM dc_number_sequences
		WHERE project_id = ? AND dc_type = ? AND financial_year = ?`,
		projectID, dcType, financialYear,
	).Scan(&nextSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to read sequence: %w", err)
	}

	return nextSeq, nil
}

// FormatDCNumber formats a DC number from its components (legacy default format).
func FormatDCNumber(prefix, financialYear, dcType string, sequence int) string {
	code := dcTypeCode[dcType]
	return fmt.Sprintf("%s-%s-%s-%03d", prefix, code, financialYear, sequence)
}

// ParseDCNumber parses a DC number string into its components.
func ParseDCNumber(dcNumber string) (*DCNumberParts, error) {
	if !dcNumberPattern.MatchString(dcNumber) {
		return nil, fmt.Errorf("invalid DC number format: %s", dcNumber)
	}

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
