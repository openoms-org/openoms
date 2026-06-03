-- OPE-407 Capability Profiles, Status Mappings & Integration Gaps
-- Per provider-version declarations for the Provider Integration Studio.
-- Platform-managed, NOT tenant-scoped, NO row-level security.

CREATE TABLE public.provider_capability_profiles (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    capability_key      text NOT NULL,
    support_status      text NOT NULL
        CHECK (support_status IN ('supported','configured','unsupported','requires_manual','degraded','unknown')),
    channel             text NOT NULL DEFAULT '',
    mode                text NOT NULL DEFAULT '',
    freshness           text NOT NULL DEFAULT '',
    required_inputs     text[] NOT NULL DEFAULT '{}'::text[],
    provided_outputs    text[] NOT NULL DEFAULT '{}'::text[],
    latency_sla_seconds integer,
    notes               text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, capability_key)
);
CREATE INDEX idx_provider_caps_version ON public.provider_capability_profiles (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_status_mappings (
    id                   uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id  uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    status_domain        text NOT NULL CHECK (status_domain IN ('order','shipment','line')),
    raw_status           text NOT NULL,
    canonical_status     text NOT NULL DEFAULT '',
    canonical_event_type text NOT NULL DEFAULT '',
    canonical_step_key   text NOT NULL DEFAULT '',
    confidence           text NOT NULL DEFAULT 'medium' CHECK (confidence IN ('high','medium','low')),
    is_terminal          boolean NOT NULL DEFAULT false,
    notes                text NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_version_id, status_domain, raw_status)
);
CREATE INDEX idx_provider_status_maps_version ON public.provider_status_mappings (provider_version_id); -- migrate:index-lock-ok

CREATE TABLE public.provider_integration_gaps (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    gap_type            text NOT NULL,
    severity            text NOT NULL CHECK (severity IN ('info','warning','action_required','system_error')),
    status              text NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    description         text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    resolved_at         timestamptz
);
CREATE INDEX idx_provider_gaps_version ON public.provider_integration_gaps (provider_version_id, status); -- migrate:index-lock-ok

-- Grant the least-privilege app role(s): production "openoms_app", self-hosted "openoms".
DO $$
DECLARE
    app_role text;
    tbl      text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            FOREACH tbl IN ARRAY ARRAY['provider_capability_profiles', 'provider_status_mappings', 'provider_integration_gaps'] LOOP
                EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO %I', tbl, app_role);
            END LOOP;
        END IF;
    END LOOP;
END;
$$;
