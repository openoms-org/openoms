-- Migration 000009 down: Remove weight and dimensions
ALTER TABLE products DROP COLUMN weight;
ALTER TABLE products DROP COLUMN width;
ALTER TABLE products DROP COLUMN height;
ALTER TABLE products DROP COLUMN depth;
