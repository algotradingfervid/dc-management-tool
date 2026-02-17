-- Add sort_order to dc_template_products for drag-to-reorder
ALTER TABLE dc_template_products ADD COLUMN sort_order INTEGER DEFAULT 0;
