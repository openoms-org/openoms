-- OPE-421/Phase-13 external-workflow connector. Additive: a tenant-scoped FORCE-RLS
-- token table for callback auth, plus a column on orchestration_attempts to record the
-- external engine's execution id. No destructive ops.

CREATE TABLE public.external_workflow_tokens (
    id             uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    tenant_id      uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    integration_id uuid NOT NULL REFERENCES public.integrations(id) ON DELETE CASCADE,
    token_hash     text NOT NULL,
    scopes         text[] NOT NULL DEFAULT '{}',
    expires_at     timestamptz,
    last_used_at   timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_external_workflow_tokens_hash ON public.external_workflow_tokens (tenant_id, token_hash); -- migrate:index-lock-ok
CREATE INDEX idx_external_workflow_tokens_integration ON public.external_workflow_tokens (tenant_id, integration_id); -- migrate:index-lock-ok

ALTER TABLE public.external_workflow_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.external_workflow_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON public.external_workflow_tokens USING ((tenant_id = (current_setting('app.current_tenant_id'::text, true))::uuid));

-- Record the external engine's execution id on the dispatch attempt (Phase-13 requirement).
ALTER TABLE public.orchestration_attempts ADD COLUMN IF NOT EXISTS external_execution_id text;

DO $$
DECLARE
    app_role text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.external_workflow_tokens TO %I', app_role);
        END IF;
    END LOOP;
END;
$$;
