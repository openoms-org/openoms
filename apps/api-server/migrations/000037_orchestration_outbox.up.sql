-- OPE-415 Orchestration outbox + attempts
-- Transactional outbox for fulfillment side effects + per-attempt records.
-- TENANT-SCOPED with RLS (like the OPE-414 fulfillment tables). Side effects are
-- enqueued in the same transaction as the state change; an idempotent worker
-- claims due rows across tenants via the privileged pool with SKIP LOCKED.

CREATE TABLE public.orchestration_outbox (
    id              uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id       uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id      uuid NOT NULL REFERENCES public.fulfillment_processes(id) ON DELETE CASCADE,
    unit_id         uuid REFERENCES public.fulfillment_units(id) ON DELETE SET NULL,
    event_type      text NOT NULL,
    payload         jsonb NOT NULL DEFAULT '{}'::jsonb,
    status          text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','claimed','succeeded','failed')),
    idempotency_key text NOT NULL,
    attempts        integer NOT NULL DEFAULT 0,
    max_attempts    integer NOT NULL DEFAULT 8,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    claimed_at      timestamptz,
    last_error      text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX idx_orchestration_outbox_due ON public.orchestration_outbox (status, next_attempt_at); -- migrate:index-lock-ok

CREATE TABLE public.orchestration_attempts (
    id             uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id      uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    outbox_id      uuid NOT NULL REFERENCES public.orchestration_outbox(id) ON DELETE CASCADE,
    attempt_number integer NOT NULL,
    status         text NOT NULL CHECK (status IN ('running','succeeded','failed')),
    error          text NOT NULL DEFAULT '',
    started_at     timestamptz NOT NULL DEFAULT now(),
    finished_at    timestamptz
);
CREATE INDEX idx_orchestration_attempts_outbox ON public.orchestration_attempts (tenant_id, outbox_id); -- migrate:index-lock-ok

-- Row-level security: tenant isolation (matches orders/fulfillment tables).
ALTER TABLE public.orchestration_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.orchestration_outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.orchestration_outbox USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

ALTER TABLE public.orchestration_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.orchestration_attempts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.orchestration_attempts USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['orchestration_outbox','orchestration_attempts'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
