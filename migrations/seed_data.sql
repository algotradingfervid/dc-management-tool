-- Seed data for development and testing

-- Insert test users (password: "password123" for all)
-- bcrypt hash of "password123"
INSERT INTO users (username, password_hash, full_name, email) VALUES
('admin', '$2a$10$I/nbAST4L/brwicvbf.vMO04XSxUwUDVW2JdX3iVztgMR9J84dSFq', 'Admin User', 'admin@example.com'),
('john', '$2a$10$I/nbAST4L/brwicvbf.vMO04XSxUwUDVW2JdX3iVztgMR9J84dSFq', 'John Doe', 'john@example.com'),
('jane', '$2a$10$I/nbAST4L/brwicvbf.vMO04XSxUwUDVW2JdX3iVztgMR9J84dSFq', 'Jane Smith', 'jane@example.com');

-- Insert sample projects
INSERT INTO projects (name, description, dc_prefix, tender_ref_number, tender_ref_details, po_reference, po_date, bill_from_address, company_gstin, created_by) VALUES
('Smart City Project - Phase 1', 'Smart city infrastructure deployment with IoT sensors', 'SCP', 'TN-2024-001', 'Smart City Modernization Initiative', 'PO/2024/001', '2024-01-15', 'Tech Solutions Inc, 123 Tech Park, Bangalore - 560001', '36AACCF9742K1Z8', 1),
('IoT Deployment - Corporate HQ', 'Corporate IoT deployment for monitoring and automation', 'IOT', 'TN-2024-002', 'IoT Infrastructure Setup', 'PO/2024/002', '2024-02-10', 'Tech Solutions Inc, 123 Tech Park, Bangalore - 560001', '36AACCF9742K1Z8', 1),
('Network Infrastructure Upgrade', 'Network infrastructure upgrade and expansion', 'NET', 'TN-2024-003', 'Network Modernization Project', 'PO/2024/003', '2024-03-05', 'Tech Solutions Inc, 123 Tech Park, Bangalore - 560001', '36AACCF9742K1Z8', 2);

-- Insert sample products for project 1
INSERT INTO products (project_id, item_name, item_description, hsn_code, uom, brand_model, per_unit_price, gst_percentage) VALUES
(1, 'Network Switch 24-Port', 'Managed Gigabit Ethernet Switch with 24 ports', '85176990', 'Nos', 'Cisco SG350-24', 15000.00, 18.00),
(1, 'CAT6 Cable (100m)', 'Category 6 Ethernet cable 100m roll', '85444200', 'Roll', 'D-Link CAT6 UTP', 2500.00, 18.00),
(1, 'PoE Injector', 'Power over Ethernet Injector 802.3af', '85176290', 'Nos', 'TP-Link TL-POE150S', 3500.00, 18.00),
(1, 'Wall Mount Rack 6U', '6U Wall Mount Network Cabinet', '73269099', 'Nos', 'APC NetShelter 6U', 5000.00, 18.00);

-- Insert sample products for project 2
INSERT INTO products (project_id, item_name, item_description, hsn_code, uom, brand_model, per_unit_price, gst_percentage) VALUES
(2, 'IoT Gateway Device', 'Industrial IoT Gateway with WiFi and Ethernet', '85176200', 'Nos', 'Advantech UNO-2271G', 25000.00, 18.00),
(2, 'Temperature Sensor', 'Digital temperature sensor -40 to 125C', '90259000', 'Nos', 'Honeywell T6820', 1500.00, 18.00),
(2, 'Motion Detector', 'PIR motion detector with adjustable sensitivity', '85318000', 'Nos', 'Bosch ISC-BPR2-W12', 2000.00, 18.00);

-- Insert address list config for project 1
INSERT INTO address_list_configs (project_id, address_type, column_definitions) VALUES
(1, 'bill_to', '[
    {"name": "Legal Name", "required": true},
    {"name": "GSTIN", "required": true},
    {"name": "Billing Address", "required": true},
    {"name": "State Code", "required": false}
]'),
(1, 'ship_to', '[
    {"name": "Site Name", "required": true},
    {"name": "Site Address", "required": true},
    {"name": "Contact Person", "required": false},
    {"name": "Contact Number", "required": false}
]');

-- Insert sample bill-to addresses for project 1
INSERT INTO addresses (config_id, address_data) VALUES
(1, '{"Legal Name": "Smart City Authority", "GSTIN": "29SMCTY1234A1Z1", "Billing Address": "100 Municipal Building, MG Road, Bangalore - 560001", "State Code": "29"}'),
(1, '{"Legal Name": "Urban Development Corp", "GSTIN": "29URBDV5678B2Y2", "Billing Address": "200 Government Complex, Residency Road, Bangalore - 560025", "State Code": "29"}');

