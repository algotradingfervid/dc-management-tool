# Phase 10: DC Number Generation Logic

## Implementation Status: COMPLETED

### Summary
Phase 10 has been fully implemented and tested. All 22 unit tests pass including concurrency tests.

### What was implemented:
1. **Financial Year Calculation** (`internal/services/financial_year.go`)
   - Indian FY (April-March) with compact format `YYYYYY` (e.g., "2526" for FY 2025-26)
   - Helper functions: `GetFinancialYear`, `GetCurrentFinancialYear`, `GetFinancialYearStart`, `GetFinancialYearEnd`, `ParseFinancialYear`

2. **DC Number Generation Service** (`internal/services/dc_numbering.go`)
   - Format: `{Prefix}-{TDC|ODC}-{FY}-{NNN}` (e.g., `SCP-TDC-2526-001`)
   - Thread-safe sequential numbering using SQLite transactions with `INSERT ... ON CONFLICT`
   - Independent sequences per project, per DC type (transit/official), per financial year
   - Auto-reset at FY rollover (April 1st)
   - Handles sequences beyond 999 gracefully
   - Parse/validate utilities: `ParseDCNumber`, `IsValidDCNumber`, `FormatDCNumber`

3. **Database Migration** (`migrations/000010_create_dc_number_sequences_table.up.sql`)
   - `dc_number_sequences` table with composite unique key `(project_id, dc_type, financial_year)`
   - Indexed for fast lookups

4. **Comprehensive Tests** (`internal/services/*_test.go`) - 22 tests:
   - FY calculation: all months, boundaries (Mar 31 vs Apr 1), year transitions
   - DC numbering: format, parse, validate, sequential integrity (20 sequential)
   - Concurrency: 50 goroutines generating simultaneously - all unique, no gaps
   - Edge cases: invalid type, missing project, empty prefix, sequence >999, FY rollover

### Files Created:
- `internal/services/financial_year.go`
- `internal/services/financial_year_test.go`
- `internal/services/dc_numbering.go`
- `internal/services/dc_numbering_test.go`
- `migrations/000010_create_dc_number_sequences_table.up.sql`
- `migrations/000010_create_dc_number_sequences_table.down.sql`

---

## Overview
This phase implements the DC (Delivery Challan) numbering system that generates unique, sequential DC numbers based on a specific format. The system handles Indian financial year calculations, automatic sequential numbering, and ensures thread-safe generation of DC numbers even under concurrent requests.

## Prerequisites
- Phase 1-5: Project management established
- Database: projects table with dc_prefix field
- Understanding of Indian Financial Year (April-March cycle)
- Basic understanding of database transactions and locking

## Goals
- Implement DC number format: {Project DC Prefix}/{Financial Year}/{DC Type Suffix}/{Sequential Number}
- Calculate Indian Financial Year correctly (April-March)
- Generate sequential numbers per project, per DC type, per FY
- Ensure thread-safe concurrent DC number generation
- Pad sequential numbers to 3 digits (001, 002, etc.)
- Handle financial year transitions automatically
- Reset numbering at the start of each financial year
- Prevent reuse of deleted DC numbers
- Provide utility function for DC number generation
- Add comprehensive unit tests for all edge cases

## Detailed Implementation Steps

### 1. Financial Year Calculation
1.1. Create utils/financial_year.go
   - Implement GetFinancialYear(date time.Time) string
   - Indian FY runs from April 1 to March 31
   - Format: YY-YY (e.g., "25-26" for FY 2025-26)

1.2. Handle edge cases
   - January-March: FY is previous year to current year
   - April-December: FY is current year to next year
   - Examples:
     - 2026-02-16 → "25-26"
     - 2026-04-01 → "26-27"
     - 2026-03-31 → "25-26"
     - 2027-01-15 → "26-27"

1.3. Add helper functions
   - GetCurrentFinancialYear() string
   - GetFinancialYearStart(year int) time.Time
   - GetFinancialYearEnd(year int) time.Time
   - ParseFinancialYear(fyString string) (startYear, endYear int, error)

