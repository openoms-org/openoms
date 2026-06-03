-- OPE-408 Validation Engine & Evidence
-- Probe definitions + immutable validation runs/results for the Provider
-- Integration Studio. Platform-managed, NOT tenant-scoped, NO row-level security.
-- Results store only redacted observations + payload hashes, never raw secrets.

CREATE TABLE public.provider_validation_probes (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    probe_type          text NOT NULL CHECK (probe_type IN (
        'auth_check','endpoint_reachability','feed_fetch','feed_parse','sample_catalog_read',
        'sample_stock_read','sample_price_read','order_preflight','sandbox_order_create',
        'order_status_read','shipment_tracking_read','invoice_read','webhook_signature_verification',
        'malformed_payload_test','rate_limit_behavior')),
    label               text NOT NULL,
    destructive         boolean NOT NULL DEFAULT false,
    required            boolean NOT NULL DEFAULT false,
    config              jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, label)
);
CREATE INDEX idx_provider_vprobes_version ON public.provider_validation_probes (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_validation_runs (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    environment         text NOT NULL CHECK (environment IN ('sandbox','production')),
    verdict             text NOT NULL DEFAULT 'pending' CHECK (verdict IN ('pending','passed','failed','error')),
    started_by          uuid REFERENCES public.users(id) ON DELETE SET NULL,
    started_at          timestamptz NOT NULL DEFAULT now(),
    finished_at         timestamptz,
    notes               text NOT NULL DEFAULT ''
);
CREATE INDEX idx_provider_vruns_version ON public.provider_validation_runs (provider_version_id, started_at DESC); -- migrate:index-lock-ok

CREATE TABLE public.provider_validation_results (
    id           uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    run_id       uuid NOT NULL REFERENCES public.provider_validation_runs(id) ON DELETE CASCADE,
    probe_type   text NOT NULL,
    label        text NOT NULL,
    status       text NOT NULL CHECK (status IN ('passed','failed','skipped','error')),
    observation  text NOT NULL DEFAULT '',
    payload_hash text NOT NULL DEFAULT '',
    findings     text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (run_id, label)
);

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['provider_validation_probes', 'provider_validation_runs', 'provider_validation_results'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
