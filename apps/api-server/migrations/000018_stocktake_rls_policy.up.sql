-- OPE-199: remove legacy fail-open stocktake RLS policies.
--
-- The original tenant_isolation policies used COALESCE(..., tenant_id), which
-- became true for every row when app.current_tenant_id was missing. Migration
-- 000016 added strict policies, but it left the legacy policies in place.
-- Multiple permissive PostgreSQL policies are OR'ed together, so both old and
-- replacement policies must be removed before creating the final policy.

ALTER TABLE public.stocktakes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stocktakes FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON public.stocktakes;
DROP POLICY IF EXISTS stocktakes_tenant ON public.stocktakes;

CREATE POLICY stocktakes_tenant ON public.stocktakes
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE public.stocktake_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.stocktake_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON public.stocktake_items;
DROP POLICY IF EXISTS stocktake_items_tenant ON public.stocktake_items;

CREATE POLICY stocktake_items_tenant ON public.stocktake_items
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DO $$
DECLARE
    table_name text;
    policy_count integer;
    weak_policy_count integer;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['stocktakes', 'stocktake_items']
    LOOP
        SELECT COUNT(*)
        INTO policy_count
        FROM pg_policies
        WHERE schemaname = 'public'
          AND tablename = table_name
          AND permissive = 'PERMISSIVE';

        IF policy_count <> 1 THEN
            RAISE EXCEPTION 'expected exactly one permissive RLS policy on public.%, found %',
                table_name,
                policy_count;
        END IF;

        SELECT COUNT(*)
        INTO weak_policy_count
        FROM pg_policies
        WHERE schemaname = 'public'
          AND tablename = table_name
          AND (
              policyname = 'tenant_isolation'
              OR qual ILIKE '%COALESCE%'
              OR with_check ILIKE '%COALESCE%'
          );

        IF weak_policy_count <> 0 THEN
            RAISE EXCEPTION 'legacy fail-open stocktake RLS policy still exists on public.%', table_name;
        END IF;
    END LOOP;
END;
$$;
