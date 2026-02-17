-- Performance indexes for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects(created_by);
CREATE INDEX IF NOT EXISTS idx_products_project_id ON products(project_id);
CREATE INDEX IF NOT EXISTS idx_dc_templates_project_id ON dc_templates(project_id);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_project_id ON delivery_challans(project_id);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_status ON delivery_challans(status);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_dc_number ON delivery_challans(dc_number);
CREATE INDEX IF NOT EXISTS idx_delivery_challans_bundle_id ON delivery_challans(bundle_id);
