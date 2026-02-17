-- ============================================================
-- Comprehensive Seed Data: Smart Classrooms OAVS Odisha
-- Project for deploying smart classroom equipment to
-- Odisha Adarsha Vidyalaya Sangathan schools across Odisha
-- ============================================================

-- Insert the project (created_by = 1 assumes admin user exists)
INSERT INTO projects (
    name, description, dc_prefix, tender_ref_number, tender_ref_details,
    po_reference, po_date, bill_from_address, dispatch_from_address,
    company_gstin, company_email, company_cin, purpose_text, created_by
) VALUES (
    'Smart Classrooms OAVS Odisha',
    'Supply, installation and commissioning of 2250 smart classrooms across 314 Odisha Adarsha Vidyalayas under Department of School & Mass Education, Govt of Odisha',
    'OAVS',
    'OCAC-OAVS-SC-2025-001',
    'RFP for Selection of Agency for Supply, Installation, Commissioning and Maintenance of Smart Classroom Solutions in OAVs across Odisha',
    'PO/OAVS/2025/SC-001',
    '2025-04-01',
    'Fountane Inc, Plot No. 42, Madhapur SEZ, Hyderabad, Telangana - 500081',
    'Fountane Inc, Warehouse No. 7, Patia Industrial Estate, Bhubaneswar, Odisha - 751024',
    '36AACCF9742K1Z8',
    'projects@fountane.com',
    'U72200TG2015PTC099825',
    'SUPPLY AND INSTALLATION OF SMART CLASSROOM EQUIPMENT AS PER PURCHASE ORDER',
    1
);

-- Get the project ID (will be the last inserted)
-- We'll reference it as a subquery where needed

-- ============================================================
-- PRODUCTS (10 items for smart classroom deployment)
-- ============================================================
INSERT INTO products (project_id, item_name, item_description, hsn_code, uom, brand_model, per_unit_price, gst_percentage) VALUES
((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Interactive Flat Panel Display 75"',
 '75-inch 4K UHD Interactive Flat Panel with 20-point multi-touch, Android 11, built-in speakers, wall mount kit included',
 '85285900', 'Nos', 'BenQ RP7504', 185000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Mini PC (OPS Module)',
 'OPS Slot-in PC Module, Intel Core i5-12400, 8GB DDR4, 256GB SSD, Windows 11 Pro, WiFi 6',
 '84713010', 'Nos', 'BenQ OPS PC Module SI-01', 42000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Document Camera',
 '13MP USB Document Camera with A3 scanning area, LED illumination, auto-focus, foldable design',
 '85258090', 'Nos', 'ELMO MX-P2', 28000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Wireless Presentation System',
 'Wireless screen sharing device supporting 4 simultaneous users, HDMI/USB-C, AirPlay, Miracast',
 '85176290', 'Nos', 'BenQ InstaShow WDC20', 35000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Ceiling Mounted Speaker System',
 '2x 20W ceiling speakers with amplifier, wall-mount volume controller, 6m cable harness',
 '85182200', 'Set', 'JBL CSS-15C-VA (Pair + Amp)', 12500.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'UPS 2KVA Online',
 '2KVA / 1800W Online Double Conversion UPS with 30min battery backup, rack/tower convertible',
 '85043400', 'Nos', 'APC Smart-UPS SRT2200UXI', 38000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Structured Cabling Kit',
 'Cat6 LAN cabling, conduit, I/O boxes, patch panel, face plates for single classroom installation',
 '85444200', 'Kit', 'D-Link Cat6 Cabling Kit', 4500.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Wireless Access Point',
 'Dual-band WiFi 6 Access Point, 802.11ax, ceiling mount, PoE powered, supports 100+ clients',
 '85176290', 'Nos', 'TP-Link EAP670', 8500.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Teacher Tablet 10"',
 '10.4-inch Android tablet with stylus, 6GB RAM, 128GB storage, classroom management app pre-loaded',
 '84713010', 'Nos', 'Samsung Galaxy Tab S6 Lite', 22000.00, 18.00),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Wall Mount Bracket & Installation Kit',
 'Heavy-duty tilt wall mount for 65-86" panels, VESA 600x400, cable management tray, anchor bolts',
 '73269099', 'Set', 'Ergotron WM-DA-02', 3500.00, 18.00);

-- ============================================================
-- ADDRESS LIST CONFIGS
-- ============================================================

-- Bill-to address config
INSERT INTO address_list_configs (project_id, address_type, column_definitions) VALUES
((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'bill_to',
 '[
    {"name": "Office Name", "required": true},
    {"name": "GSTIN", "required": true},
    {"name": "Address", "required": true},
    {"name": "State Code", "required": false},
    {"name": "Contact Person", "required": false},
    {"name": "Phone", "required": false}
 ]');

-- Ship-to address config
INSERT INTO address_list_configs (project_id, address_type, column_definitions) VALUES
((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'ship_to',
 '[
    {"name": "School Name", "required": true},
    {"name": "School Code", "required": true},
    {"name": "Block", "required": true},
    {"name": "Address", "required": true},
    {"name": "PIN Code", "required": true},
    {"name": "Principal Name", "required": false},
    {"name": "Principal Phone", "required": false}
 ]');

-- ============================================================
-- BILL-TO ADDRESSES (10 district/regional offices)
-- ============================================================
INSERT INTO addresses (config_id, address_data, district_name) VALUES

-- 1
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "OAVS State Headquarters", "GSTIN": "21OAVS0001A1Z1", "Address": "N/6-012, Nayapalli, IRC Village, Bhubaneswar, Odisha - 751015", "State Code": "21", "Contact Person": "Dr. Pratap Kumar Mishra", "Phone": "+91-674-2562890"}',
 'Khordha'),

-- 2
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "OCAC - Odisha Computer Application Centre", "GSTIN": "21OCAC0002B2Z2", "Address": "N-1/7-D, Acharya Vihar, RRL Post Office, Bhubaneswar - 751013", "State Code": "21", "Contact Person": "Shri Manoj Kumar Pattnaik", "Phone": "+91-674-2567280"}',
 'Khordha'),

-- 3
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Cuttack", "GSTIN": "21DEOC0003C3Z3", "Address": "District Education Office, Cantonment Road, Cuttack - 753001", "State Code": "21", "Contact Person": "Shri Ramesh Chandra Behera", "Phone": "+91-671-2301456"}',
 'Cuttack'),

-- 4
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Ganjam", "GSTIN": "21DEOG0004D4Z4", "Address": "District Education Office, Near Collectorate, Chhatrapur, Ganjam - 761020", "State Code": "21", "Contact Person": "Shri Sunil Kumar Dash", "Phone": "+91-680-2261345"}',
 'Ganjam'),

-- 5
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Mayurbhanj", "GSTIN": "21DEOM0005E5Z5", "Address": "District Education Office, Baripada, Mayurbhanj - 757001", "State Code": "21", "Contact Person": "Shri Ashok Kumar Mohapatra", "Phone": "+91-6792-252678"}',
 'Mayurbhanj'),

-- 6
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Sundargarh", "GSTIN": "21DEOS0006F6Z6", "Address": "District Education Office, Near SP Office, Sundargarh - 770001", "State Code": "21", "Contact Person": "Shri Pradeep Kumar Sahu", "Phone": "+91-6622-272345"}',
 'Sundargarh'),

-- 7
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Balasore", "GSTIN": "21DEOB0007G7Z7", "Address": "District Education Office, Sahadevkhunta, Balasore - 756001", "State Code": "21", "Contact Person": "Smt. Lipika Rath", "Phone": "+91-6782-265890"}',
 'Balasore'),

-- 8
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Sambalpur", "GSTIN": "21DEOS0008H8Z8", "Address": "District Education Office, Ainthapali, Sambalpur - 768004", "State Code": "21", "Contact Person": "Shri Bijay Kumar Panda", "Phone": "+91-663-2540123"}',
 'Sambalpur'),

-- 9
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Koraput", "GSTIN": "21DEOK0009I9Z9", "Address": "District Education Office, Near Bus Stand, Koraput - 764020", "State Code": "21", "Contact Person": "Shri Debashis Patra", "Phone": "+91-6852-251234"}',
 'Koraput'),

-- 10
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'bill_to'),
 '{"Office Name": "DEO Kalahandi", "GSTIN": "21DEOK0010J0Z0", "Address": "District Education Office, Bhawanipatna, Kalahandi - 766001", "State Code": "21", "Contact Person": "Shri Narayan Pradhan", "Phone": "+91-6670-230567"}',
 'Kalahandi');

-- ============================================================
-- SHIP-TO ADDRESSES (100 OAV schools across Odisha districts)
-- ============================================================

