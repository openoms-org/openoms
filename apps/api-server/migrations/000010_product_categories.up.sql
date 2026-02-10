-- Migration 000010: Add category to products
ALTER TABLE products ADD COLUMN category TEXT DEFAULT NULL;
