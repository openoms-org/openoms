-- OPE-203: enforce idempotency for imported marketplace orders.
-- Keep the earliest order for each tenant/source/external_id and clear the duplicate
-- external_id values before adding the partial unique index.
WITH duplicate_orders AS (
    SELECT
        id,
        external_id,
        ROW_NUMBER() OVER (
            PARTITION BY tenant_id, source, external_id
            ORDER BY created_at ASC, id ASC
        ) AS row_num
    FROM public.orders
    WHERE external_id IS NOT NULL
      AND external_id <> ''
)
UPDATE public.orders AS o
SET
    metadata = COALESCE(o.metadata, '{}'::jsonb) || jsonb_build_object(
        'dedup_original_external_id', o.external_id,
        'dedup_migration', '000021_unique_order_external_id',
        'dedup_at', now()
    ),
    external_id = NULL,
    updated_at = now()
FROM duplicate_orders AS d
WHERE o.id = d.id
  AND d.row_num > 1;

DROP INDEX IF EXISTS public.idx_orders_external;

CREATE UNIQUE INDEX idx_orders_tenant_source_external_id_unique
    ON public.orders USING btree (tenant_id, source, external_id)
    WHERE external_id IS NOT NULL AND external_id <> '';
