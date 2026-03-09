# Phase 1: Database Schema & Migrations

## Status: ⬜ Not Started
## Last Updated: 2026-03-09

## Dependencies
- None (foundation phase)

## Overview

This phase creates the database tables and schema changes needed for the Split Shipment feature. The key changes are:

1. **Update `delivery_challans.dc_type` CHECK constraint** to accept `'transfer'`
2. **Update `delivery_challans.status` CHECK constraint** to accept `'splitting'` and `'split'`
3. **New `transfer_dcs` table** — stores Transfer DC-specific metadata (hub location, parent tracking)
4. **New `transfer_dc_destinations` table** — maps each destination address to a Transfer DC with planned quantities and split status
5. **New `transfer_dc_serials` table** — stores the master serial list for the Transfer DC (bulk, not per-destination)
6. **New `transfer_dc_splits` table** — tracks each split operation linking a Transfer DC to a child shipment group

---

## Database Design

### Table: `transfer_dcs`
Stores metadata specific to Transfer DCs (analogous to `dc_transit_details` for Transit DCs).

```sql
CREATE TABLE transfer_dcs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    dc_id           INTEGER NOT NULL UNIQUE REFERENCES delivery_challans(id) ON DELETE CASCADE,
    hub_address_id  INTEGER NOT NULL REFERENCES addresses(id),
    template_id     INTEGER REFERENCES dc_templates(id) ON DELETE SET NULL,
    tax_type        TEXT NOT NULL DEFAULT 'cgst_sgst' CHECK (tax_type IN ('cgst_sgst', 'igst')),
    reverse_charge  TEXT NOT NULL DEFAULT 'N' CHECK (reverse_charge IN ('Y', 'N')),
    transporter_name TEXT,
    vehicle_number  TEXT,
    eway_bill_number TEXT,
    docket_number   TEXT,
    notes           TEXT,
    num_destinations INTEGER NOT NULL DEFAULT 0,
    num_split       INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_transfer_dcs_dc_id ON transfer_dcs(dc_id);
```

### Table: `transfer_dc_destinations`
Maps each final delivery destination to the Transfer DC with planned product quantities.

```sql
CREATE TABLE transfer_dc_destinations (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
    ship_to_address_id  INTEGER NOT NULL REFERENCES addresses(id),
    split_group_id      INTEGER REFERENCES transfer_dc_splits(id) ON DELETE SET NULL,
    is_split            INTEGER NOT NULL DEFAULT 0,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tdc_dest_transfer_dc_id ON transfer_dc_destinations(transfer_dc_id);
CREATE INDEX idx_tdc_dest_ship_to ON transfer_dc_destinations(ship_to_address_id);
CREATE INDEX idx_tdc_dest_split_group ON transfer_dc_destinations(split_group_id);
CREATE UNIQUE INDEX idx_tdc_dest_unique ON transfer_dc_destinations(transfer_dc_id, ship_to_address_id);
```

### Table: `transfer_dc_destination_quantities`
Per-product, per-destination planned quantities (the quantity grid data).

```sql
CREATE TABLE transfer_dc_destination_quantities (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    destination_id  INTEGER NOT NULL REFERENCES transfer_dc_destinations(id) ON DELETE CASCADE,
    product_id      INTEGER NOT NULL REFERENCES products(id),
    quantity        INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tdc_dq_destination_id ON transfer_dc_destination_quantities(destination_id);
CREATE UNIQUE INDEX idx_tdc_dq_unique ON transfer_dc_destination_quantities(destination_id, product_id);
```

### Table: `transfer_dc_splits`
Tracks each split operation — links a Transfer DC to a child shipment group.

```sql
CREATE TABLE transfer_dc_splits (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
    shipment_group_id   INTEGER NOT NULL UNIQUE REFERENCES shipment_groups(id) ON DELETE CASCADE,
    split_number        INTEGER NOT NULL,
    created_by          INTEGER REFERENCES users(id),
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tdc_splits_transfer_dc_id ON transfer_dc_splits(transfer_dc_id);
CREATE UNIQUE INDEX idx_tdc_splits_unique ON transfer_dc_splits(transfer_dc_id, split_number);
```

### ALTER existing tables

```sql
-- delivery_challans: expand dc_type CHECK constraint
-- SQLite doesn't support ALTER CHECK, so we need a pragmatic approach:
-- Option A: Remove CHECK constraint in new migration (SQLite allows inserting any value if CHECK is dropped)
-- Option B: Recreate table (complex, risky with existing data)
-- Option C: Just insert 'transfer' — SQLite CHECK constraints can be worked around
-- We'll go with Option A: the application layer enforces valid dc_type values via Go validation

-- delivery_challans: add transfer_dc_id FK column for linking
ALTER TABLE delivery_challans ADD COLUMN transfer_dc_id INTEGER REFERENCES transfer_dcs(id);

-- shipment_groups: add transfer_dc_id to track parent Transfer DC
ALTER TABLE shipment_groups ADD COLUMN transfer_dc_id INTEGER REFERENCES transfer_dcs(id);
ALTER TABLE shipment_groups ADD COLUMN split_id INTEGER REFERENCES transfer_dc_splits(id);
```

