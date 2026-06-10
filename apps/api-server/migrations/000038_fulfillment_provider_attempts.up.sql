-- OPE-417 Fulfillment provider attempts
-- Records every carrier/provider side effect (create shipment, generate/download
-- label, sync tracking) as a tenant-scoped attempt row, so the fulfillment model
-- has an auditable, per-attempt history of provider interactions. Populated as a
-- best-effort side effect, gated behind FULFILLMENT_PROCESS_ENABLED.
--
-- TENANT-SCOPED with RLS (like the OPE-414/415 fulfillment + orchestration tables).
-- process_id is nullable: an attempt may be recorded before a fulfillment process
-- exists for the order (correlation_id then carries the shipment id). result_redacted
-- and payload_hash NEVER hold raw secrets/tokens — only redacted summaries / digests.

CREATE TABLE public.fulfillment_provider_attempts (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id           uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    process_id          uuid REFERENCES public.fulfillment_processes(id) ON DELETE SET NULL,
    provider            text NOT NULL,
    operation           text NOT NULL,
    status              text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','succeeded','failed')),
    request_id          text,
    correlation_id      text,
    raw_provider_status text,
    result_redacted     jsonb,
    payload_hash        text,
    error_code          text,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_fulfillment_provider_attempts_process ON public.fulfillment_provider_attempts (tenant_id, process_id); -- migrate:index-lock-ok
CREATE INDEX idx_fulfillment_provider_attempts_provider_op ON public.fulfillment_provider_attempts (tenant_id, provider, operation); -- migrate:index-lock-ok

-- Row-level security: tenant isolation (matches orders/fulfillment tables).
ALTER TABLE public.fulfillment_provider_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.fulfillment_provider_attempts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.fulfillment_provider_attempts USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['fulfillment_provider_attempts'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