-- Khordha District (10 schools)
INSERT INTO addresses (config_id, address_data, district_name, mandal_name, mandal_code) VALUES
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Balianta", "School Code": "OAV-KHD-001", "Block": "Balianta", "Address": "At/Po - Balianta, Dist - Khordha, Odisha", "PIN Code": "752101", "Principal Name": "Shri Sanjay Kumar Nayak", "Principal Phone": "+91-9437201001"}',
 'Khordha', 'Balianta', 'KHD-BAL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Begunia", "School Code": "OAV-KHD-002", "Block": "Begunia", "Address": "At/Po - Begunia, Dist - Khordha, Odisha", "PIN Code": "752062", "Principal Name": "Smt. Sasmita Mohanty", "Principal Phone": "+91-9437201002"}',
 'Khordha', 'Begunia', 'KHD-BEG'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bolagarh", "School Code": "OAV-KHD-003", "Block": "Bolagarh", "Address": "At/Po - Bolagarh, Dist - Khordha, Odisha", "PIN Code": "752066", "Principal Name": "Shri Ranjan Kumar Das", "Principal Phone": "+91-9437201003"}',
 'Khordha', 'Bolagarh', 'KHD-BOL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jatni", "School Code": "OAV-KHD-004", "Block": "Jatni", "Address": "At/Po - Jatni, Dist - Khordha, Odisha", "PIN Code": "752050", "Principal Name": "Shri Prasanna Kumar Sahoo", "Principal Phone": "+91-9437201004"}',
 'Khordha', 'Jatni', 'KHD-JAT'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Tangi", "School Code": "OAV-KHD-005", "Block": "Tangi", "Address": "At/Po - Tangi, Dist - Khordha, Odisha", "PIN Code": "752079", "Principal Name": "Smt. Mamata Panda", "Principal Phone": "+91-9437201005"}',
 'Khordha', 'Tangi', 'KHD-TAN'),

-- Cuttack District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Athagarh", "School Code": "OAV-CTC-001", "Block": "Athagarh", "Address": "At/Po - Athagarh, Dist - Cuttack, Odisha", "PIN Code": "754029", "Principal Name": "Shri Debasis Swain", "Principal Phone": "+91-9437202001"}',
 'Cuttack', 'Athagarh', 'CTC-ATH'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Banki", "School Code": "OAV-CTC-002", "Block": "Banki", "Address": "At/Po - Banki, Dist - Cuttack, Odisha", "PIN Code": "754008", "Principal Name": "Shri Kishore Chandra Panda", "Principal Phone": "+91-9437202002"}',
 'Cuttack', 'Banki', 'CTC-BNK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Baramba", "School Code": "OAV-CTC-003", "Block": "Baramba", "Address": "At/Po - Baramba, Dist - Cuttack, Odisha", "PIN Code": "754005", "Principal Name": "Smt. Gitanjali Mishra", "Principal Phone": "+91-9437202003"}',
 'Cuttack', 'Baramba', 'CTC-BAR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Nischintakoili", "School Code": "OAV-CTC-004", "Block": "Nischintakoili", "Address": "At/Po - Nischintakoili, Dist - Cuttack, Odisha", "PIN Code": "754207", "Principal Name": "Shri Sarat Kumar Rout", "Principal Phone": "+91-9437202004"}',
 'Cuttack', 'Nischintakoili', 'CTC-NIS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Salepur", "School Code": "OAV-CTC-005", "Block": "Salepur", "Address": "At/Po - Salepur, Dist - Cuttack, Odisha", "PIN Code": "754202", "Principal Name": "Shri Niranjan Jena", "Principal Phone": "+91-9437202005"}',
 'Cuttack', 'Salepur', 'CTC-SAL'),

-- Ganjam District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Aska", "School Code": "OAV-GJM-001", "Block": "Aska", "Address": "At/Po - Aska, Dist - Ganjam, Odisha", "PIN Code": "761110", "Principal Name": "Shri Santosh Kumar Sahu", "Principal Phone": "+91-9437203001"}',
 'Ganjam', 'Aska', 'GJM-ASK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bhanjanagar", "School Code": "OAV-GJM-002", "Block": "Bhanjanagar", "Address": "At/Po - Bhanjanagar, Dist - Ganjam, Odisha", "PIN Code": "761126", "Principal Name": "Smt. Sunita Behera", "Principal Phone": "+91-9437203002"}',
 'Ganjam', 'Bhanjanagar', 'GJM-BHJ'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Chatrapur", "School Code": "OAV-GJM-003", "Block": "Chatrapur", "Address": "At/Po - Chatrapur, Dist - Ganjam, Odisha", "PIN Code": "761020", "Principal Name": "Shri Manoranjan Pradhan", "Principal Phone": "+91-9437203003"}',
 'Ganjam', 'Chatrapur', 'GJM-CHP'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Digapahandi", "School Code": "OAV-GJM-004", "Block": "Digapahandi", "Address": "At/Po - Digapahandi, Dist - Ganjam, Odisha", "PIN Code": "761012", "Principal Name": "Shri Gagan Bihari Sethi", "Principal Phone": "+91-9437203004"}',
 'Ganjam', 'Digapahandi', 'GJM-DIG'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Hinjilicut", "School Code": "OAV-GJM-005", "Block": "Hinjilicut", "Address": "At/Po - Hinjilicut, Dist - Ganjam, Odisha", "PIN Code": "761102", "Principal Name": "Smt. Prativa Nayak", "Principal Phone": "+91-9437203005"}',
 'Ganjam', 'Hinjilicut', 'GJM-HIN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Khallikote", "School Code": "OAV-GJM-006", "Block": "Khallikote", "Address": "At/Po - Khallikote, Dist - Ganjam, Odisha", "PIN Code": "761030", "Principal Name": "Shri Basanta Kumar Patra", "Principal Phone": "+91-9437203006"}',
 'Ganjam', 'Khallikote', 'GJM-KHL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Polasara", "School Code": "OAV-GJM-007", "Block": "Polasara", "Address": "At/Po - Polasara, Dist - Ganjam, Odisha", "PIN Code": "761105", "Principal Name": "Shri Rabindra Nath Dash", "Principal Phone": "+91-9437203007"}',
 'Ganjam', 'Polasara', 'GJM-POL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Purusottampur", "School Code": "OAV-GJM-008", "Block": "Purusottampur", "Address": "At/Po - Purusottampur, Dist - Ganjam, Odisha", "PIN Code": "761018", "Principal Name": "Smt. Jayanti Mishra", "Principal Phone": "+91-9437203008"}',
 'Ganjam', 'Purusottampur', 'GJM-PUR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kabisuryanagar", "School Code": "OAV-GJM-009", "Block": "Kabisuryanagar", "Address": "At/Po - Kabisuryanagar, Dist - Ganjam, Odisha", "PIN Code": "761104", "Principal Name": "Shri Dillip Kumar Sahu", "Principal Phone": "+91-9437203009"}',
 'Ganjam', 'Kabisuryanagar', 'GJM-KAB'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rangeilunda", "School Code": "OAV-GJM-010", "Block": "Rangeilunda", "Address": "At/Po - Rangeilunda, Near Berhampur, Dist - Ganjam, Odisha", "PIN Code": "760002", "Principal Name": "Shri Satya Narayan Panda", "Principal Phone": "+91-9437203010"}',
 'Ganjam', 'Rangeilunda', 'GJM-RAN'),

