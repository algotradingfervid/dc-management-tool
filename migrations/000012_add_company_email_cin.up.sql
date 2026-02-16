ALTER TABLE company_settings ADD COLUMN email VARCHAR(255) DEFAULT '';
ALTER TABLE company_settings ADD COLUMN cin VARCHAR(50) DEFAULT '';

UPDATE company_settings SET
    email = 'odishaprojects@fervidsmart.com',
    cin = 'U45100TG2016PTC113752'
WHERE id = 1;
