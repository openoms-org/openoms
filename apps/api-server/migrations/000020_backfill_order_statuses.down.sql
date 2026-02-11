-- Down migration: remove out_for_delivery from tenant order status configs
-- This is a best-effort rollback; manually review if needed.

DO $$
DECLARE
    t RECORD;
    statuses jsonb;
    new_statuses jsonb;
    transitions jsonb;
    elem jsonb;
BEGIN
    FOR t IN
        SELECT id, settings
        FROM tenants
        WHERE settings ? 'order_statuses'
    LOOP
        statuses := t.settings->'order_statuses'->'statuses';
        transitions := t.settings->'order_statuses'->'transitions';

        IF statuses IS NULL THEN
            CONTINUE;
        END IF;

        -- Remove out_for_delivery from statuses array
        new_statuses := '[]'::jsonb;
        FOR elem IN SELECT * FROM jsonb_array_elements(statuses) LOOP
            IF elem->>'key' != 'out_for_delivery' THEN
                new_statuses := new_statuses || jsonb_build_array(elem);
            END IF;
        END LOOP;

        -- Remove out_for_delivery from transitions
        IF transitions IS NOT NULL THEN
            transitions := transitions - 'out_for_delivery';
        END IF;

        UPDATE tenants
        SET settings = jsonb_set(
            jsonb_set(t.settings, '{order_statuses,statuses}', new_statuses),
            '{order_statuses,transitions}',
            COALESCE(transitions, '{}'::jsonb)
        )
        WHERE id = t.id;
    END LOOP;
END $$;