-- Mayurbhanj District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rairangpur", "School Code": "OAV-MBJ-001", "Block": "Rairangpur", "Address": "At - Sanmouda, Po - Rairangpur, Dist - Mayurbhanj, Odisha", "PIN Code": "757043", "Principal Name": "Shri Tapan Kumar Mohanty", "Principal Phone": "+91-9437204001"}',
 'Mayurbhanj', 'Rairangpur', 'MBJ-RAI'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Baripada", "School Code": "OAV-MBJ-002", "Block": "Baripada", "Address": "At/Po - Baripada, Dist - Mayurbhanj, Odisha", "PIN Code": "757001", "Principal Name": "Smt. Kabita Sahoo", "Principal Phone": "+91-9437204002"}',
 'Mayurbhanj', 'Baripada', 'MBJ-BAR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Karanjia", "School Code": "OAV-MBJ-003", "Block": "Karanjia", "Address": "At/Po - Karanjia, Dist - Mayurbhanj, Odisha", "PIN Code": "757037", "Principal Name": "Shri Biswajit Nayak", "Principal Phone": "+91-9437204003"}',
 'Mayurbhanj', 'Karanjia', 'MBJ-KAR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Udala", "School Code": "OAV-MBJ-004", "Block": "Udala", "Address": "At/Po - Udala, Dist - Mayurbhanj, Odisha", "PIN Code": "757041", "Principal Name": "Shri Pradeep Kumar Singh", "Principal Phone": "+91-9437204004"}',
 'Mayurbhanj', 'Udala', 'MBJ-UDA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bangriposi", "School Code": "OAV-MBJ-005", "Block": "Bangriposi", "Address": "At/Po - Bangriposi, Dist - Mayurbhanj, Odisha", "PIN Code": "757032", "Principal Name": "Smt. Laxmi Priya Hota", "Principal Phone": "+91-9437204005"}',
 'Mayurbhanj', 'Bangriposi', 'MBJ-BNG'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Betanoti", "School Code": "OAV-MBJ-006", "Block": "Betanoti", "Address": "At/Po - Betanoti, Dist - Mayurbhanj, Odisha", "PIN Code": "757025", "Principal Name": "Shri Hemanta Kumar Das", "Principal Phone": "+91-9437204006"}',
 'Mayurbhanj', 'Betanoti', 'MBJ-BET'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Morada", "School Code": "OAV-MBJ-007", "Block": "Morada", "Address": "At/Po - Morada, Dist - Mayurbhanj, Odisha", "PIN Code": "757027", "Principal Name": "Shri Susanta Kumar Behera", "Principal Phone": "+91-9437204007"}',
 'Mayurbhanj', 'Morada', 'MBJ-MOR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Thakurmunda", "School Code": "OAV-MBJ-008", "Block": "Thakurmunda", "Address": "At/Po - Thakurmunda, Dist - Mayurbhanj, Odisha", "PIN Code": "757038", "Principal Name": "Smt. Radha Rani Soren", "Principal Phone": "+91-9437204008"}',
 'Mayurbhanj', 'Thakurmunda', 'MBJ-THA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rasgovindpur", "School Code": "OAV-MBJ-009", "Block": "Rasgovindpur", "Address": "At/Po - Rasgovindpur, Dist - Mayurbhanj, Odisha", "PIN Code": "757091", "Principal Name": "Shri Ghanashyam Murmu", "Principal Phone": "+91-9437204009"}',
 'Mayurbhanj', 'Rasgovindpur', 'MBJ-RAS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jashipur", "School Code": "OAV-MBJ-010", "Block": "Jashipur", "Address": "At/Po - Jashipur, Dist - Mayurbhanj, Odisha", "PIN Code": "757091", "Principal Name": "Shri Ashutosh Hembram", "Principal Phone": "+91-9437204010"}',
 'Mayurbhanj', 'Jashipur', 'MBJ-JAS'),

-- Sundargarh District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rourkela", "School Code": "OAV-SGH-001", "Block": "Rourkela", "Address": "At/Po - Rourkela, Dist - Sundargarh, Odisha", "PIN Code": "769001", "Principal Name": "Shri Chandra Sekhar Mishra", "Principal Phone": "+91-9437205001"}',
 'Sundargarh', 'Rourkela', 'SGH-RKL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rajgangpur", "School Code": "OAV-SGH-002", "Block": "Rajgangpur", "Address": "At/Po - Rajgangpur, Dist - Sundargarh, Odisha", "PIN Code": "770017", "Principal Name": "Smt. Annapurna Patel", "Principal Phone": "+91-9437205002"}',
 'Sundargarh', 'Rajgangpur', 'SGH-RJG'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bonai", "School Code": "OAV-SGH-003", "Block": "Bonai", "Address": "At/Po - Bonai, Dist - Sundargarh, Odisha", "PIN Code": "770036", "Principal Name": "Shri Nirmal Kumar Lakra", "Principal Phone": "+91-9437205003"}',
 'Sundargarh', 'Bonai', 'SGH-BON'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Lahunipara", "School Code": "OAV-SGH-004", "Block": "Lahunipara", "Address": "At/Po - Lahunipara, Dist - Sundargarh, Odisha", "PIN Code": "770040", "Principal Name": "Shri Ajay Kumar Kisan", "Principal Phone": "+91-9437205004"}',
 'Sundargarh', 'Lahunipara', 'SGH-LAH'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kuarmunda", "School Code": "OAV-SGH-005", "Block": "Kuarmunda", "Address": "At/Po - Kuarmunda, Dist - Sundargarh, Odisha", "PIN Code": "770039", "Principal Name": "Smt. Jyotsna Minz", "Principal Phone": "+91-9437205005"}',
 'Sundargarh', 'Kuarmunda', 'SGH-KUA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bisra", "School Code": "OAV-SGH-006", "Block": "Bisra", "Address": "At/Po - Bisra, Dist - Sundargarh, Odisha", "PIN Code": "770001", "Principal Name": "Shri Dinabandhu Toppo", "Principal Phone": "+91-9437205006"}',
 'Sundargarh', 'Bisra', 'SGH-BIS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Lephripara", "School Code": "OAV-SGH-007", "Block": "Lephripara", "Address": "At/Po - Lephripara, Dist - Sundargarh, Odisha", "PIN Code": "770070", "Principal Name": "Shri Bimal Kerketta", "Principal Phone": "+91-9437205007"}',
 'Sundargarh', 'Lephripara', 'SGH-LEP'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Nuagaon", "School Code": "OAV-SGH-008", "Block": "Nuagaon", "Address": "At/Po - Nuagaon, Dist - Sundargarh, Odisha", "PIN Code": "770023", "Principal Name": "Smt. Sarita Ekka", "Principal Phone": "+91-9437205008"}',
 'Sundargarh', 'Nuagaon', 'SGH-NUA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Tangarpali", "School Code": "OAV-SGH-009", "Block": "Tangarpali", "Address": "At/Po - Tangarpali, Dist - Sundargarh, Odisha", "PIN Code": "770002", "Principal Name": "Shri Suresh Chandra Nayak", "Principal Phone": "+91-9437205009"}',
 'Sundargarh', 'Tangarpali', 'SGH-TAN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Subdega", "School Code": "OAV-SGH-010", "Block": "Subdega", "Address": "At/Po - Subdega, Dist - Sundargarh, Odisha", "PIN Code": "770022", "Principal Name": "Shri Judhisthir Kujur", "Principal Phone": "+91-9437205010"}',
 'Sundargarh', 'Subdega', 'SGH-SUB'),

-- Balasore District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Baliapal", "School Code": "OAV-BLS-001", "Block": "Baliapal", "Address": "At/Po - Baliapal, Dist - Balasore, Odisha", "PIN Code": "756026", "Principal Name": "Shri Bhagirathi Jena", "Principal Phone": "+91-9437206001"}',
 'Balasore', 'Baliapal', 'BLS-BAL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Basta", "School Code": "OAV-BLS-002", "Block": "Basta", "Address": "At/Po - Basta, Dist - Balasore, Odisha", "PIN Code": "756029", "Principal Name": "Smt. Minati Mohapatra", "Principal Phone": "+91-9437206002"}',
 'Balasore', 'Basta', 'BLS-BAS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jaleswar", "School Code": "OAV-BLS-003", "Block": "Jaleswar", "Address": "At/Po - Jaleswar, Dist - Balasore, Odisha", "PIN Code": "756032", "Principal Name": "Shri Rajendra Prasad Das", "Principal Phone": "+91-9437206003"}',
 'Balasore', 'Jaleswar', 'BLS-JAL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Nilgiri", "School Code": "OAV-BLS-004", "Block": "Nilgiri", "Address": "At/Po - Nilgiri, Dist - Balasore, Odisha", "PIN Code": "756040", "Principal Name": "Shri Umakanta Panda", "Principal Phone": "+91-9437206004"}',
 'Balasore', 'Nilgiri', 'BLS-NIL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Remuna", "School Code": "OAV-BLS-005", "Block": "Remuna", "Address": "At/Po - Remuna, Dist - Balasore, Odisha", "PIN Code": "756019", "Principal Name": "Smt. Sarojini Rout", "Principal Phone": "+91-9437206005"}',
 'Balasore', 'Remuna', 'BLS-REM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Simulia", "School Code": "OAV-BLS-006", "Block": "Simulia", "Address": "At/Po - Simulia, Dist - Balasore, Odisha", "PIN Code": "756021", "Principal Name": "Shri Akshya Kumar Rout", "Principal Phone": "+91-9437206006"}',
 'Balasore', 'Simulia', 'BLS-SIM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Soro", "School Code": "OAV-BLS-007", "Block": "Soro", "Address": "At/Po - Soro, Dist - Balasore, Odisha", "PIN Code": "756045", "Principal Name": "Shri Prafulla Kumar Nayak", "Principal Phone": "+91-9437206007"}',
 'Balasore', 'Soro', 'BLS-SOR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Khaira", "School Code": "OAV-BLS-008", "Block": "Khaira", "Address": "At/Po - Khaira, Dist - Balasore, Odisha", "PIN Code": "756042", "Principal Name": "Shri Brundaban Mallick", "Principal Phone": "+91-9437206008"}',
 'Balasore', 'Khaira', 'BLS-KHA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Oupada", "School Code": "OAV-BLS-009", "Block": "Oupada", "Address": "At/Po - Oupada, Dist - Balasore, Odisha", "PIN Code": "756047", "Principal Name": "Smt. Sanghamitra Parida", "Principal Phone": "+91-9437206009"}',
 'Balasore', 'Oupada', 'BLS-OUP'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bhograi", "School Code": "OAV-BLS-010", "Block": "Bhograi", "Address": "At/Po - Bhograi, Dist - Balasore, Odisha", "PIN Code": "756036", "Principal Name": "Shri Trinath Rout", "Principal Phone": "+91-9437206010"}',
 'Balasore', 'Bhograi', 'BLS-BHO'),

