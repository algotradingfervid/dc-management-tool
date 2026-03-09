-- +goose Up
CREATE TABLE IF NOT EXISTS transfer_dcs (
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

CREATE TABLE IF NOT EXISTS transfer_dc_splits (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    transfer_dc_id      INTEGER NOT NULL REFERENCES transfer_dcs(id) ON DELETE CASCADE,
    shipment_group_id   INTEGER NOT NULL UNIQUE REFERENCES shipment_groups(id) ON DELETE CASCADE,
    split_number        INTEGER NOT NULL,
    created_by          INTEGER REFERENCES users(id),
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tdc_splits_transfer_dc_id ON transfer_dc_splits(transfer_dc_id);
CREATE UNIQUE INDEX idx_tdc_splits_unique ON transfer_dc_splits(transfer_dc_id, split_number);

CREATE TABLE IF NOT EXISTS transfer_dc_destinations (
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

CREATE TABLE IF NOT EXISTS transfer_dc_destination_quantities (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    destination_id  INTEGER NOT NULL REFERENCES transfer_dc_destinations(id) ON DELETE CASCADE,
    product_id      INTEGER NOT NULL REFERENCES products(id),
    quantity        INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tdc_dq_destination_id ON transfer_dc_destination_quantities(destination_id);
CREATE UNIQUE INDEX idx_tdc_dq_unique ON transfer_dc_destination_quantities(destination_id, product_id);

ALTER TABLE delivery_challans ADD COLUMN transfer_dc_id INTEGER REFERENCES transfer_dcs(id);
ALTER TABLE shipment_groups ADD COLUMN transfer_dc_id INTEGER REFERENCES transfer_dcs(id);
ALTER TABLE shipment_groups ADD COLUMN split_id INTEGER REFERENCES transfer_dc_splits(id);

-- +goose Down
ALTER TABLE shipment_groups DROP COLUMN split_id;
ALTER TABLE shipment_groups DROP COLUMN transfer_dc_id;
ALTER TABLE delivery_challans DROP COLUMN transfer_dc_id;
DROP TABLE IF EXISTS transfer_dc_destination_quantities;
DROP TABLE IF EXISTS transfer_dc_destinations;
DROP TABLE IF EXISTS transfer_dc_splits;
DROP TABLE IF EXISTS transfer_dcs;
