-- Migration 000012: Returns/RMA table
CREATE TABLE returns (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id      UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status        VARCHAR(20)  NOT NULL DEFAULT 'requested',
    reason        TEXT         NOT NULL DEFAULT '',
    items         JSONB        NOT NULL DEFAULT '[]',
    refund_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    notes         TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_returns_tenant ON returns(tenant_id, created_at DESC);
CREATE INDEX idx_returns_order ON returns(tenant_id, order_id);

ALTER TABLE returns ENABLE ROW LEVEL SECURITY;
CREATE POLICY returns_tenant_isolation ON returns
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON returns TO openoms_app;