-- Sambalpur District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dhankauda", "School Code": "OAV-SBP-001", "Block": "Dhankauda", "Address": "At/Po - Dhankauda, Dist - Sambalpur, Odisha", "PIN Code": "768006", "Principal Name": "Shri Saroj Kumar Meher", "Principal Phone": "+91-9437207001"}',
 'Sambalpur', 'Dhankauda', 'SBP-DHK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jamankira", "School Code": "OAV-SBP-002", "Block": "Jamankira", "Address": "At/Po - Jamankira, Dist - Sambalpur, Odisha", "PIN Code": "768113", "Principal Name": "Smt. Mamata Bag", "Principal Phone": "+91-9437207002"}',
 'Sambalpur', 'Jamankira', 'SBP-JAM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jujomura", "School Code": "OAV-SBP-003", "Block": "Jujomura", "Address": "At/Po - Jujomura, Dist - Sambalpur, Odisha", "PIN Code": "768105", "Principal Name": "Shri Laxmidhar Sahu", "Principal Phone": "+91-9437207003"}',
 'Sambalpur', 'Jujomura', 'SBP-JUJ'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kuchinda", "School Code": "OAV-SBP-004", "Block": "Kuchinda", "Address": "At/Po - Kuchinda, Dist - Sambalpur, Odisha", "PIN Code": "768222", "Principal Name": "Shri Akshya Kumar Bhoi", "Principal Phone": "+91-9437207004"}',
 'Sambalpur', 'Kuchinda', 'SBP-KUC'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Maneswar", "School Code": "OAV-SBP-005", "Block": "Maneswar", "Address": "At/Po - Maneswar, Dist - Sambalpur, Odisha", "PIN Code": "768112", "Principal Name": "Smt. Sulochana Naik", "Principal Phone": "+91-9437207005"}',
 'Sambalpur', 'Maneswar', 'SBP-MAN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Naktideul", "School Code": "OAV-SBP-006", "Block": "Naktideul", "Address": "At/Po - Naktideul, Dist - Sambalpur, Odisha", "PIN Code": "768107", "Principal Name": "Shri Purna Chandra Sahu", "Principal Phone": "+91-9437207006"}',
 'Sambalpur', 'Naktideul', 'SBP-NAK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rairakhol", "School Code": "OAV-SBP-007", "Block": "Rairakhol", "Address": "At/Po - Rairakhol, Dist - Sambalpur, Odisha", "PIN Code": "768115", "Principal Name": "Shri Jadunath Pradhan", "Principal Phone": "+91-9437207007"}',
 'Sambalpur', 'Rairakhol', 'SBP-RAI'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bamra", "School Code": "OAV-SBP-008", "Block": "Bamra", "Address": "At/Po - Bamra, Dist - Sambalpur, Odisha", "PIN Code": "768221", "Principal Name": "Smt. Sushama Naik", "Principal Phone": "+91-9437207008"}',
 'Sambalpur', 'Bamra', 'SBP-BAM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rengali", "School Code": "OAV-SBP-009", "Block": "Rengali", "Address": "At/Po - Rengali, Dist - Sambalpur, Odisha", "PIN Code": "768106", "Principal Name": "Shri Govinda Sahu", "Principal Phone": "+91-9437207009"}',
 'Sambalpur', 'Rengali', 'SBP-REN'),

-- Koraput District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jeypore", "School Code": "OAV-KPT-001", "Block": "Jeypore", "Address": "At/Po - Jeypore, Dist - Koraput, Odisha", "PIN Code": "764001", "Principal Name": "Shri Surya Narayan Pattnaik", "Principal Phone": "+91-9437208001"}',
 'Koraput', 'Jeypore', 'KPT-JEY'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Koraput", "School Code": "OAV-KPT-002", "Block": "Koraput", "Address": "At/Po - Koraput, Dist - Koraput, Odisha", "PIN Code": "764020", "Principal Name": "Smt. Bijaylaxmi Rath", "Principal Phone": "+91-9437208002"}',
 'Koraput', 'Koraput', 'KPT-KOR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Laxmipur", "School Code": "OAV-KPT-003", "Block": "Laxmipur", "Address": "At/Po - Laxmipur, Dist - Koraput, Odisha", "PIN Code": "764081", "Principal Name": "Shri Trilochan Sabar", "Principal Phone": "+91-9437208003"}',
 'Koraput', 'Laxmipur', 'KPT-LAX'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Nandapur", "School Code": "OAV-KPT-004", "Block": "Nandapur", "Address": "At/Po - Nandapur, Dist - Koraput, Odisha", "PIN Code": "764036", "Principal Name": "Shri Krushna Chandra Khara", "Principal Phone": "+91-9437208004"}',
 'Koraput', 'Nandapur', 'KPT-NAN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Semiliguda", "School Code": "OAV-KPT-005", "Block": "Semiliguda", "Address": "At/Po - Semiliguda, Dist - Koraput, Odisha", "PIN Code": "764036", "Principal Name": "Smt. Pramila Majhi", "Principal Phone": "+91-9437208005"}',
 'Koraput', 'Semiliguda', 'KPT-SEM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Boipariguda", "School Code": "OAV-KPT-006", "Block": "Boipariguda", "Address": "At/Po - Boipariguda, Dist - Koraput, Odisha", "PIN Code": "764043", "Principal Name": "Shri Madhab Chandra Gouda", "Principal Phone": "+91-9437208006"}',
 'Koraput', 'Boipariguda', 'KPT-BOI'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Borigumma", "School Code": "OAV-KPT-007", "Block": "Borigumma", "Address": "At/Po - Borigumma, Dist - Koraput, Odisha", "PIN Code": "764056", "Principal Name": "Shri Jagannath Harijan", "Principal Phone": "+91-9437208007"}',
 'Koraput', 'Borigumma', 'KPT-BOR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dasmantpur", "School Code": "OAV-KPT-008", "Block": "Dasmantpur", "Address": "At/Po - Dasmantpur, Dist - Koraput, Odisha", "PIN Code": "764028", "Principal Name": "Smt. Sasmita Santa", "Principal Phone": "+91-9437208008"}',
 'Koraput', 'Dasmantpur', 'KPT-DAS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Pottangi", "School Code": "OAV-KPT-009", "Block": "Pottangi", "Address": "At/Po - Pottangi, Dist - Koraput, Odisha", "PIN Code": "764039", "Principal Name": "Shri Debendra Khillo", "Principal Phone": "+91-9437208009"}',
 'Koraput', 'Pottangi', 'KPT-POT'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Lamtaput", "School Code": "OAV-KPT-010", "Block": "Lamtaput", "Address": "At/Po - Lamtaput, Dist - Koraput, Odisha", "PIN Code": "764081", "Principal Name": "Shri Banamali Khara", "Principal Phone": "+91-9437208010"}',
 'Koraput', 'Lamtaput', 'KPT-LAM'),

