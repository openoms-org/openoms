-- Fix RLS policies to use current_setting(..., true) for transaction mode pooler compatibility.
-- Without missing_ok=true, current_setting() throws "unrecognized configuration parameter"
-- when the parameter hasn't been set yet in the current transaction (e.g. Supabase transaction mode).

DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN
    SELECT schemaname, tablename, policyname, cmd
    FROM pg_policies
    WHERE schemaname = 'public'
      AND qual IS NOT NULL
      AND qual LIKE '%current_setting%'
      AND qual NOT LIKE '%true%'
  LOOP
    EXECUTE format('DROP POLICY %I ON %I.%I', r.policyname, r.schemaname, r.tablename);

    IF r.cmd = 'ALL' THEN
      EXECUTE format(
        'CREATE POLICY %I ON %I.%I FOR ALL USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
        r.policyname, r.schemaname, r.tablename
      );
    ELSIF r.cmd = 'SELECT' THEN
      EXECUTE format(
        'CREATE POLICY %I ON %I.%I FOR SELECT USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
        r.policyname, r.schemaname, r.tablename
      );
    ELSIF r.cmd = 'INSERT' THEN
      EXECUTE format(
        'CREATE POLICY %I ON %I.%I FOR INSERT WITH CHECK (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
        r.policyname, r.schemaname, r.tablename
      );
    ELSE
      EXECUTE format(
        'CREATE POLICY %I ON %I.%I USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
        r.policyname, r.schemaname, r.tablename
      );
    END IF;
  END LOOP;
END;
$$;
