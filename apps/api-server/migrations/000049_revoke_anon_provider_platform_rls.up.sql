-- Security (Supabase advisor: rls_disabled_in_public). The platform-admin tables (000031)
-- and the Provider Studio tables (OPE-403, 000032-000035) were created in the public schema
-- WITHOUT row-level security and with the default anon/authenticated grants, so they were
-- reachable via the Supabase Data API (PostgREST) with the public anon key — including write
-- access to platform_admins (a super-admin insert = full takeover). The Data API has been
-- disabled (primary fix); this adds DB-layer protection so the tables stay safe regardless of
-- that setting.
--
-- These are GLOBAL tables (no tenant_id) read/written by the app role directly (not via
-- database.WithTenant), so the usual tenant-isolation policy does not apply. Instead: revoke
-- the Data API roles' grants outright, and enable RLS with a permissive app-role policy. The
-- worker/migrations run as a superuser that bypasses RLS; anon/authenticated get no policy and
-- are denied. RLS is enabled ONLY when an app role is detected, so the app is never locked out
-- (RLS with no matching policy denies every non-owner role). On non-Supabase / test databases
-- (no anon/authenticated/app roles) this migration is a no-op.

SET LOCAL lock_timeout = '5s';

DO $$
DECLARE
	t text;
	app_role text;
	tbls text[] := ARRAY[
		'platform_admins', 'platform_audit_log',
		'provider_definitions', 'provider_versions', 'provider_publication_events',
		'provider_field_schemas', 'provider_capability_profiles', 'provider_status_mappings',
		'provider_integration_gaps', 'provider_validation_runs', 'provider_validation_probes',
		'provider_validation_results', 'provider_tenant_enables'
	];
BEGIN
	-- App pool connects as openoms_app (prod) or openoms (staging); prefer openoms_app.
	SELECT rolname INTO app_role FROM pg_roles
	WHERE rolname IN ('openoms_app', 'openoms')
	ORDER BY (rolname = 'openoms_app') DESC
	LIMIT 1;

	FOREACH t IN ARRAY tbls LOOP
		IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
			EXECUTE format('REVOKE ALL ON public.%I FROM anon', t);
		END IF;
		IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
			EXECUTE format('REVOKE ALL ON public.%I FROM authenticated', t);
		END IF;

		IF app_role IS NOT NULL THEN
			EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t);
			IF NOT EXISTS (
				SELECT 1 FROM pg_policies
				WHERE schemaname = 'public' AND tablename = t AND policyname = 'app_full_access'
			) THEN
				EXECUTE format(
					'CREATE POLICY app_full_access ON public.%I FOR ALL TO %I USING (true) WITH CHECK (true)',
					t, app_role);
			END IF;
		END IF;
	END LOOP;
END $$;
