ALTER TABLE customers DROP COLUMN IF EXISTS price_list_id;

DROP TRIGGER IF EXISTS update_price_list_items_updated_at ON price_list_items;
DROP TRIGGER IF EXISTS update_price_lists_updated_at ON price_lists;

DROP TABLE IF EXISTS price_list_items;
DROP TABLE IF EXISTS price_lists;
