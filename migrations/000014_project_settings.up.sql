-- Add project settings fields
ALTER TABLE projects ADD COLUMN dispatch_from_address TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN dc_number_format TEXT NOT NULL DEFAULT '{PREFIX}-{TYPE}-{FY}-{SEQ}';
ALTER TABLE projects ADD COLUMN dc_number_separator TEXT NOT NULL DEFAULT '-';
ALTER TABLE projects ADD COLUMN company_email TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN company_cin TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN company_seal_path TEXT;
ALTER TABLE projects ADD COLUMN purpose_text TEXT NOT NULL DEFAULT 'DELIVERED AS PART OF PROJECT EXECUTION';
ALTER TABLE projects ADD COLUMN seq_padding INTEGER NOT NULL DEFAULT 3;
