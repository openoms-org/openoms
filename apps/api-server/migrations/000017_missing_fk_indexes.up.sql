-- Add indexes on frequently queried FK columns missing from init schema.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_returns_order_id ON returns(order_id);
CREATE INDEX IF NOT EXISTS idx_shipments_warehouse_id ON shipments(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_warehouse_doc_items_document ON warehouse_document_items(document_id);