### 2. DC Number Format Implementation
2.1. Define DC number components
   - Project DC Prefix (from projects.dc_prefix)
   - Financial Year (YY-YY format)
   - DC Type Suffix (T for Transit, D for Official)
   - Sequential Number (001, 002, 003...)

2.2. Format template
   - {Prefix}/{FY}/{Type}/{Seq}
   - Example: FS/GS/25-26/T/005
   - Example: PWD/AP/25-26/D/012

2.3. Create utils/dc_numbering.go
   - Define constants for DC types
   - Implement format string generation
   - Implement number padding logic

### 3. Sequential Number Management
3.1. Create dc_number_sequences table
   - Track next sequence number per project/type/FY
   - Composite unique key prevents duplicates
   - Supports concurrent access

3.2. Implement GetNextSequenceNumber function
   - Fetch current sequence for project/type/FY
   - Increment sequence
   - Return new sequence number
   - Use database transaction for atomicity

3.3. Handle first-time generation
   - Initialize sequence to 1 for new project/type/FY combination
   - INSERT OR UPDATE logic

### 4. DC Number Generator
4.1. Create GenerateDCNumber function signature
   ```go
   GenerateDCNumber(db *sql.DB, projectID int, dcType string) (string, error)
   ```

4.2. Implementation steps
   - Begin database transaction
   - Get project DC prefix from projects table
   - Calculate current financial year
   - Get next sequence number (with row locking)
   - Format DC number string
   - Commit transaction
   - Return formatted DC number

4.3. Error handling
   - Project not found
   - Invalid DC type
   - Database transaction failures
   - Concurrent access conflicts

### 5. Thread-Safe Concurrent Access
5.1. Use database transactions
   - BEGIN IMMEDIATE for SQLite
   - Row-level locking where possible
   - Serialize sequence generation

5.2. Implement retry logic
   - Handle database locked errors
   - Exponential backoff
   - Maximum retry attempts

5.3. Testing concurrent generation
   - Simulate multiple simultaneous DC creations
   - Verify no duplicate numbers generated
   - Verify sequential integrity

### 6. Database Schema
6.1. Create dc_number_sequences table
   - Columns: project_id, dc_type, financial_year, next_sequence
   - Composite unique key on (project_id, dc_type, financial_year)
   - Index for fast lookups

6.2. Add dc_number column to delivery_challans table
   - Store generated DC number
   - Unique constraint to prevent duplicates
   - Not nullable

6.3. Migration for existing projects
   - Backfill sequences if implementing in existing system

### 7. Validation and Constraints
7.1. DC Type validation
   - Only allow 'transit' or 'official'
   - Constants defined in code

7.2. Project validation
   - Ensure project exists
   - Ensure project has dc_prefix set
   - Validate dc_prefix format (no special chars)

7.3. Number format validation
   - Validate generated number matches expected format
   - Add regex validation pattern

### 8. Utilities and Helpers
8.1. Parse DC Number
   - ParseDCNumber(dcNumber string) (*DCNumberParts, error)
   - Extract prefix, FY, type, sequence
   - Validate format

8.2. Format DC Number
   - FormatDCNumber(prefix, fy, dcType string, seq int) string
   - Consistent formatting logic

8.3. Validation helpers
   - IsValidDCNumber(dcNumber string) bool
   - ValidateDCNumberFormat(dcNumber string) error

### 9. Testing Implementation
9.1. Unit tests for financial year
   - Test dates in each month
   - Test edge cases (March 31, April 1)
   - Test year boundaries

9.2. Unit tests for DC numbering
   - Test number generation
   - Test sequential incrementing
   - Test FY transition
   - Test different DC types

9.3. Concurrency tests
   - Generate 100 DC numbers concurrently
   - Verify all unique
   - Verify sequential
   - No gaps or duplicates

9.4. Integration tests
   - Full DC creation workflow
   - Cross-financial-year scenarios

