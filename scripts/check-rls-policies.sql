DO $$
DECLARE
    bad_policy text;
    missing_force text;
BEGIN
    SELECT string_agg(format('%I.%I policy %I', schemaname, tablename, policyname), E'\n')
    INTO bad_policy
    FROM pg_policies
    WHERE schemaname = 'public'
      AND tablename IN ('allegro_parameter_mappings', 'message_templates')
      AND (
          COALESCE(qual, '') NOT LIKE '%current_setting(%true%'
          OR COALESCE(with_check, '') NOT LIKE '%current_setting(%true%'
      );

    IF bad_policy IS NOT NULL THEN
        RAISE EXCEPTION 'RLS policies without missing_ok=true:%', E'\n' || bad_policy;
    END IF;

    SELECT string_agg(format('%I.%I', n.nspname, c.relname), E'\n')
    INTO missing_force
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public'
      AND c.relname IN ('allegro_parameter_mappings', 'message_templates')
      AND c.relforcerowsecurity IS DISTINCT FROM true;

    IF missing_force IS NOT NULL THEN
        RAISE EXCEPTION 'Tenant tables without FORCE ROW LEVEL SECURITY:%', E'\n' || missing_force;
    END IF;
END $$;
