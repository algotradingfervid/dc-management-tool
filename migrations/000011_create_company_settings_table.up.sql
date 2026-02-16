CREATE TABLE IF NOT EXISTS company_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name VARCHAR(255) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    state_code VARCHAR(10),
    pincode VARCHAR(10),
    gstin VARCHAR(20),
    signature_image VARCHAR(500),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO company_settings (id, name, address, city, state, state_code, pincode, gstin)
VALUES (
    1,
    'Fervid Smart Solutions Pvt. Ltd.',
    'Plot No 14/2, Dwaraka Park View, 1st Floor, Sector-1, HUDA Techno Enclave, Madhapur',
    'Hyderabad',
    'Telangana',
    '36',
    '500081',
    '36AACCF9742K1Z8'
);