-- Kalahandi District (10 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bhawanipatna", "School Code": "OAV-KLH-001", "Block": "Bhawanipatna", "Address": "At/Po - Bhawanipatna, Dist - Kalahandi, Odisha", "PIN Code": "766001", "Principal Name": "Shri Satyabrata Mishra", "Principal Phone": "+91-9437209001"}',
 'Kalahandi', 'Bhawanipatna', 'KLH-BHW'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dharmagarh", "School Code": "OAV-KLH-002", "Block": "Dharmagarh", "Address": "At/Po - Dharmagarh, Dist - Kalahandi, Odisha", "PIN Code": "766015", "Principal Name": "Smt. Saudamini Sahu", "Principal Phone": "+91-9437209002"}',
 'Kalahandi', 'Dharmagarh', 'KLH-DHA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Junagarh", "School Code": "OAV-KLH-003", "Block": "Junagarh", "Address": "At/Po - Junagarh, Dist - Kalahandi, Odisha", "PIN Code": "766014", "Principal Name": "Shri Manas Ranjan Bag", "Principal Phone": "+91-9437209003"}',
 'Kalahandi', 'Junagarh', 'KLH-JUN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kesinga", "School Code": "OAV-KLH-004", "Block": "Kesinga", "Address": "At/Po - Kesinga, Dist - Kalahandi, Odisha", "PIN Code": "766012", "Principal Name": "Shri Jayanta Kumar Pradhan", "Principal Phone": "+91-9437209004"}',
 'Kalahandi', 'Kesinga', 'KLH-KES'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Lanjigarh", "School Code": "OAV-KLH-005", "Block": "Lanjigarh", "Address": "At/Po - Lanjigarh, Dist - Kalahandi, Odisha", "PIN Code": "766027", "Principal Name": "Smt. Priyambada Sahu", "Principal Phone": "+91-9437209005"}',
 'Kalahandi', 'Lanjigarh', 'KLH-LAN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Thuamul Rampur", "School Code": "OAV-KLH-006", "Block": "Thuamul Rampur", "Address": "At/Po - Thuamul Rampur, Dist - Kalahandi, Odisha", "PIN Code": "766037", "Principal Name": "Shri Bhaskar Majhi", "Principal Phone": "+91-9437209006"}',
 'Kalahandi', 'Thuamul Rampur', 'KLH-THU'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Golamunda", "School Code": "OAV-KLH-007", "Block": "Golamunda", "Address": "At/Po - Golamunda, Dist - Kalahandi, Odisha", "PIN Code": "766017", "Principal Name": "Shri Nityananda Sahu", "Principal Phone": "+91-9437209007"}',
 'Kalahandi', 'Golamunda', 'KLH-GOL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Karlamunda", "School Code": "OAV-KLH-008", "Block": "Karlamunda", "Address": "At/Po - Karlamunda, Dist - Kalahandi, Odisha", "PIN Code": "766018", "Principal Name": "Smt. Rajani Bag", "Principal Phone": "+91-9437209008"}',
 'Kalahandi', 'Karlamunda', 'KLH-KAR'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kokasara", "School Code": "OAV-KLH-009", "Block": "Kokasara", "Address": "At/Po - Kokasara, Dist - Kalahandi, Odisha", "PIN Code": "766019", "Principal Name": "Shri Madan Mohan Pradhan", "Principal Phone": "+91-9437209009"}',
 'Kalahandi', 'Kokasara', 'KLH-KOK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jaipatna", "School Code": "OAV-KLH-010", "Block": "Jaipatna", "Address": "At/Po - Jaipatna, Dist - Kalahandi, Odisha", "PIN Code": "766023", "Principal Name": "Shri Lingaraj Sahu", "Principal Phone": "+91-9437209010"}',
 'Kalahandi', 'Jaipatna', 'KLH-JAI'),

-- Puri District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Nimapara", "School Code": "OAV-PRI-001", "Block": "Nimapara", "Address": "At/Po - Nimapara, Dist - Puri, Odisha", "PIN Code": "752106", "Principal Name": "Shri Niranjan Parida", "Principal Phone": "+91-9437210001"}',
 'Puri', 'Nimapara', 'PRI-NIM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Pipili", "School Code": "OAV-PRI-002", "Block": "Pipili", "Address": "At/Po - Pipili, Dist - Puri, Odisha", "PIN Code": "752104", "Principal Name": "Smt. Subhadra Sahu", "Principal Phone": "+91-9437210002"}',
 'Puri', 'Pipili', 'PRI-PIP'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Satyabadi", "School Code": "OAV-PRI-003", "Block": "Satyabadi", "Address": "At/Po - Satyabadi, Dist - Puri, Odisha", "PIN Code": "752110", "Principal Name": "Shri Braja Kishore Dash", "Principal Phone": "+91-9437210003"}',
 'Puri', 'Satyabadi', 'PRI-SAT'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kakatpur", "School Code": "OAV-PRI-004", "Block": "Kakatpur", "Address": "At/Po - Kakatpur, Dist - Puri, Odisha", "PIN Code": "752108", "Principal Name": "Shri Durga Prasad Swain", "Principal Phone": "+91-9437210004"}',
 'Puri', 'Kakatpur', 'PRI-KAK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Krushnaprasad", "School Code": "OAV-PRI-005", "Block": "Krushnaprasad", "Address": "At/Po - Krushnaprasad, Dist - Puri, Odisha", "PIN Code": "752109", "Principal Name": "Smt. Puspalata Behera", "Principal Phone": "+91-9437210005"}',
 'Puri', 'Krushnaprasad', 'PRI-KRU'),

-- Angul District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Angul", "School Code": "OAV-ANG-001", "Block": "Angul", "Address": "At/Po - Angul, Dist - Angul, Odisha", "PIN Code": "759122", "Principal Name": "Shri Subash Chandra Sahu", "Principal Phone": "+91-9437211001"}',
 'Angul', 'Angul', 'ANG-ANG'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Talcher", "School Code": "OAV-ANG-002", "Block": "Talcher", "Address": "At/Po - Talcher, Dist - Angul, Odisha", "PIN Code": "759100", "Principal Name": "Smt. Suchitra Mishra", "Principal Phone": "+91-9437211002"}',
 'Angul', 'Talcher', 'ANG-TAL'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Athmallik", "School Code": "OAV-ANG-003", "Block": "Athmallik", "Address": "At/Po - Athmallik, Dist - Angul, Odisha", "PIN Code": "759128", "Principal Name": "Shri Prashant Kumar Nayak", "Principal Phone": "+91-9437211003"}',
 'Angul', 'Athmallik', 'ANG-ATH'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Banarpal", "School Code": "OAV-ANG-004", "Block": "Banarpal", "Address": "At/Po - Banarpal, Dist - Angul, Odisha", "PIN Code": "759128", "Principal Name": "Shri Manoranjan Behera", "Principal Phone": "+91-9437211004"}',
 'Angul', 'Banarpal', 'ANG-BAN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Chhendipada", "School Code": "OAV-ANG-005", "Block": "Chhendipada", "Address": "At/Po - Chhendipada, Dist - Angul, Odisha", "PIN Code": "759118", "Principal Name": "Smt. Bijayini Sahu", "Principal Phone": "+91-9437211005"}',
 'Angul', 'Chhendipada', 'ANG-CHH'),

-- Keonjhar District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Keonjhar", "School Code": "OAV-KJR-001", "Block": "Keonjhar", "Address": "At/Po - Keonjhar, Dist - Keonjhar, Odisha", "PIN Code": "758001", "Principal Name": "Shri Raghunath Mohapatra", "Principal Phone": "+91-9437212001"}',
 'Keonjhar', 'Keonjhar', 'KJR-KEO'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Anandapur", "School Code": "OAV-KJR-002", "Block": "Anandapur", "Address": "At/Po - Anandapur, Dist - Keonjhar, Odisha", "PIN Code": "758021", "Principal Name": "Smt. Sucheta Behera", "Principal Phone": "+91-9437212002"}',
 'Keonjhar', 'Anandapur', 'KJR-ANA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Champua", "School Code": "OAV-KJR-003", "Block": "Champua", "Address": "At/Po - Champua, Dist - Keonjhar, Odisha", "PIN Code": "758041", "Principal Name": "Shri Jagannath Nayak", "Principal Phone": "+91-9437212003"}',
 'Keonjhar', 'Champua', 'KJR-CHA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jhumpura", "School Code": "OAV-KJR-004", "Block": "Jhumpura", "Address": "At/Po - Jhumpura, Dist - Keonjhar, Odisha", "PIN Code": "758025", "Principal Name": "Shri Bishnu Charan Sethi", "Principal Phone": "+91-9437212004"}',
 'Keonjhar', 'Jhumpura', 'KJR-JHU'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Patna", "School Code": "OAV-KJR-005", "Block": "Patna", "Address": "At/Po - Patna, Dist - Keonjhar, Odisha", "PIN Code": "758035", "Principal Name": "Smt. Gita Rani Sahu", "Principal Phone": "+91-9437212005"}',
 'Keonjhar', 'Patna', 'KJR-PAT'),