### 10. Documentation
10.1. Code documentation
   - Comprehensive function comments
   - Usage examples
   - Edge case documentation

10.2. Developer guide
   - How to generate DC numbers
   - Troubleshooting guide
   - Performance considerations

## Files to Create/Modify

### New Files
```
/migrations/011_create_dc_number_sequences_table.sql
/migrations/012_add_dc_number_to_delivery_challans.sql
/utils/financial_year.go
/utils/dc_numbering.go
/utils/dc_numbering_test.go
/utils/financial_year_test.go
/models/dc_sequence.go
/docs/dc-numbering-guide.md
```

### Modified Files
```
/main.go (run new migrations)
/models/delivery_challan.go (add dc_number field - will be created in Phase 11)
```

## API Routes / Endpoints
No new API routes needed. DC number generation is an internal utility function called during DC creation (Phase 11-12).

## Database Queries

### Table Creation

#### dc_number_sequences table
```sql
CREATE TABLE dc_number_sequences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    dc_type TEXT NOT NULL CHECK(dc_type IN ('transit', 'official')),
    financial_year TEXT NOT NULL, -- Format: YY-YY (e.g., '25-26')
    next_sequence INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE (project_id, dc_type, financial_year)
);

CREATE INDEX idx_dc_sequences_lookup ON dc_number_sequences(project_id, dc_type, financial_year);
```

#### Add dc_number to delivery_challans (future table)
```sql
-- This will be part of delivery_challans table creation in Phase 11/12
-- Including here for reference
ALTER TABLE delivery_challans ADD COLUMN dc_number TEXT UNIQUE NOT NULL;
CREATE UNIQUE INDEX idx_dc_number ON delivery_challans(dc_number);
```

### Key Queries

#### Get next sequence number (thread-safe)
```sql
-- SQLite approach with immediate transaction
BEGIN IMMEDIATE TRANSACTION;

-- Try to get existing sequence
SELECT next_sequence
FROM dc_number_sequences
WHERE project_id = ? AND dc_type = ? AND financial_year = ?;

-- If exists, increment
UPDATE dc_number_sequences
SET next_sequence = next_sequence + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE project_id = ? AND dc_type = ? AND financial_year = ?;

-- Get the new value
SELECT next_sequence
FROM dc_number_sequences
WHERE project_id = ? AND dc_type = ? AND financial_year = ?;

COMMIT;

-- If doesn't exist, insert
INSERT INTO dc_number_sequences (project_id, dc_type, financial_year, next_sequence)
VALUES (?, ?, ?, 2)
ON CONFLICT (project_id, dc_type, financial_year)
DO UPDATE SET next_sequence = next_sequence + 1;

-- Return 1 for first number
```

#### Get project DC prefix
```sql
SELECT dc_prefix
FROM projects
WHERE id = ?;
```

#### Check if DC number already exists
```sql
SELECT COUNT(*)
FROM delivery_challans
WHERE dc_number = ?;
```

#### Get latest DC number for project/type/FY
```sql
SELECT dc_number
FROM delivery_challans
WHERE project_id = ?
  AND dc_type = ?
  AND financial_year = ?
ORDER BY created_at DESC
LIMIT 1;
```

## Go Code Implementation

