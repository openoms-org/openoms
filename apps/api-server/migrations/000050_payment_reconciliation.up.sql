-- Payment settlements: imported gateway payouts/settlements
CREATE TABLE payment_settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider TEXT NOT NULL,
    settlement_id TEXT,
    settlement_date DATE NOT NULL,
    total_amount DECIMAL(12,2) NOT NULL,
    fee_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    net_amount DECIMAL(12,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'PLN',
    status TEXT NOT NULL DEFAULT 'pending',
    notes TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Payment transactions: individual transactions within a settlement
CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    settlement_id UUID REFERENCES payment_settlements(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id),
    provider TEXT NOT NULL,
    external_transaction_id TEXT,
    amount DECIMAL(12,2) NOT NULL,
    fee DECIMAL(12,2) NOT NULL DEFAULT 0,
    net_amount DECIMAL(12,2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'PLN',
    transaction_type TEXT NOT NULL DEFAULT 'payment',
    transaction_date TIMESTAMPTZ NOT NULL,
    match_status TEXT NOT NULL DEFAULT 'unmatched',
    match_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS
ALTER TABLE payment_settlements ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payment_settlements USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON payment_transactions USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_payment_settlements_tenant ON payment_settlements(tenant_id);
CREATE INDEX idx_payment_settlements_provider ON payment_settlements(provider);
CREATE INDEX idx_payment_settlements_date ON payment_settlements(settlement_date);
CREATE INDEX idx_payment_transactions_settlement ON payment_transactions(settlement_id);
CREATE INDEX idx_payment_transactions_order ON payment_transactions(order_id);
CREATE INDEX idx_payment_transactions_external ON payment_transactions(external_transaction_id);
CREATE INDEX idx_payment_transactions_match ON payment_transactions(match_status);

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON payment_settlements TO openoms_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON payment_transactions TO openoms_app;
