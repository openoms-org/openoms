-- Add external_id column to shipments table for linking with external systems (e.g., Allegro "Wysyłam z Allegro").
ALTER TABLE shipments ADD COLUMN external_id VARCHAR(255);

CREATE INDEX idx_shipments_external ON shipments(tenant_id, external_id) WHERE external_id IS NOT NULL;