### Financial Year Utility (utils/financial_year.go)
```go
package utils

import (
    "fmt"
    "time"
)

// GetFinancialYear returns the Indian financial year for a given date
// Indian FY runs from April 1 to March 31
// Returns format: "YY-YY" (e.g., "25-26" for FY 2025-26)
func GetFinancialYear(date time.Time) string {
    year := date.Year()
    month := date.Month()

    var startYear int
    if month >= time.April {
        // April to December: FY is current year to next year
        startYear = year
    } else {
        // January to March: FY is previous year to current year
        startYear = year - 1
    }

    endYear := startYear + 1

    return fmt.Sprintf("%02d-%02d", startYear%100, endYear%100)
}

// GetCurrentFinancialYear returns the current financial year
func GetCurrentFinancialYear() string {
    return GetFinancialYear(time.Now())
}

// GetFinancialYearStart returns the start date of a financial year
func GetFinancialYearStart(startYear int) time.Time {
    return time.Date(startYear, time.April, 1, 0, 0, 0, 0, time.UTC)
}

// GetFinancialYearEnd returns the end date of a financial year
func GetFinancialYearEnd(startYear int) time.Time {
    return time.Date(startYear+1, time.March, 31, 23, 59, 59, 0, time.UTC)
}

// ParseFinancialYear parses a financial year string (e.g., "25-26")
// and returns the start and end years
func ParseFinancialYear(fyString string) (startYear, endYear int, err error) {
    var sy, ey int
    _, err = fmt.Sscanf(fyString, "%02d-%02d", &sy, &ey)
    if err != nil {
        return 0, 0, fmt.Errorf("invalid financial year format: %s", fyString)
    }

    // Convert 2-digit years to 4-digit years
    // Assume 2000s for now (will need updating in 2100!)
    startYear = 2000 + sy
    endYear = 2000 + ey

    return startYear, endYear, nil
}
```

### DC Numbering Utility (utils/dc_numbering.go)
```go
package utils

import (
    "database/sql"
    "fmt"
    "time"
)

// DC Type constants
const (
    DCTypeTransit  = "transit"
    DCTypeOfficial = "official"
)

// DC Type suffix mapping
var dcTypeSuffix = map[string]string{
    DCTypeTransit:  "T",
    DCTypeOfficial: "D",
}

// DCNumberParts represents the components of a DC number
type DCNumberParts struct {
    Prefix         string
    FinancialYear  string
    DCType         string
    SequenceNumber int
}

// GenerateDCNumber generates a unique DC number for a delivery challan
// Format: {Prefix}/{FY}/{Type}/{Seq}
// Example: FS/GS/25-26/T/005
func GenerateDCNumber(db *sql.DB, projectID int, dcType string) (string, error) {
    // Validate DC type
    if dcType != DCTypeTransit && dcType != DCTypeOfficial {
        return "", fmt.Errorf("invalid DC type: %s", dcType)
    }

    // Begin transaction for thread-safe sequence generation
    tx, err := db.Begin()
    if err != nil {
        return "", fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get project DC prefix
    var dcPrefix string
    err = tx.QueryRow("SELECT dc_prefix FROM projects WHERE id = ?", projectID).Scan(&dcPrefix)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", fmt.Errorf("project not found: %d", projectID)
        }
        return "", fmt.Errorf("failed to get project prefix: %w", err)
    }

    // Calculate financial year
    financialYear := GetCurrentFinancialYear()

    // Get next sequence number
    sequence, err := getNextSequence(tx, projectID, dcType, financialYear)
    if err != nil {
        return "", fmt.Errorf("failed to get next sequence: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return "", fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Format DC number
    dcNumber := FormatDCNumber(dcPrefix, financialYear, dcType, sequence)

    return dcNumber, nil
}

// getNextSequence retrieves and increments the sequence number
// Must be called within a transaction
func getNextSequence(tx *sql.Tx, projectID int, dcType, financialYear string) (int, error) {
    var nextSeq int

    // Try to get existing sequence
    err := tx.QueryRow(`
        SELECT next_sequence
        FROM dc_number_sequences
        WHERE project_id = ? AND dc_type = ? AND financial_year = ?`,
        projectID, dcType, financialYear,
    ).Scan(&nextSeq)

    if err == sql.ErrNoRows {
        // First DC for this project/type/FY combination
        nextSeq = 1

        // Insert initial sequence record
        _, err = tx.Exec(`
            INSERT INTO dc_number_sequences (project_id, dc_type, financial_year, next_sequence)
            VALUES (?, ?, ?, 2)`,
            projectID, dcType, financialYear,
        )
        if err != nil {
            return 0, fmt.Errorf("failed to initialize sequence: %w", err)
        }
    } else if err != nil {
        return 0, fmt.Errorf("failed to query sequence: %w", err)
    } else {
        // Increment sequence for next use
        _, err = tx.Exec(`
            UPDATE dc_number_sequences
            SET next_sequence = next_sequence + 1,
                updated_at = CURRENT_TIMESTAMP
            WHERE project_id = ? AND dc_type = ? AND financial_year = ?`,
            projectID, dcType, financialYear,
        )
        if err != nil {
            return 0, fmt.Errorf("failed to increment sequence: %w", err)
        }
    }

    return nextSeq, nil
}

// FormatDCNumber formats a DC number from its components
func FormatDCNumber(prefix, financialYear, dcType string, sequence int) string {
    typeSuffix := dcTypeSuffix[dcType]
    return fmt.Sprintf("%s/%s/%s/%03d", prefix, financialYear, typeSuffix, sequence)
}

// ParseDCNumber parses a DC number into its components
func ParseDCNumber(dcNumber string) (*DCNumberParts, error) {
    var parts DCNumberParts
    var typeSuffix string

    _, err := fmt.Sscanf(dcNumber, "%[^/]/%[^/]/%[^/]/%d",
        &parts.Prefix,
        &parts.FinancialYear,
        &typeSuffix,
        &parts.SequenceNumber,
    )
    if err != nil {
        return nil, fmt.Errorf("invalid DC number format: %s", dcNumber)
    }

    // Convert type suffix to DC type
    switch typeSuffix {
    case "T":
        parts.DCType = DCTypeTransit
    case "D":
        parts.DCType = DCTypeOfficial
    default:
        return nil, fmt.Errorf("invalid DC type suffix: %s", typeSuffix)
    }

    return &parts, nil
}

// IsValidDCNumber validates a DC number format
func IsValidDCNumber(dcNumber string) bool {
    _, err := ParseDCNumber(dcNumber)
    return err == nil
}
```

