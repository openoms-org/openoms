-- OPE-418 supplier-availability read-model. Two additive TENANT-SCOPED tables (FORCE
-- ROW LEVEL SECURITY + tenant_isolation policy, accessed through database.WithTenant).
-- supplier_availability is the raw observational snapshot per supplier_product x
-- warehouse; supplier_availability_policy holds the 4-scope tenant rules. available_to_sell
-- is computed on read, so no value is materialised here.

CREATE TABLE public.supplier_availability (
    id                    uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id             uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    supplier_id           uuid NOT NULL REFERENCES public.suppliers(id) ON DELETE CASCADE,
    supplier_product_id   uuid NOT NULL REFERENCES public.supplier_products(id) ON DELETE CASCADE,
    product_id            uuid REFERENCES public.products(id) ON DELETE SET NULL,
    warehouse_external_id text NOT NULL DEFAULT '',
    source_quantity       integer NOT NULL DEFAULT 0,
    availability_type     text NOT NULL DEFAULT 'unknown'
        CHECK (availability_type IN ('exact_quantity','bucket','boolean','eta_only','unknown')),
    min_handling_days     integer,
    max_handling_days     integer,
    next_delivery_date    date,
    reservation_supported boolean NOT NULL DEFAULT false,
    freshness_observed_at timestamptz NOT NULL DEFAULT now(),
    source_max_stale_seconds integer,
    last_successful_sync_id uuid REFERENCES public.sync_jobs(id) ON DELETE SET NULL,
    raw                   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_supplier_availability_product_wh
    ON public.supplier_availability (tenant_id, supplier_product_id, warehouse_external_id); -- migrate:index-lock-ok
CREATE INDEX idx_supplier_availability_supplier
    ON public.supplier_availability (tenant_id, supplier_id); -- migrate:index-lock-ok
CREATE INDEX idx_supplier_availability_product
    ON public.supplier_availability (tenant_id, product_id); -- migrate:index-lock-ok

CREATE TABLE public.supplier_availability_policy (
    id              uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id       uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    scope           text NOT NULL CHECK (scope IN ('supplier','product','listing','channel')),
    supplier_id     uuid REFERENCES public.suppliers(id) ON DELETE CASCADE,
    product_id      uuid REFERENCES public.products(id) ON DELETE CASCADE,
    listing_id      uuid REFERENCES public.product_listings(id) ON DELETE CASCADE,
    channel         text,
    mode            text NOT NULL DEFAULT 'auto' CHECK (mode IN ('auto','manual','paused')),
    safety_buffer   integer NOT NULL DEFAULT 0,
    freshness_window_seconds integer,
    max_lead_time_days integer,
    override_quantity integer,
    allow_channel_increase boolean NOT NULL DEFAULT false,
    require_reservation boolean NOT NULL DEFAULT false,
    require_preflight boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_sap_supplier ON public.supplier_availability_policy (tenant_id, supplier_id) WHERE scope = 'supplier'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_product ON public.supplier_availability_policy (tenant_id, supplier_id, product_id) WHERE scope = 'product'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_listing ON public.supplier_availability_policy (tenant_id, listing_id) WHERE scope = 'listing'; -- migrate:index-lock-ok
CREATE UNIQUE INDEX uq_sap_channel ON public.supplier_availability_policy (tenant_id, channel) WHERE scope = 'channel'; -- migrate:index-lock-ok

ALTER TABLE public.supplier_availability ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_availability FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_availability USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.supplier_availability_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.supplier_availability_policy FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.supplier_availability_policy USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['supplier_availability','supplier_availability_policy'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
