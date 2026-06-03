-- OPE-405 Provider Registry (rollback). Drop in FK-safe order.
DROP TABLE IF EXISTS public.provider_tenant_enables;
DROP TABLE IF EXISTS public.provider_publication_events;
ALTER TABLE IF EXISTS public.provider_definitions DROP CONSTRAINT IF EXISTS provider_definitions_latest_version_fkey;
DROP TABLE IF EXISTS public.provider_versions;
DROP TABLE IF EXISTS public.provider_definitions;
