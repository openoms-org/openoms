-- OPE-406 Credential & Settings Schema Contract
-- One field-schema per provider version (the source of truth that generates
-- admin validation forms and customer setup forms). Platform-managed, NO RLS.
-- The schema only DECLARES which fields are secret; tenant credential VALUES
-- continue to live AES-encrypted in integrations.credentials.

CREATE TABLE public.provider_field_schemas (
    id                  uuid PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    provider_version_id uuid NOT NULL UNIQUE REFERENCES public.provider_versions(id) ON DELETE CASCADE,
    schema              jsonb NOT NULL DEFAULT '{"groups":[]}'::jsonb,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- Grant the least-privilege app role(s): production (Supabase) "openoms_app",
-- self-hosted/staging "openoms". Detect dynamically for portability.
DO $$
DECLARE
    app_role text;
BEGIN
    FOREACH app_role IN ARRAY ARRAY['openoms_app', 'openoms'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = app_role) THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.provider_field_schemas TO %I', app_role);
        END IF;
    END LOOP;
END;
$$;
