-- Migration 000008 down: Remove description fields
ALTER TABLE products DROP COLUMN description_short;
ALTER TABLE products DROP COLUMN description_long;
