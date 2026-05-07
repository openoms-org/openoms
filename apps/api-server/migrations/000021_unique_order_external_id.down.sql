DROP INDEX IF EXISTS public.idx_orders_tenant_source_external_id_unique;

CREATE INDEX IF NOT EXISTS idx_orders_external
    ON public.orders USING btree (tenant_id, source, external_id);
