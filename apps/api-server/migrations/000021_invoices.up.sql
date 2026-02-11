CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    external_id TEXT,
    external_number TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    invoice_type TEXT NOT NULL DEFAULT 'vat',
    total_net DECIMAL(12,2),
    total_gross DECIMAL(12,2),
    currency TEXT NOT NULL DEFAULT 'PLN',
    issue_date DATE,
    due_date DATE,
    pdf_url TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_invoices_tenant ON invoices(tenant_id, created_at DESC);
CREATE INDEX idx_invoices_order ON invoices(tenant_id, order_id);
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invoices
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON invoices TO openoms_app;
CREATE TRIGGER trigger_invoices_updated_at
    BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION update_updated_at();
