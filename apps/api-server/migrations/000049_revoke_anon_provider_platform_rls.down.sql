-- Revert: disable RLS + drop the app-role policy on the platform/provider tables.
-- (The anon/authenticated grant revoke is intentionally NOT undone — re-granting would
-- re-open the Data API exposure this migration closed.)
DO $$
DECLARE
	t text;
	tbls text[] := ARRAY[
		'platform_admins', 'platform_audit_log',
		'provider_definitions', 'provider_versions', 'provider_publication_events',
		'provider_field_schemas', 'provider_capability_profiles', 'provider_status_mappings',
		'provider_integration_gaps', 'provider_validation_runs', 'provider_validation_probes',
		'provider_validation_results', 'provider_tenant_enables'
	];
BEGIN
	FOREACH t IN ARRAY tbls LOOP
		EXECUTE format('DROP POLICY IF EXISTS app_full_access ON public.%I', t);
		EXECUTE format('ALTER TABLE public.%I DISABLE ROW LEVEL SECURITY', t);
	END LOOP;
END $$;
