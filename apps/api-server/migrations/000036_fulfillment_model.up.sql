-- OPE-414 Fulfillment process data model (ADR-424)
-- Canonical, TENANT-SCOPED fulfillment model: processes / units / steps / blockers.
-- Unlike the platform Studio tables, these are tenant data: RLS enforced via the
-- tenant_isolation policy and accessed through database.WithTenant.

CREATE TABLE public.fulfillment_processes (
    id              uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id       uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    order_id        uuid NOT NULL REFERENCES public.orders(id) ON DELETE CASCADE,
    aggregate_status text NOT NULL DEFAULT 'new'
        CHECK (aggregate_status IN ('new','validating','ready','in_progress','waiting_external','blocked','completed','cancelled')),
    health_status   text NOT NULL DEFAULT 'ok'
        CHECK (health_status IN ('ok','warning','action_required','system_error')),
    metadata        jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_processes_order ON public.fulfillment_processes (tenant_id, order_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_units (
    id             uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id      uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id     uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    parent_unit_id uuid REFERENCES public.fulfillment_units(id) ON DELETE SET NULL,
    unit_type      text NOT NULL CHECK (unit_type IN ('warehouse','dropship','backorder','mixed_child','manual')),
    status         text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','ready','running','waiting_external','blocked','succeeded','failed','cancelled','skipped')),
    metadata       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_units_process ON public.fulfillment_units (tenant_id, process_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_steps (
    id         uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id  uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    unit_id    uuid NOT NULL REFERENCES public.fulfillment_units(id) ON DELETE CASCADE,
    step_key   text NOT NULL,
    status     text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','ready','running','waiting_external','blocked','succeeded','failed','cancelled','skipped')),
    attempts   integer NOT NULL DEFAULT 0,
    metadata   jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_steps_unit ON public.fulfillment_steps (tenant_id, unit_id); -- migrate:index-lock-ok

CREATE TABLE public.fulfillment_blockers (
    id          uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id   uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id  uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    unit_id     uuid REFERENCES public.fulfillment_units(id) ON DELETE CASCADE,
    code        text NOT NULL,
    category    text NOT NULL,
    status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    description text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);
CREATE INDEX idx_fulfillment_blockers_process ON public.fulfillment_blockers (tenant_id, process_id, status); -- migrate:index-lock-ok

-- Row-level security: tenant isolation (matches orders/shipments).
ALTER TABLE public.fulfillment_processes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.fulfillment_processes FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.fulfillment_processes USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.fulfillment_units ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.fulfillment_units FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.fulfillment_units USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.fulfillment_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.fulfillment_steps FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.fulfillment_steps USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.fulfillment_blockers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.fulfillment_blockers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.fulfillment_blockers USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['fulfillment_processes','fulfillment_units','fulfillment_steps','fulfillment_blockers'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
