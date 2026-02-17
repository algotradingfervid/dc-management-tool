ALTER TABLE delivery_challans ADD COLUMN shipment_group_id INTEGER REFERENCES shipment_groups(id);
ALTER TABLE delivery_challans ADD COLUMN bill_from_address_id INTEGER;
ALTER TABLE delivery_challans ADD COLUMN dispatch_from_address_id INTEGER;