-- Insert sample ship-to addresses for project 1
INSERT INTO addresses (config_id, address_data) VALUES
(2, '{"Site Name": "Central Park Site", "Site Address": "Central Park, Cubbon Park Area, Bangalore - 560001", "Contact Person": "Rajesh Kumar", "Contact Number": "+91-9876543210"}'),
(2, '{"Site Name": "East Zone Hub", "Site Address": "Whitefield Tech Park, Phase 2, Bangalore - 560066", "Contact Person": "Priya Sharma", "Contact Number": "+91-9876543211"}'),
(2, '{"Site Name": "South Zone Office", "Site Address": "JP Nagar 3rd Phase, Bangalore - 560078", "Contact Person": "Amit Patel", "Contact Number": "+91-9876543212"}');

-- Insert DC template for project 1
INSERT INTO dc_templates (project_id, name, purpose) VALUES
(1, 'Standard Network Kit', 'Standard networking equipment bundle for site installations'),
(1, 'Basic IoT Setup', 'Basic IoT sensor kit for initial deployment');

-- Insert template products
INSERT INTO dc_template_products (template_id, product_id, default_quantity) VALUES
(1, 1, 2),  -- 2x Network Switch
(1, 2, 5),  -- 5x CAT6 Cable
(1, 3, 4),  -- 4x PoE Injector
(2, 1, 1),  -- 1x Network Switch
(2, 2, 2);  -- 2x CAT6 Cable

-- Insert sample delivery challans
INSERT INTO delivery_challans (project_id, dc_number, dc_type, status, template_id, bill_to_address_id, ship_to_address_id, challan_date, created_by) VALUES
(1, 'TDC-001', 'transit', 'draft', 1, NULL, 3, '2024-03-15', 1),
(1, 'ODC-001', 'official', 'issued', 2, 1, 3, '2024-03-20', 1),
(1, 'TDC-002', 'transit', 'issued', NULL, NULL, 4, '2024-03-22', 2);

-- Update issued DCs
UPDATE delivery_challans SET issued_at = '2024-03-20 10:30:00', issued_by = 1 WHERE dc_number = 'ODC-001';
UPDATE delivery_challans SET issued_at = '2024-03-22 14:15:00', issued_by = 2 WHERE dc_number = 'TDC-002';

-- Insert transit details for transit DCs
INSERT INTO dc_transit_details (dc_id, transporter_name, vehicle_number, eway_bill_number, notes) VALUES
(1, 'Fast Logistics Pvt Ltd', 'KA-01-AB-1234', 'EWB123456789', 'Handle with care - fragile items'),
(3, 'Express Movers', 'KA-05-CD-5678', 'EWB987654321', 'Urgent delivery required');

-- Insert line items for DC 1 (TDC-001 - draft)
INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES
(1, 1, 2, 15000.00, 18.00, 30000.00, 5400.00, 35400.00, 1),
(1, 2, 5, 2500.00, 18.00, 12500.00, 2250.00, 14750.00, 2),
(1, 3, 4, 3500.00, 18.00, 14000.00, 2520.00, 16520.00, 3);

-- Insert line items for DC 2 (ODC-001 - issued)
INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES
(2, 1, 1, 15000.00, 18.00, 15000.00, 2700.00, 17700.00, 1),
(2, 2, 2, 2500.00, 18.00, 5000.00, 900.00, 5900.00, 2);

-- Insert serial numbers for DC 2 line items (issued DC must have serials)
INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES
(1, 4, 'NS24P-2024-001'),
(1, 5, 'CAT6-ROLL-2024-001'),
(1, 5, 'CAT6-ROLL-2024-002');

-- Insert line items for DC 3 (TDC-002 - issued)
INSERT INTO dc_line_items (dc_id, product_id, quantity, rate, tax_percentage, taxable_amount, tax_amount, total_amount, line_order) VALUES
(3, 4, 3, 5000.00, 18.00, 15000.00, 2700.00, 17700.00, 1);

-- Insert serial numbers for DC 3
INSERT INTO serial_numbers (project_id, line_item_id, serial_number) VALUES
(1, 6, 'RACK6U-2024-001'),
(1, 6, 'RACK6U-2024-002'),
(1, 6, 'RACK6U-2024-003');

-- Update project last DC numbers
UPDATE projects SET last_transit_dc_number = 2, last_official_dc_number = 1 WHERE id = 1;