-- Jajpur District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Jajpur", "School Code": "OAV-JJP-001", "Block": "Jajpur", "Address": "At/Po - Jajpur, Dist - Jajpur, Odisha", "PIN Code": "755001", "Principal Name": "Shri Satyanarayan Pati", "Principal Phone": "+91-9437213001"}',
 'Jajpur', 'Jajpur', 'JJP-JAJ'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Binjharpur", "School Code": "OAV-JJP-002", "Block": "Binjharpur", "Address": "At/Po - Binjharpur, Dist - Jajpur, Odisha", "PIN Code": "755004", "Principal Name": "Smt. Madhusmita Pattnaik", "Principal Phone": "+91-9437213002"}',
 'Jajpur', 'Binjharpur', 'JJP-BIN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dasarathpur", "School Code": "OAV-JJP-003", "Block": "Dasarathpur", "Address": "At/Po - Dasarathpur, Dist - Jajpur, Odisha", "PIN Code": "755006", "Principal Name": "Shri Bijay Kumar Sahoo", "Principal Phone": "+91-9437213003"}',
 'Jajpur', 'Dasarathpur', 'JJP-DAS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dharmasala", "School Code": "OAV-JJP-004", "Block": "Dharmasala", "Address": "At/Po - Dharmasala, Dist - Jajpur, Odisha", "PIN Code": "755007", "Principal Name": "Shri Purna Chandra Nayak", "Principal Phone": "+91-9437213004"}',
 'Jajpur', 'Dharmasala', 'JJP-DHA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Koraigora", "School Code": "OAV-JJP-005", "Block": "Koraigora", "Address": "At/Po - Koraigora, Dist - Jajpur, Odisha", "PIN Code": "755009", "Principal Name": "Smt. Swarnalata Mohanty", "Principal Phone": "+91-9437213005"}',
 'Jajpur', 'Koraigora', 'JJP-KOR'),

-- Dhenkanal District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Dhenkanal", "School Code": "OAV-DKL-001", "Block": "Dhenkanal Sadar", "Address": "At/Po - Dhenkanal, Dist - Dhenkanal, Odisha", "PIN Code": "759001", "Principal Name": "Shri Baikuntha Nath Sahoo", "Principal Phone": "+91-9437214001"}',
 'Dhenkanal', 'Dhenkanal Sadar', 'DKL-DHK'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Hindol", "School Code": "OAV-DKL-002", "Block": "Hindol", "Address": "At/Po - Hindol, Dist - Dhenkanal, Odisha", "PIN Code": "759022", "Principal Name": "Smt. Sanjukta Parida", "Principal Phone": "+91-9437214002"}',
 'Dhenkanal', 'Hindol', 'DKL-HIN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Kamakhyanagar", "School Code": "OAV-DKL-003", "Block": "Kamakhyanagar", "Address": "At/Po - Kamakhyanagar, Dist - Dhenkanal, Odisha", "PIN Code": "759018", "Principal Name": "Shri Narayan Chandra Sahu", "Principal Phone": "+91-9437214003"}',
 'Dhenkanal', 'Kamakhyanagar', 'DKL-KAM'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Odapada", "School Code": "OAV-DKL-004", "Block": "Odapada", "Address": "At/Po - Odapada, Dist - Dhenkanal, Odisha", "PIN Code": "759015", "Principal Name": "Shri Prakash Chandra Behera", "Principal Phone": "+91-9437214004"}',
 'Dhenkanal', 'Odapada', 'DKL-ODA'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Parjang", "School Code": "OAV-DKL-005", "Block": "Parjang", "Address": "At/Po - Parjang, Dist - Dhenkanal, Odisha", "PIN Code": "759017", "Principal Name": "Smt. Gitanjali Sahoo", "Principal Phone": "+91-9437214005"}',
 'Dhenkanal', 'Parjang', 'DKL-PAR'),

-- Rayagada District (5 schools)
((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Rayagada", "School Code": "OAV-RGD-001", "Block": "Rayagada", "Address": "At/Po - Rayagada, Dist - Rayagada, Odisha", "PIN Code": "765001", "Principal Name": "Shri Dilip Kumar Patra", "Principal Phone": "+91-9437215001"}',
 'Rayagada', 'Rayagada', 'RGD-RAY'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Gunupur", "School Code": "OAV-RGD-002", "Block": "Gunupur", "Address": "At/Po - Gunupur, Dist - Rayagada, Odisha", "PIN Code": "765022", "Principal Name": "Smt. Kalyani Gomango", "Principal Phone": "+91-9437215002"}',
 'Rayagada', 'Gunupur', 'RGD-GUN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Bissamcuttack", "School Code": "OAV-RGD-003", "Block": "Bissamcuttack", "Address": "At/Po - Bissamcuttack, Dist - Rayagada, Odisha", "PIN Code": "765019", "Principal Name": "Shri Rabi Narayan Jhodia", "Principal Phone": "+91-9437215003"}',
 'Rayagada', 'Bissamcuttack', 'RGD-BIS'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Muniguda", "School Code": "OAV-RGD-004", "Block": "Muniguda", "Address": "At/Po - Muniguda, Dist - Rayagada, Odisha", "PIN Code": "765020", "Principal Name": "Shri Trinath Miniaka", "Principal Phone": "+91-9437215004"}',
 'Rayagada', 'Muniguda', 'RGD-MUN'),

((SELECT id FROM address_list_configs WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND address_type = 'ship_to'),
 '{"School Name": "OAV Padmapur", "School Code": "OAV-RGD-005", "Block": "Padmapur", "Address": "At/Po - Padmapur, Dist - Rayagada, Odisha", "PIN Code": "765025", "Principal Name": "Smt. Sujata Sabar", "Principal Phone": "+91-9437215005"}',
 'Rayagada', 'Padmapur', 'RGD-PAD');

-- ============================================================
-- DC TEMPLATES (3 templates)
-- ============================================================
INSERT INTO dc_templates (project_id, name, purpose) VALUES
((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Full Smart Classroom Kit',
 'Complete smart classroom setup including IFP display, OPS PC, document camera, wireless presenter, speakers, UPS, cabling, WiFi AP, teacher tablet, and wall mount'),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Display & Computing Unit',
 'Core display and computing components - IFP display, OPS PC module, wall mount bracket, and UPS'),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Peripherals & Networking Kit',
 'Peripheral devices and networking equipment - document camera, wireless presenter, speakers, WiFi AP, cabling kit, and teacher tablet');

-- Template 1: Full Smart Classroom Kit (all 10 products)
INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order)
SELECT
    (SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Full Smart Classroom Kit'),
    p.id, 1, ROW_NUMBER() OVER (ORDER BY p.id)
FROM products p
WHERE p.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Template 2: Display & Computing Unit (IFP, OPS PC, UPS, Wall Mount)
INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order) VALUES
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Display & Computing Unit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Interactive Flat Panel%'), 1, 1),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Display & Computing Unit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Mini PC%'), 1, 2),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Display & Computing Unit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'UPS%'), 1, 3),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Display & Computing Unit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Wall Mount%'), 1, 4);

