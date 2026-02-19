ALTER TABLE supplier_products DROP COLUMN IF EXISTS source_category;
DROP TABLE IF EXISTS supplier_category_mappings;
ALTER TABLE suppliers DROP COLUMN IF EXISTS default_category_id;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;
