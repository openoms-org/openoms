-- Migrate existing flat product categories from tenant settings JSON to product_categories table.
-- Also backfill products.category_id from existing products.category (TEXT slug match).

DO $$
DECLARE
    t RECORD;
    cat RECORD;
BEGIN
    FOR t IN SELECT id, settings FROM tenants WHERE settings IS NOT NULL LOOP
        FOR cat IN
            SELECT * FROM jsonb_array_elements(
                COALESCE(t.settings->'product_categories'->'categories', '[]'::jsonb)
            ) AS elem
        LOOP
            INSERT INTO product_categories (tenant_id, name, slug, color, position, depth)
            VALUES (
                t.id,
                cat.elem->>'label',
                cat.elem->>'key',
                COALESCE(cat.elem->>'color', '#6b7280'),
                COALESCE((cat.elem->>'position')::int, 0),
                0
            ) ON CONFLICT (tenant_id, slug) DO NOTHING;
        END LOOP;

        -- Backfill category_id on products using slug match
        UPDATE products p SET category_id = pc.id
        FROM product_categories pc
        WHERE p.tenant_id = t.id AND pc.tenant_id = t.id
          AND p.category = pc.slug AND p.category_id IS NULL;
    END LOOP;
END $$;
