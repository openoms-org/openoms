-- Revert merged_into and split_from FK constraints to default (NO ACTION).

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_merged_into_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_merged_into_fkey
    FOREIGN KEY (merged_into) REFERENCES orders(id);

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_split_from_fkey;
ALTER TABLE orders ADD CONSTRAINT orders_split_from_fkey
    FOREIGN KEY (split_from) REFERENCES orders(id);
