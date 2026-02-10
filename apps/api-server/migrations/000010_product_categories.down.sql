-- Migration 000010 down: Remove category from products
ALTER TABLE products DROP COLUMN category;