### Unit Tests (utils/financial_year_test.go)
```go
package utils

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
        {
            name:     "February 2026",
            date:     time.Date(2026, time.February, 16, 0, 0, 0, 0, time.UTC),
            expected: "25-26",
        },
        {
            name:     "April 1st 2026 (FY start)",
            date:     time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
            expected: "26-27",
        },
        {
            name:     "March 31st 2026 (FY end)",
            date:     time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC),
            expected: "25-26",
        },
        {
            name:     "December 2026",
            date:     time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC),
            expected: "26-27",
        },
        {
            name:     "January 2027",
            date:     time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC),
            expected: "26-27",
        },
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

func TestParseFinancialYear(t *testing.T) {
    tests := []struct {
        name          string
        fyString      string
        expectedStart int
        expectedEnd   int
        expectError   bool
    }{
        {
            name:          "Valid FY 25-26",
            fyString:      "25-26",
            expectedStart: 2025,
            expectedEnd:   2026,
            expectError:   false,
        },
        {
            name:          "Valid FY 99-00",
            fyString:      "99-00",
            expectedStart: 2099,
            expectedEnd:   2100,
            expectError:   false,
        },
        {
            name:        "Invalid format",
            fyString:    "2025-2026",
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            start, end, err := ParseFinancialYear(tt.fyString)

            if tt.expectError {
                if err == nil {
                    t.Errorf("Expected error but got none")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
                return
            }

            if start != tt.expectedStart || end != tt.expectedEnd {
                t.Errorf("ParseFinancialYear(%s) = (%d, %d); want (%d, %d)",
                    tt.fyString, start, end, tt.expectedStart, tt.expectedEnd)
            }
        })
    }
}
```

