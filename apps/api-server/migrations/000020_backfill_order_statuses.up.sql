-- Migration 000020: Backfill out_for_delivery status for tenants with old configs
--
-- Tenants that saved order_statuses before out_for_delivery was added are missing it.
-- This adds the status and its transitions to any config that lacks it.

DO $$
DECLARE
    t RECORD;
    statuses jsonb;
    new_statuses jsonb;
    transitions jsonb;
    elem jsonb;
    has_out_for_delivery boolean;
    delivered_pos int;
BEGIN
    FOR t IN
        SELECT id, settings
        FROM tenants
        WHERE settings ? 'order_statuses'
    LOOP
        statuses := t.settings->'order_statuses'->'statuses';
        transitions := t.settings->'order_statuses'->'transitions';

        IF statuses IS NULL OR transitions IS NULL THEN
            CONTINUE;
        END IF;

        -- Check if out_for_delivery already exists
        has_out_for_delivery := false;
        FOR elem IN SELECT * FROM jsonb_array_elements(statuses) LOOP
            IF elem->>'key' = 'out_for_delivery' THEN
                has_out_for_delivery := true;
            END IF;
        END LOOP;

        IF has_out_for_delivery THEN
            CONTINUE;
        END IF;

        -- Find delivered position
        delivered_pos := 0;
        FOR elem IN SELECT * FROM jsonb_array_elements(statuses) LOOP
            IF elem->>'key' = 'delivered' THEN
                delivered_pos := (elem->>'position')::int;
            END IF;
        END LOOP;

        IF delivered_pos = 0 THEN
            delivered_pos := 8; -- default
        END IF;

        -- Bump positions for delivered and everything after
        new_statuses := '[]'::jsonb;
        FOR elem IN SELECT * FROM jsonb_array_elements(statuses) LOOP
            IF (elem->>'position')::int >= delivered_pos THEN
                new_statuses := new_statuses || jsonb_build_array(
                    jsonb_set(elem, '{position}', to_jsonb((elem->>'position')::int + 1))
                );
            ELSE
                new_statuses := new_statuses || jsonb_build_array(elem);
            END IF;
        END LOOP;

        -- Insert out_for_delivery at delivered's old position
        new_statuses := new_statuses || jsonb_build_array(
            jsonb_build_object(
                'key', 'out_for_delivery',
                'label', 'W doręczeniu',
                'color', 'teal',
                'position', delivered_pos
            )
        );

        -- Update transitions: in_transit -> [..., out_for_delivery]
        IF transitions ? 'in_transit' THEN
            IF NOT (transitions->'in_transit' @> '"out_for_delivery"'::jsonb) THEN
                transitions := jsonb_set(
                    transitions,
                    '{in_transit}',
                    transitions->'in_transit' || '["out_for_delivery"]'::jsonb
                );
            END IF;
        END IF;

        -- out_for_delivery -> [delivered, refunded]
        transitions := transitions || '{"out_for_delivery": ["delivered", "refunded"]}'::jsonb;

        -- Save back
        UPDATE tenants
        SET settings = jsonb_set(
            jsonb_set(t.settings, '{order_statuses,statuses}', new_statuses),
            '{order_statuses,transitions}',
            transitions
        )
        WHERE id = t.id;
    END LOOP;
END $$;
