-- Reverse data migration: clear backfilled category_id references.
-- Does NOT delete product_categories rows (those are owned by 000066 down migration).
UPDATE products SET category_id = NULL;