-- Template 3: Peripherals & Networking Kit
INSERT INTO dc_template_products (template_id, product_id, default_quantity, sort_order) VALUES
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Document Camera%'), 1, 1),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Wireless Presentation%'), 1, 2),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Ceiling Mounted%'), 1, 3),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Structured Cabling%'), 1, 4),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Wireless Access%'), 1, 5),
((SELECT t.id FROM dc_templates t JOIN projects p ON t.project_id = p.id WHERE p.name = 'Smart Classrooms OAVS Odisha' AND t.name = 'Peripherals & Networking Kit'),
 (SELECT id FROM products WHERE project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha') AND item_name LIKE 'Teacher Tablet%'), 1, 6);

-- ============================================================
-- TRANSPORTERS (10 transport companies based in Odisha)
-- ============================================================
INSERT INTO transporters (project_id, company_name, contact_person, phone, gst_number, is_active) VALUES
((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Odisha Express Logistics Pvt Ltd', 'Shri Bikash Mohanty', '+91-674-2563401', '21AABCO1234E1Z5', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Eastern Freight Carriers', 'Shri Rajesh Pattnaik', '+91-674-2540234', '21AABCE5678F2Z6', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Kalinga Transport Corporation', 'Shri Sushil Kumar Das', '+91-671-2345678', '21AABCK9012G3Z7', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Mahanadi Logistics Solutions', 'Smt. Rekha Mishra', '+91-663-2541890', '21AABCM3456H4Z8', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Utkal Road Transport', 'Shri Pramod Kumar Swain', '+91-6782-234567', '21AABCU7890I5Z9', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Blue Dart Express - Bhubaneswar', 'Shri Asit Ranjan Sahoo', '+91-674-2300456', '21AABCB1234J6Z0', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Jagannath Freight Services', 'Shri Durga Prasad Nayak', '+91-674-2567123', '21AABCJ5678K7Z1', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Hirakud Roadways', 'Shri Biswanath Meher', '+91-663-2530789', '21AABCH9012L8Z2', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Coastal Cargo Movers', 'Shri Hemanta Kumar Rout', '+91-680-2234567', '21AABCC3456M9Z3', 1),

((SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha'),
 'Sambalpur Goods Transport', 'Shri Nilambar Bhoi', '+91-663-2542345', '21AABCS7890N0Z4', 1);

-- ============================================================
-- VEHICLES (10 per transporter = 100 vehicles total)
-- ============================================================

-- Transporter 1: Odisha Express Logistics
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-02-AB-1001' AS vehicle_number, 'truck' AS vehicle_type, 'Ramesh Sahoo' AS driver_name, '+91-9437301001' AS driver_phone1, '+91-8337401001' AS driver_phone2 UNION ALL
    SELECT 'OD-02-AB-1002', 'truck', 'Suresh Barik', '+91-9437301002', '' UNION ALL
    SELECT 'OD-02-CD-1003', 'mini_truck', 'Ganesh Pradhan', '+91-9437301003', '+91-8337401003' UNION ALL
    SELECT 'OD-02-CD-1004', 'mini_truck', 'Mahesh Jena', '+91-9437301004', '' UNION ALL
    SELECT 'OD-02-EF-1005', 'truck', 'Dinesh Mohanty', '+91-9437301005', '+91-8337401005' UNION ALL
    SELECT 'OD-02-EF-1006', 'container', 'Nagesh Behera', '+91-9437301006', '' UNION ALL
    SELECT 'OD-02-GH-1007', 'truck', 'Rakesh Swain', '+91-9437301007', '+91-8337401007' UNION ALL
    SELECT 'OD-02-GH-1008', 'mini_truck', 'Lokesh Parida', '+91-9437301008', '' UNION ALL
    SELECT 'OD-02-IJ-1009', 'truck', 'Jayesh Nayak', '+91-9437301009', '+91-8337401009' UNION ALL
    SELECT 'OD-02-IJ-1010', 'container', 'Hitesh Das', '+91-9437301010', ''
) v
WHERE t.company_name = 'Odisha Express Logistics Pvt Ltd'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 2: Eastern Freight Carriers
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-02-KL-2001' AS vehicle_number, 'truck' AS vehicle_type, 'Biswanath Rout' AS driver_name, '+91-9437302001' AS driver_phone1, '' AS driver_phone2 UNION ALL
    SELECT 'OD-02-KL-2002', 'truck', 'Prashant Mishra', '+91-9437302002', '+91-8337402002' UNION ALL
    SELECT 'OD-02-MN-2003', 'mini_truck', 'Santosh Panda', '+91-9437302003', '' UNION ALL
    SELECT 'OD-02-MN-2004', 'mini_truck', 'Ashok Sahoo', '+91-9437302004', '+91-8337402004' UNION ALL
    SELECT 'OD-02-OP-2005', 'truck', 'Kishore Mohapatra', '+91-9437302005', '' UNION ALL
    SELECT 'OD-02-OP-2006', 'container', 'Manoj Lenka', '+91-9437302006', '+91-8337402006' UNION ALL
    SELECT 'OD-02-QR-2007', 'truck', 'Tapan Behera', '+91-9437302007', '' UNION ALL
    SELECT 'OD-02-QR-2008', 'mini_truck', 'Deepak Sahu', '+91-9437302008', '+91-8337402008' UNION ALL
    SELECT 'OD-02-ST-2009', 'truck', 'Sarat Jena', '+91-9437302009', '' UNION ALL
    SELECT 'OD-02-ST-2010', 'container', 'Bikram Nayak', '+91-9437302010', ''
) v
WHERE t.company_name = 'Eastern Freight Carriers'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 3: Kalinga Transport Corporation
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-05-AB-3001' AS vehicle_number, 'truck' AS vehicle_type, 'Ranjit Swain' AS driver_name, '+91-9437303001' AS driver_phone1, '+91-8337403001' AS driver_phone2 UNION ALL
    SELECT 'OD-05-AB-3002', 'truck', 'Ajit Parida', '+91-9437303002', '' UNION ALL
    SELECT 'OD-05-CD-3003', 'mini_truck', 'Sujit Pattnaik', '+91-9437303003', '+91-8337403003' UNION ALL
    SELECT 'OD-05-CD-3004', 'mini_truck', 'Rohit Dash', '+91-9437303004', '' UNION ALL
    SELECT 'OD-05-EF-3005', 'truck', 'Amit Behera', '+91-9437303005', '+91-8337403005' UNION ALL
    SELECT 'OD-05-EF-3006', 'container', 'Sumit Rout', '+91-9437303006', '' UNION ALL
    SELECT 'OD-05-GH-3007', 'truck', 'Lalit Sahoo', '+91-9437303007', '+91-8337403007' UNION ALL
    SELECT 'OD-05-GH-3008', 'mini_truck', 'Mohit Mohanty', '+91-9437303008', '' UNION ALL
    SELECT 'OD-05-IJ-3009', 'truck', 'Vinit Jena', '+91-9437303009', '+91-8337403009' UNION ALL
    SELECT 'OD-05-IJ-3010', 'container', 'Ankit Pradhan', '+91-9437303010', ''
) v
WHERE t.company_name = 'Kalinga Transport Corporation'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 4: Mahanadi Logistics Solutions
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-15-AB-4001' AS vehicle_number, 'truck' AS vehicle_type, 'Krushna Meher' AS driver_name, '+91-9437304001' AS driver_phone1, '' AS driver_phone2 UNION ALL
    SELECT 'OD-15-AB-4002', 'truck', 'Basant Bag', '+91-9437304002', '+91-8337404002' UNION ALL
    SELECT 'OD-15-CD-4003', 'mini_truck', 'Deben Naik', '+91-9437304003', '' UNION ALL
    SELECT 'OD-15-CD-4004', 'mini_truck', 'Fakir Sahu', '+91-9437304004', '+91-8337404004' UNION ALL
    SELECT 'OD-15-EF-4005', 'truck', 'Gopal Bhoi', '+91-9437304005', '' UNION ALL
    SELECT 'OD-15-EF-4006', 'container', 'Harish Pradhan', '+91-9437304006', '+91-8337404006' UNION ALL
    SELECT 'OD-15-GH-4007', 'truck', 'Iswar Seth', '+91-9437304007', '' UNION ALL
    SELECT 'OD-15-GH-4008', 'mini_truck', 'Jatin Kumbhar', '+91-9437304008', '+91-8337404008' UNION ALL
    SELECT 'OD-15-IJ-4009', 'truck', 'Kalia Dehuri', '+91-9437304009', '' UNION ALL
    SELECT 'OD-15-IJ-4010', 'container', 'Laxman Agrawal', '+91-9437304010', ''
) v
WHERE t.company_name = 'Mahanadi Logistics Solutions'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 5: Utkal Road Transport
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-03-AB-5001' AS vehicle_number, 'truck' AS vehicle_type, 'Madan Mallick' AS driver_name, '+91-9437305001' AS driver_phone1, '+91-8337405001' AS driver_phone2 UNION ALL
    SELECT 'OD-03-AB-5002', 'truck', 'Narayan Sethi', '+91-9437305002', '' UNION ALL
    SELECT 'OD-03-CD-5003', 'mini_truck', 'Omkar Rath', '+91-9437305003', '+91-8337405003' UNION ALL
    SELECT 'OD-03-CD-5004', 'mini_truck', 'Paresh Mohanty', '+91-9437305004', '' UNION ALL
    SELECT 'OD-03-EF-5005', 'truck', 'Rabindra Sahoo', '+91-9437305005', '+91-8337405005' UNION ALL
    SELECT 'OD-03-EF-5006', 'container', 'Sanjay Jena', '+91-9437305006', '' UNION ALL
    SELECT 'OD-03-GH-5007', 'truck', 'Tapas Behera', '+91-9437305007', '+91-8337405007' UNION ALL
    SELECT 'OD-03-GH-5008', 'mini_truck', 'Umesh Panda', '+91-9437305008', '' UNION ALL
    SELECT 'OD-03-IJ-5009', 'truck', 'Vikram Das', '+91-9437305009', '+91-8337405009' UNION ALL
    SELECT 'OD-03-IJ-5010', 'container', 'Wasim Khan', '+91-9437305010', ''
) v
WHERE t.company_name = 'Utkal Road Transport'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 6: Blue Dart Express - Bhubaneswar
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-02-UV-6001' AS vehicle_number, 'truck' AS vehicle_type, 'Anil Rout' AS driver_name, '+91-9437306001' AS driver_phone1, '' AS driver_phone2 UNION ALL
    SELECT 'OD-02-UV-6002', 'truck', 'Bimal Nayak', '+91-9437306002', '+91-8337406002' UNION ALL
    SELECT 'OD-02-WX-6003', 'mini_truck', 'Chandan Parida', '+91-9437306003', '' UNION ALL
    SELECT 'OD-02-WX-6004', 'mini_truck', 'Dhruba Swain', '+91-9437306004', '+91-8337406004' UNION ALL
    SELECT 'OD-02-YZ-6005', 'truck', 'Ekanath Sahoo', '+91-9437306005', '' UNION ALL
    SELECT 'OD-02-YZ-6006', 'container', 'Fakira Behera', '+91-9437306006', '+91-8337406006' UNION ALL
    SELECT 'OD-02-AA-6007', 'truck', 'Gobinda Mishra', '+91-9437306007', '' UNION ALL
    SELECT 'OD-02-AA-6008', 'mini_truck', 'Harish Jena', '+91-9437306008', '+91-8337406008' UNION ALL
    SELECT 'OD-02-BB-6009', 'truck', 'Indra Mohanty', '+91-9437306009', '' UNION ALL
    SELECT 'OD-02-BB-6010', 'container', 'Jagdish Pattnaik', '+91-9437306010', ''
) v
WHERE t.company_name = 'Blue Dart Express - Bhubaneswar'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 7: Jagannath Freight Services
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-02-CC-7001' AS vehicle_number, 'truck' AS vehicle_type, 'Keshab Lenka' AS driver_name, '+91-9437307001' AS driver_phone1, '+91-8337407001' AS driver_phone2 UNION ALL
    SELECT 'OD-02-CC-7002', 'truck', 'Laxmikant Sahu', '+91-9437307002', '' UNION ALL
    SELECT 'OD-02-DD-7003', 'mini_truck', 'Mrutyunjay Biswal', '+91-9437307003', '+91-8337407003' UNION ALL
    SELECT 'OD-02-DD-7004', 'mini_truck', 'Niranjan Sahoo', '+91-9437307004', '' UNION ALL
    SELECT 'OD-02-EE-7005', 'truck', 'Prafulla Rout', '+91-9437307005', '+91-8337407005' UNION ALL
    SELECT 'OD-02-EE-7006', 'container', 'Rabindra Nayak', '+91-9437307006', '' UNION ALL
    SELECT 'OD-02-FF-7007', 'truck', 'Sachidananda Panda', '+91-9437307007', '+91-8337407007' UNION ALL
    SELECT 'OD-02-FF-7008', 'mini_truck', 'Trinath Behera', '+91-9437307008', '' UNION ALL
    SELECT 'OD-02-GG-7009', 'truck', 'Upendra Das', '+91-9437307009', '+91-8337407009' UNION ALL
    SELECT 'OD-02-GG-7010', 'container', 'Vivekananda Jena', '+91-9437307010', ''
) v
WHERE t.company_name = 'Jagannath Freight Services'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 8: Hirakud Roadways
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-15-KL-8001' AS vehicle_number, 'truck' AS vehicle_type, 'Akshya Meher' AS driver_name, '+91-9437308001' AS driver_phone1, '' AS driver_phone2 UNION ALL
    SELECT 'OD-15-KL-8002', 'truck', 'Bhagaban Bhoi', '+91-9437308002', '+91-8337408002' UNION ALL
    SELECT 'OD-15-MN-8003', 'mini_truck', 'Chaturbhuj Naik', '+91-9437308003', '' UNION ALL
    SELECT 'OD-15-MN-8004', 'mini_truck', 'Damodar Sahu', '+91-9437308004', '+91-8337408004' UNION ALL
    SELECT 'OD-15-OP-8005', 'truck', 'Eshwar Bag', '+91-9437308005', '' UNION ALL
    SELECT 'OD-15-OP-8006', 'container', 'Fakir Mohan Seth', '+91-9437308006', '+91-8337408006' UNION ALL
    SELECT 'OD-15-QR-8007', 'truck', 'Gadadhar Pradhan', '+91-9437308007', '' UNION ALL
    SELECT 'OD-15-QR-8008', 'mini_truck', 'Hrushikesh Kumbhar', '+91-9437308008', '+91-8337408008' UNION ALL
    SELECT 'OD-15-ST-8009', 'truck', 'Ishwar Agrawal', '+91-9437308009', '' UNION ALL
    SELECT 'OD-15-ST-8010', 'container', 'Jugal Kishore Patel', '+91-9437308010', ''
) v
WHERE t.company_name = 'Hirakud Roadways'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 9: Coastal Cargo Movers
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-10-AB-9001' AS vehicle_number, 'truck' AS vehicle_type, 'Kamalakanta Patra' AS driver_name, '+91-9437309001' AS driver_phone1, '+91-8337409001' AS driver_phone2 UNION ALL
    SELECT 'OD-10-AB-9002', 'truck', 'Laxminarayan Dash', '+91-9437309002', '' UNION ALL
    SELECT 'OD-10-CD-9003', 'mini_truck', 'Manas Ranjan Sahu', '+91-9437309003', '+91-8337409003' UNION ALL
    SELECT 'OD-10-CD-9004', 'mini_truck', 'Nirakar Behera', '+91-9437309004', '' UNION ALL
    SELECT 'OD-10-EF-9005', 'truck', 'Padmanabha Sethi', '+91-9437309005', '+91-8337409005' UNION ALL
    SELECT 'OD-10-EF-9006', 'container', 'Rabindranath Swain', '+91-9437309006', '' UNION ALL
    SELECT 'OD-10-GH-9007', 'truck', 'Sadananda Rout', '+91-9437309007', '+91-8337409007' UNION ALL
    SELECT 'OD-10-GH-9008', 'mini_truck', 'Tukuna Parida', '+91-9437309008', '' UNION ALL
    SELECT 'OD-10-IJ-9009', 'truck', 'Udayanath Mohapatra', '+91-9437309009', '+91-8337409009' UNION ALL
    SELECT 'OD-10-IJ-9010', 'container', 'Yudhisthir Jena', '+91-9437309010', ''
) v
WHERE t.company_name = 'Coastal Cargo Movers'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- Transporter 10: Sambalpur Goods Transport
INSERT INTO transporter_vehicles (transporter_id, vehicle_number, vehicle_type, driver_name, driver_phone1, driver_phone2)
SELECT t.id, v.vehicle_number, v.vehicle_type, v.driver_name, v.driver_phone1, v.driver_phone2
FROM transporters t
CROSS JOIN (
    SELECT 'OD-15-UV-0001' AS vehicle_number, 'truck' AS vehicle_type, 'Abinash Sahu' AS driver_name, '+91-9437310001' AS driver_phone1, '' AS driver_phone2 UNION ALL
    SELECT 'OD-15-UV-0002', 'truck', 'Bidyadhar Meher', '+91-9437310002', '+91-8337410002' UNION ALL
    SELECT 'OD-15-WX-0003', 'mini_truck', 'Chintamani Bhoi', '+91-9437310003', '' UNION ALL
    SELECT 'OD-15-WX-0004', 'mini_truck', 'Dasarathi Naik', '+91-9437310004', '+91-8337410004' UNION ALL
    SELECT 'OD-15-YZ-0005', 'truck', 'Eklabya Bag', '+91-9437310005', '' UNION ALL
    SELECT 'OD-15-YZ-0006', 'container', 'Ganeswar Seth', '+91-9437310006', '+91-8337410006' UNION ALL
    SELECT 'OD-15-AA-0007', 'truck', 'Hemant Pradhan', '+91-9437310007', '' UNION ALL
    SELECT 'OD-15-AA-0008', 'mini_truck', 'Ish Kumar Kumbhar', '+91-9437310008', '+91-8337410008' UNION ALL
    SELECT 'OD-15-BB-0009', 'truck', 'Jagabandhu Agrawal', '+91-9437310009', '' UNION ALL
    SELECT 'OD-15-BB-0010', 'container', 'Karunakara Patel', '+91-9437310010', ''
) v
WHERE t.company_name = 'Sambalpur Goods Transport'
AND t.project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha');

-- ============================================================
-- Grant access to admin user and set as last project
-- ============================================================
INSERT INTO user_projects (user_id, project_id)
SELECT u.id, p.id FROM users u, projects p
WHERE u.username = 'admin' AND p.name = 'Smart Classrooms OAVS Odisha';

UPDATE users SET last_project_id = (SELECT id FROM projects WHERE name = 'Smart Classrooms OAVS Odisha')
WHERE username = 'admin';
