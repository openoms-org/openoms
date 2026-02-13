CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_orders_customer_name_trgm ON orders USING gin (customer_name gin_trgm_ops);
CREATE INDEX idx_orders_customer_email_trgm ON orders USING gin (customer_email gin_trgm_ops);
CREATE INDEX idx_customers_name_trgm ON customers USING gin (name gin_trgm_ops);
CREATE INDEX idx_customers_email_trgm ON customers USING gin (email gin_trgm_ops);
CREATE INDEX idx_products_name_trgm ON products USING gin (name gin_trgm_ops);
