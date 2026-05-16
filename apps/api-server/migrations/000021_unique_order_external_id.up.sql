-- OPE-203: enforce idempotency for imported marketplace orders.
-- Keep the earliest order for each tenant/source/external_id and clear the duplicate
-- external_id values before adding the partial unique index.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';

WITH duplicate_keys AS (
    SELECT tenant_id, source, external_id
    FROM public.orders
    WHERE external_id IS NOT NULL
      AND external_id <> ''
    GROUP BY tenant_id, source, external_id
    HAVING COUNT(*) > 1
),
duplicate_orders AS (
    SELECT
        o.id,
        o.external_id,
        ROW_NUMBER() OVER (
            PARTITION BY o.tenant_id, o.source, o.external_id
            ORDER BY o.created_at ASC, o.id ASC
        ) AS row_num
    FROM public.orders AS o
    JOIN duplicate_keys AS d
      ON d.tenant_id = o.tenant_id
     AND d.source = o.source
     AND d.external_id = o.external_id
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
