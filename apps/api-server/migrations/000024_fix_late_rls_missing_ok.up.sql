-- Fix late RLS policies that were added after 000003 and reintroduced
-- current_setting(...) without missing_ok=true.

ALTER TABLE public.allegro_parameter_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.allegro_parameter_mappings FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON public.allegro_parameter_mappings;

CREATE POLICY tenant_isolation ON public.allegro_parameter_mappings
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE public.message_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.message_templates FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS message_templates_tenant ON public.message_templates;

CREATE POLICY message_templates_tenant ON public.message_templates
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DO $$
DECLARE
    bad_policy_count integer;
    missing_force_count integer;
BEGIN
    SELECT COUNT(*)
    INTO bad_policy_count
    FROM pg_policies
    WHERE schemaname = 'public'
      AND tablename IN ('allegro_parameter_mappings', 'message_templates')
      AND (
          qual NOT LIKE '%current_setting(%true%'
          OR with_check NOT LIKE '%current_setting(%true%'
      );

    IF bad_policy_count <> 0 THEN
        RAISE EXCEPTION 'OPE-292: unsafe RLS policies remain on allegro_parameter_mappings/message_templates';
    END IF;

    SELECT COUNT(*)
    INTO missing_force_count
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public'
      AND c.relname IN ('allegro_parameter_mappings', 'message_templates')
      AND c.relforcerowsecurity IS DISTINCT FROM true;

    IF missing_force_count <> 0 THEN
        RAISE EXCEPTION 'OPE-292: FORCE ROW LEVEL SECURITY missing on allegro_parameter_mappings/message_templates';
    END IF;
END $$;