### Unit Tests (utils/dc_numbering_test.go)
```go
package utils

import (
    "testing"
)

func TestFormatDCNumber(t *testing.T) {
    tests := []struct {
        name     string
        prefix   string
        fy       string
        dcType   string
        seq      int
        expected string
    }{
        {
            name:     "Transit DC",
            prefix:   "FS/GS",
            fy:       "25-26",
            dcType:   DCTypeTransit,
            seq:      5,
            expected: "FS/GS/25-26/T/005",
        },
        {
            name:     "Official DC",
            prefix:   "PWD/AP",
            fy:       "26-27",
            dcType:   DCTypeOfficial,
            seq:      123,
            expected: "PWD/AP/26-27/D/123",
        },
        {
            name:     "First DC",
            prefix:   "TEST",
            fy:       "25-26",
            dcType:   DCTypeTransit,
            seq:      1,
            expected: "TEST/25-26/T/001",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FormatDCNumber(tt.prefix, tt.fy, tt.dcType, tt.seq)
            if result != tt.expected {
                t.Errorf("FormatDCNumber() = %s; want %s", result, tt.expected)
            }
        })
    }
}

func TestParseDCNumber(t *testing.T) {
    tests := []struct {
        name        string
        dcNumber    string
        expected    *DCNumberParts
        expectError bool
    }{
        {
            name:     "Valid Transit DC",
            dcNumber: "FS/GS/25-26/T/005",
            expected: &DCNumberParts{
                Prefix:         "FS/GS",
                FinancialYear:  "25-26",
                DCType:         DCTypeTransit,
                SequenceNumber: 5,
            },
            expectError: false,
        },
        {
            name:     "Valid Official DC",
            dcNumber: "PWD/AP/26-27/D/123",
            expected: &DCNumberParts{
                Prefix:         "PWD/AP",
                FinancialYear:  "26-27",
                DCType:         DCTypeOfficial,
                SequenceNumber: 123,
            },
            expectError: false,
        },
        {
            name:        "Invalid format",
            dcNumber:    "INVALID",
            expectError: true,
        },
        {
            name:        "Invalid type suffix",
            dcNumber:    "TEST/25-26/X/001",
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseDCNumber(tt.dcNumber)

            if tt.expectError {
                if err == nil {
                    t.Errorf("Expected error but got none")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
                return
            }

            if result.Prefix != tt.expected.Prefix ||
                result.FinancialYear != tt.expected.FinancialYear ||
                result.DCType != tt.expected.DCType ||
                result.SequenceNumber != tt.expected.SequenceNumber {
                t.Errorf("ParseDCNumber() = %+v; want %+v", result, tt.expected)
            }
        })
    }
}

func TestIsValidDCNumber(t *testing.T) {
    tests := []struct {
        dcNumber string
        expected bool
    }{
        {"FS/GS/25-26/T/005", true},
        {"PWD/AP/26-27/D/123", true},
        {"INVALID", false},
        {"", false},
        {"TEST/25-26/X/001", false},
    }

    for _, tt := range tests {
        t.Run(tt.dcNumber, func(t *testing.T) {
            result := IsValidDCNumber(tt.dcNumber)
            if result != tt.expected {
                t.Errorf("IsValidDCNumber(%s) = %v; want %v", tt.dcNumber, result, tt.expected)
            }
        })
    }
}
```

## Testing Checklist

### Financial Year Tests
- [ ] February returns previous FY (e.g., Feb 2026 → "25-26")
- [ ] April 1st starts new FY (e.g., Apr 1 2026 → "26-27")
- [ ] March 31st is last day of FY (e.g., Mar 31 2026 → "25-26")
- [ ] December returns current FY (e.g., Dec 2026 → "26-27")
- [ ] January returns previous FY (e.g., Jan 2027 → "26-27")
- [ ] All months tested
- [ ] Year boundary handling correct
- [ ] Parse FY string correctly
- [ ] GetFinancialYearStart returns correct date
- [ ] GetFinancialYearEnd returns correct date

### DC Number Format Tests
- [ ] Format DC number correctly
- [ ] Sequence padded to 3 digits
- [ ] Transit type uses "T" suffix
- [ ] Official type uses "D" suffix
- [ ] Prefix included correctly
- [ ] Financial year included correctly
- [ ] Parse DC number correctly
- [ ] Validate DC number format
- [ ] Reject invalid DC numbers

