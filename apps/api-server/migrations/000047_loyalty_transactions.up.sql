-- OPE-538 loyalty accrual ledger. Additive TENANT-SCOPED table (FORCE ROW LEVEL
-- SECURITY + tenant_isolation policy, accessed through database.WithTenant) that records
-- one row per order-driven loyalty accrual, keyed UNIQUE by (tenant_id, order_id,
-- program_id). The unique key gives exactly-once accrual: AwardPointsForOrder inserts
-- with ON CONFLICT DO NOTHING and skips all point increments when a row already exists,
-- so force/repeat transitions into the accrual status never double-award.

CREATE TABLE public.loyalty_transactions (
    id          uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id   uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    customer_id uuid NOT NULL REFERENCES public.customers(id) ON DELETE CASCADE,
    program_id  uuid NOT NULL REFERENCES public.loyalty_programs(id) ON DELETE CASCADE,
    order_id    uuid REFERENCES public.orders(id) ON DELETE CASCADE,
    amount      numeric(12,2) NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_loyalty_transactions_order_program ON public.loyalty_transactions (tenant_id, order_id, program_id); -- migrate:index-lock-ok
CREATE INDEX idx_loyalty_transactions_customer ON public.loyalty_transactions (tenant_id, customer_id); -- migrate:index-lock-ok

ALTER TABLE public.loyalty_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.loyalty_transactions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.loyalty_transactions USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.loyalty_transactions TO %I', app_role);
        END IF;
    END LOOP;
END;
$$;
