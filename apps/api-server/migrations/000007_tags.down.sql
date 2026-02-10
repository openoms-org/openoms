-- Migration 000007 down: Remove tags columns
ALTER TABLE orders DROP COLUMN tags;
ALTER TABLE products DROP COLUMN tags;
