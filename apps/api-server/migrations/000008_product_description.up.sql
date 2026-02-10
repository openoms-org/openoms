-- Migration 000008: Add description fields to products
ALTER TABLE products ADD COLUMN description_short TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN description_long TEXT NOT NULL DEFAULT '';
