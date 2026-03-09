# Phase 2: DC Numbering & Constants

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- Phase 1 (Database Schema) — tables must exist for sequence inserts

## Overview

Extend the DC numbering system to support the new `transfer` DC type with its own independent sequence counter (`STDC` code). Update all constants, type maps, regex patterns, and validation logic.

---

## Affected Files

| File | Changes |
|------|---------|
| `internal/services/dc_numbering.go` | Add constant, type codes, regex, validation |
| `internal/services/dc_numbering_test.go` | Add transfer type test cases |
| `internal/models/delivery_challan.go` | Update `oneof` validation tag |

---

## Tests to Write First

- [ ] `TestDCTypeTransferConstant` — Verify `DCTypeTransfer == "transfer"` exists
- [ ] `TestDCTypeCodeMapping` — Verify `dcTypeCode["transfer"] == "STDC"` and reverse mapping
- [ ] `TestGenerateDCNumber_Transfer` — Generate a transfer DC number, verify format `PREFIX-STDC-FY-001`
- [ ] `TestGenerateDCNumber_TransferSequenceIndependent` — Generate transit + transfer numbers, verify sequences are independent
- [ ] `TestPeekNextDCNumber_Transfer` — Peek at next transfer number without incrementing
- [ ] `TestFormatDCNumber_Transfer` — Format with configurable format string including transfer type
- [ ] `TestParseDCNumber_Transfer` — Parse a transfer DC number back into components
- [ ] `TestIsValidDCNumber_Transfer` — Validate transfer DC number format
- [ ] `TestFormatDCNumberConfigurable_Transfer` — Configurable format with `{TYPE}` token resolving to `STDC`
- [ ] `TestGenerateDCNumberForDate_Transfer` — Generate transfer number for specific date/FY

---

## Implementation Steps

1. **Add transfer constant** — `internal/services/dc_numbering.go`
   ```go
   // Line ~15: Add to existing constants
   const (
       DCTypeTransit  = "transit"
       DCTypeOfficial = "official"
       DCTypeTransfer = "transfer"  // NEW
   )
   ```

2. **Update type code maps** — `internal/services/dc_numbering.go`
   ```go
   // Line ~20: Add to dcTypeCode map
   var dcTypeCode = map[string]string{
       DCTypeTransit:  "TDC",
       DCTypeOfficial: "ODC",
       DCTypeTransfer: "STDC",  // NEW: Split Transfer DC
   }

   // Line ~26: Add to dcCodeToType map
   var dcCodeToType = map[string]string{
       "TDC":  DCTypeTransit,
       "ODC":  DCTypeOfficial,
       "STDC": DCTypeTransfer,  // NEW
   }
   ```

3. **Update regex pattern** — `internal/services/dc_numbering.go`
   ```go
   // Line ~31: Update dcNumberPattern to accept STDC
   // FROM: `^([A-Z]+)-(TDC|ODC)-(\d{4})-(\d+)$`
   // TO:   `^([A-Z]+)-(TDC|ODC|STDC)-(\d{4})-(\d+)$`
   ```

4. **Update validation in GenerateDCNumber** — `internal/services/dc_numbering.go`
   - Lines ~92-94: Ensure `dcTypeCode` lookup handles "transfer" (already does via map)
   - Lines ~139-140: Same for `GenerateDCNumberForDate`

5. **Update model validation** — `internal/models/delivery_challan.go`
   ```go
   // Line ~12: Update DCType validation tag
   DCType string `validate:"required,oneof=transit official transfer"`
   ```

6. **Add status constants** — `internal/models/delivery_challan.go` or new constants file
   ```go
   const (
       DCStatusDraft     = "draft"
       DCStatusIssued    = "issued"
       DCStatusSplitting = "splitting"
       DCStatusSplit     = "split"
   )
   ```

7. **Run and verify all tests** — `task test`

---

## Acceptance Criteria

- [ ] `services.DCTypeTransfer` constant exists and equals `"transfer"`
- [ ] `dcTypeCode["transfer"]` returns `"STDC"`
- [ ] `dcCodeToType["STDC"]` returns `"transfer"`
- [ ] `GenerateDCNumber(projectID, "transfer")` produces `PREFIX-STDC-YYYY-001`
- [ ] Transfer sequence is independent from transit/official sequences
- [ ] `PeekNextDCNumber()` works for transfer type
- [ ] `ParseDCNumber()` correctly parses STDC-format numbers
- [ ] `IsValidDCNumber()` accepts STDC-format numbers
- [ ] `FormatDCNumberConfigurable()` correctly substitutes `{TYPE}` with `STDC` for transfer
- [ ] Model validation accepts `dc_type="transfer"` and new status values
- [ ] All existing numbering tests still pass (no regression)
- [ ] `go vet ./...` and `go build ./...` clean
