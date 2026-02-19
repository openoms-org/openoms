-- Link suppliers to integrations for encrypted credential storage.
-- BTP and other API-based suppliers use integration credentials instead of plain feed URLs.
ALTER TABLE suppliers ADD COLUMN integration_id UUID REFERENCES integrations(id) ON DELETE SET NULL;

CREATE INDEX idx_suppliers_integration_id ON suppliers(integration_id) WHERE integration_id IS NOT NULL;