### Sequence Generation Tests
- [ ] First DC gets sequence 001
- [ ] Second DC gets sequence 002
- [ ] Sequence increments correctly
- [ ] Different projects have independent sequences
- [ ] Different DC types have independent sequences
- [ ] Different FYs have independent sequences
- [ ] Sequence resets each financial year
- [ ] Deleted DC numbers not reused

### Concurrency Tests
- [ ] 100 concurrent DC generations produce unique numbers
- [ ] No duplicate DC numbers under load
- [ ] Sequential numbers have no gaps
- [ ] Transaction rollback doesn't create gaps
- [ ] Database locking works correctly
- [ ] Retry logic handles locked database

### Integration Tests
- [ ] Generate DC number for new project
- [ ] Generate DC number across FY boundary
- [ ] Generate Transit and Official DCs independently
- [ ] Database constraints prevent duplicates
- [ ] Error handling for missing project
- [ ] Error handling for invalid DC type

### Edge Cases
- [ ] Sequence reaches 999
- [ ] Sequence exceeds 999 (should still work, just wider)
- [ ] FY transition at midnight (Mar 31 → Apr 1)
- [ ] Concurrent requests at FY boundary
- [ ] Project with no DC prefix (should error)
- [ ] Invalid DC type (should error)

## Acceptance Criteria

### Must Have
1. DC number format: {Prefix}/{FY}/{Type}/{Seq}
2. Financial year calculated correctly (Indian FY: April-March)
3. Financial year format: YY-YY (e.g., "25-26")
4. Transit DCs use "T" suffix
5. Official DCs use "D" suffix
6. Sequential numbers padded to 3 digits (001, 002, etc.)
7. Sequential numbers auto-increment per project/type/FY
8. Numbering resets each financial year
9. Thread-safe generation (no duplicates under concurrency)
10. Deleted DC numbers are NOT reused
11. Function signature: GenerateDCNumber(db, projectID, dcType) (string, error)
12. Database transaction ensures atomicity
13. Unique constraint on dc_number prevents duplicates
14. Unit tests cover all edge cases
15. Code is well-documented

### Should Have
16. Concurrency tests verify thread-safety
17. Retry logic for database locked errors
18. Parse DC number utility function
19. Validate DC number format function
20. Financial year helper functions (start, end, parse)
21. Error messages are clear and actionable
22. Performance: generate DC number in <100ms
23. Handles 1000+ DCs per project/type/FY
24. Comprehensive test coverage (>90%)

### Nice to Have
25. DC number preview function (without incrementing)
26. Bulk DC number generation (reserve multiple)
27. DC number analytics (usage by FY, type, etc.)
28. Migration script for existing DCs
29. Admin function to reset sequence (with safeguards)
30. Audit log of DC number generation
31. Configurable sequence padding (3, 4, 5 digits)
32. Configurable FY start month (for non-Indian systems)
33. DC number format customization per project
34. Sequence gaps detection and reporting

## Implementation Notes

### SQLite-Specific Considerations
1. Use `BEGIN IMMEDIATE` for write transactions
2. Handle `SQLITE_BUSY` errors with retry
3. Use `INSERT OR IGNORE` for upsert operations
4. Consider WAL mode for better concurrency
5. Test with realistic concurrent load

### Performance Optimization
1. Index on (project_id, dc_type, financial_year)
2. Use prepared statements where possible
3. Minimize transaction duration
4. Cache project DC prefix in memory (optional)
5. Connection pooling for concurrent requests

### Migration Strategy
If implementing in existing system with DCs:
1. Add dc_number_sequences table
2. Backfill sequences from existing DCs
3. Set next_sequence to max(sequence) + 1
4. Add dc_number column (nullable initially)
5. Backfill dc_number for existing DCs
6. Make dc_number NOT NULL after backfill
7. Add unique constraint

### Future Enhancements
1. Support custom number formats per project
2. Support different FY calendars (calendar year, custom)
3. Alphanumeric sequences (A001, B001, etc.)
4. Hierarchical numbering (project/dept/sequence)
5. QR code generation from DC number
