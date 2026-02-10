-- Migration 000007: Add tags column to orders and products
ALTER TABLE orders ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE products ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';