---

## Migration File

**File**: `internal/migrations/00034_create_transfer_dc_tables.sql`

---

## Tests to Write First

- [ ] `TestMigration00034_TablesExist` — Verify all 4 new tables exist after migration
- [ ] `TestMigration00034_TransferDCInsert` — Insert a transfer DC record and verify all columns
- [ ] `TestMigration00034_DestinationInsert` — Insert destinations with quantities
- [ ] `TestMigration00034_SplitInsert` — Insert a split record linking Transfer DC to shipment group
- [ ] `TestMigration00034_CascadeDelete` — Verify CASCADE behavior when Transfer DC is deleted
- [ ] `TestMigration00034_UniqueConstraints` — Verify unique constraints on destination and split tables
- [ ] `TestDeliveryChallanTransferType` — Verify `dc_type='transfer'` can be inserted into `delivery_challans`
- [ ] `TestDeliveryChallanSplittingStatus` — Verify `status='splitting'` and `status='split'` work

---

## Implementation Steps

1. **Create migration file** — `internal/migrations/00034_create_transfer_dc_tables.sql`
   - `transfer_dcs` table with indexes
   - `transfer_dc_destinations` table with indexes and unique constraint
   - `transfer_dc_destination_quantities` table with indexes
   - `transfer_dc_splits` table with indexes
   - ALTER `delivery_challans` to add `transfer_dc_id` column
   - ALTER `shipment_groups` to add `transfer_dc_id` and `split_id` columns

2. **Update model validation** — `internal/models/delivery_challan.go`
   - Change `DCType` validation from `oneof=transit official` to `oneof=transit official transfer`
   - Change `Status` validation (if exists) to include `splitting` and `split`

3. **Add Transfer DC model structs** — `internal/models/transfer_dc.go` (NEW FILE)
   ```go
   type TransferDC struct {
       ID              int
       DCID            int
       HubAddressID    int
       TemplateID      *int
       TaxType         string
       ReverseCharge   string
       TransporterName string
       VehicleNumber   string
       EwayBillNumber  string
       DocketNumber    string
       Notes           string
       NumDestinations int
       NumSplit        int
       CreatedAt       time.Time
       UpdatedAt       time.Time
       // Computed/joined fields
       HubAddressName  string
       TemplateName    string
       DCNumber        string
       DCStatus        string
       ChallanDate     *string
       ProjectID       int
   }

   type TransferDCDestination struct {
       ID              int
       TransferDCID    int
       ShipToAddressID int
       SplitGroupID    *int
       IsSplit         bool
       CreatedAt       time.Time
       // Computed/joined fields
       AddressName     string
       Quantities      []TransferDCDestinationQty
   }

   type TransferDCDestinationQty struct {
       ID            int
       DestinationID int
       ProductID     int
       Quantity      int
       // Computed/joined
       ProductName   string
   }

   type TransferDCSplit struct {
       ID              int
       TransferDCID    int
       ShipmentGroupID int
       SplitNumber     int
       CreatedBy       int
       CreatedAt       time.Time
       // Computed/joined
       ShipmentGroup   *ShipmentGroup
       Destinations    []*TransferDCDestination
   }
   ```

4. **Run migration** — `task migrate` to apply schema changes

5. **Verify** — Run tests, confirm tables exist, confirm INSERT/SELECT works for all new tables

---

## Acceptance Criteria

- [ ] Migration `00034` runs successfully on fresh and existing databases
- [ ] All 4 new tables (`transfer_dcs`, `transfer_dc_destinations`, `transfer_dc_destination_quantities`, `transfer_dc_splits`) exist with correct columns and indexes
- [ ] `delivery_challans` table accepts `dc_type='transfer'` and `status='splitting'`/`'split'`
- [ ] `delivery_challans.transfer_dc_id` column exists
- [ ] `shipment_groups.transfer_dc_id` and `shipment_groups.split_id` columns exist
- [ ] CASCADE deletes work: deleting a transfer_dc cascades to destinations, quantities, and splits
- [ ] Unique constraints enforced: no duplicate destination per Transfer DC, no duplicate split number
- [ ] Model structs in `internal/models/transfer_dc.go` compile and validate correctly
- [ ] Existing delivery challan operations (transit/official) are NOT affected by schema changes
- [ ] All existing tests still pass
