-- Supplier portal access tokens
CREATE TABLE supplier_portal_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(token_hash)
);

-- Supplier messages/notes on POs
CREATE TABLE supplier_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    supplier_id UUID REFERENCES suppliers(id),
    user_id UUID REFERENCES users(id),
    message TEXT NOT NULL,
    is_from_supplier BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE supplier_portal_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON supplier_portal_tokens USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON supplier_messages USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE INDEX idx_supplier_portal_tokens_supplier ON supplier_portal_tokens(supplier_id);
CREATE INDEX idx_supplier_portal_tokens_hash ON supplier_portal_tokens(token_hash);
CREATE INDEX idx_supplier_messages_po ON supplier_messages(purchase_order_id);

-- Add portal_enabled flag to suppliers
ALTER TABLE suppliers ADD COLUMN portal_enabled BOOLEAN NOT NULL DEFAULT false;

GRANT SELECT, INSERT, UPDATE, DELETE ON supplier_portal_tokens TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON supplier_messages TO openoms_app;
