-- +goose Up
-- ============================================================================
-- SAMPLE DATA: a fully populated org for demo / review purposes.
-- Creates 1 org, 3 sites, 9 assets, 3 sensors, 3 policies, ~60 days of
-- readings, and then runs refresh_emissions() to generate rollups + violations.
-- ============================================================================

-- ── Organization ────────────────────────────────────────────────────────────

INSERT INTO organizations (id, name, description, office_location, email)
VALUES ('a0000000-0000-0000-0000-000000000001',
        'Acme Energy Corp',
        'Sample organization seeded for demo and testing',
        'Houston, TX',
        'ops@acme-energy.example.com');

-- ── Sites ───────────────────────────────────────────────────────────────────

INSERT INTO sites (id, org_id, name, location) VALUES
    ('b0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'North Ridge Well Pad', 'Permian Basin, TX'),
    ('b0000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001',
     'Coastal Refinery',     'Corpus Christi, TX'),
    ('b0000000-0000-0000-0000-000000000003',
     'a0000000-0000-0000-0000-000000000001',
     'Midland Processing',   'Midland, TX');

-- ── Assets (3 per site, using seeded asset_types from migration 003) ────────

INSERT INTO assets (id, org_id, site_id, asset_type_id, name) VALUES
    -- North Ridge Well Pad
    ('c0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000001',
     (SELECT id FROM asset_types WHERE name = 'compressor'),
     'Compressor A1'),
    ('c0000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000001',
     (SELECT id FROM asset_types WHERE name = 'wellhead'),
     'Wellhead W-101'),
    ('c0000000-0000-0000-0000-000000000003',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000001',
     (SELECT id FROM asset_types WHERE name = 'separator'),
     'Separator S-10'),
    -- Coastal Refinery
    ('c0000000-0000-0000-0000-000000000004',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000002',
     (SELECT id FROM asset_types WHERE name = 'tank'),
     'Storage Tank T-200'),
    ('c0000000-0000-0000-0000-000000000005',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000002',
     (SELECT id FROM asset_types WHERE name = 'flare'),
     'Flare Stack F-1'),
    ('c0000000-0000-0000-0000-000000000006',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000002',
     (SELECT id FROM asset_types WHERE name = 'heat_exchanger'),
     'Heat Exchanger HX-5'),
    -- Midland Processing
    ('c0000000-0000-0000-0000-000000000007',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000003',
     (SELECT id FROM asset_types WHERE name = 'pump'),
     'Transfer Pump P-30'),
    ('c0000000-0000-0000-0000-000000000008',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000003',
     (SELECT id FROM asset_types WHERE name = 'generator'),
     'Generator G-2'),
    ('c0000000-0000-0000-0000-000000000009',
     'a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000003',
     (SELECT id FROM asset_types WHERE name = 'valve'),
     'Relief Valve RV-7');

-- ── Recorder Devices (org-scoped sensors) ───────────────────────────────────

INSERT INTO recorder_devices (id, org_id, type, device_metadata) VALUES
    ('d0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'sensor', '{"model": "EM-500", "serial": "SN-ACME-001"}'),
    ('d0000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001',
     'sensor', '{"model": "EM-500", "serial": "SN-ACME-002"}'),
    ('d0000000-0000-0000-0000-000000000003',
     'a0000000-0000-0000-0000-000000000001',
     'satellite', '{"model": "SAT-200", "serial": "SN-ACME-SAT-001"}');

-- ── Emission Policies ───────────────────────────────────────────────────────
-- Set limits low enough that the sample data will trigger violations.

INSERT INTO emission_policies (id, org_id, entity_type, entity_id,
                               emission_limit, unit, period, effective_from) VALUES
    -- Org-level: max 5000 kg/month
    ('e0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'org', 'a0000000-0000-0000-0000-000000000001',
     5000, 'kg_co2e', 'month', now() - INTERVAL '90 days'),
    -- Site-level: North Ridge max 200 kg/day
    ('e0000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001',
     'site', 'b0000000-0000-0000-0000-000000000001',
     200, 'kg_co2e', 'day', now() - INTERVAL '90 days'),
    -- Asset-level: Flare Stack max 50 kg/hour
    ('e0000000-0000-0000-0000-000000000003',
     'a0000000-0000-0000-0000-000000000001',
     'asset', 'c0000000-0000-0000-0000-000000000005',
     50, 'kg_co2e', 'hour', now() - INTERVAL '90 days');

-- ── Ensure partitions exist for the seed date range ─────────────────────────
-- The foundation migration only creates partitions for a few months ahead.
-- The seed inserts readings going back 60 days, which may fall in earlier
-- months without partitions. This block creates any missing monthly partitions
-- for both asset_emission_readings and emission_rollup_hourly.

-- +goose StatementBegin
DO $$
DECLARE
    v_start DATE := date_trunc('month', now() - INTERVAL '60 days')::DATE;
    v_end   DATE := date_trunc('month', now() + INTERVAL '1 month')::DATE;
    v_month DATE;
    v_next  DATE;
    v_name  TEXT;
BEGIN
    v_month := v_start;
    WHILE v_month <= v_end LOOP
        v_next := v_month + INTERVAL '1 month';

        -- asset_emission_readings partition
        v_name := 'asset_emission_readings_' || to_char(v_month, 'YYYY_MM');
        IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = v_name) THEN
            EXECUTE format(
                'CREATE TABLE %I PARTITION OF asset_emission_readings FOR VALUES FROM (%L) TO (%L)',
                v_name, v_month, v_next
            );
            RAISE NOTICE 'Created partition %', v_name;
        END IF;

        -- emission_rollup_hourly partition
        v_name := 'emission_rollup_hourly_' || to_char(v_month, 'YYYY_MM');
        IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = v_name) THEN
            EXECUTE format(
                'CREATE TABLE %I PARTITION OF emission_rollup_hourly FOR VALUES FROM (%L) TO (%L)',
                v_name, v_month, v_next
            );
            RAISE NOTICE 'Created partition %', v_name;
        END IF;

        v_month := v_next;
    END LOOP;
END;
$$;
-- +goose StatementEnd

-- ── Generate Readings ───────────────────────────────────────────────────────
-- ~60 days of hourly-ish readings for each of the 9 assets. Each asset gets
-- 2-4 readings per day, totalling ~15k-25k rows.

-- +goose StatementBegin
DO $$
DECLARE
    v_org_id      UUID := 'a0000000-0000-0000-0000-000000000001';
    v_batch_id    UUID;
    v_asset       RECORD;
    v_device_ids  UUID[] := ARRAY[
        'd0000000-0000-0000-0000-000000000001'::UUID,
        'd0000000-0000-0000-0000-000000000002'::UUID,
        'd0000000-0000-0000-0000-000000000003'::UUID
    ];
    v_day         INT;
    v_hour        INT;
    v_reading     NUMERIC;
    v_ts          TIMESTAMPTZ;
    v_device_idx  INT;
    -- Base emission rates per asset (varied to make data interesting)
    v_base_rates  NUMERIC[] := ARRAY[
        25.0,  -- Compressor A1 (high emitter)
        12.0,  -- Wellhead W-101
        8.0,   -- Separator S-10
        15.0,  -- Storage Tank T-200
        45.0,  -- Flare Stack F-1 (highest - will breach hourly policy)
        10.0,  -- Heat Exchanger HX-5
        6.0,   -- Transfer Pump P-30
        20.0,  -- Generator G-2
        3.0    -- Relief Valve RV-7 (lowest)
    ];
    v_asset_idx   INT;
BEGIN
    v_asset_idx := 0;

    FOR v_asset IN
        SELECT id, site_id
        FROM assets
        WHERE org_id = v_org_id
        ORDER BY id
    LOOP
        v_asset_idx := v_asset_idx + 1;
        v_batch_id := gen_random_uuid();

        -- Generate readings for the past 60 days
        FOR v_day IN 0..59 LOOP
            -- 4 readings per day at varied hours
            FOR v_hour IN 1..4 LOOP
                v_ts := date_trunc('hour', now()) - (v_day || ' days')::INTERVAL
                         + ((v_hour * 6) || ' hours')::INTERVAL;

                -- Base rate + random variation (±40%) + daily pattern
                v_reading := v_base_rates[v_asset_idx]
                             * (0.6 + random() * 0.8)
                             * (0.8 + 0.4 * sin(v_hour * 1.57));

                -- Occasional spikes (10% chance, 2-3x normal)
                IF random() < 0.10 THEN
                    v_reading := v_reading * (2.0 + random());
                END IF;

                v_reading := round(v_reading, 2);
                v_device_idx := 1 + (v_asset_idx % 3);

                INSERT INTO asset_emission_readings
                    (org_id, site_id, asset_id, recorder_device_id,
                     batch_id, reading, unit, recorded_at)
                VALUES
                    (v_org_id, v_asset.site_id, v_asset.id,
                     v_device_ids[v_device_idx],
                     v_batch_id, v_reading, 'kg_co2e', v_ts);
            END LOOP;
        END LOOP;
    END LOOP;

    RAISE NOTICE 'Seeded readings for 9 assets over 60 days';
END;
$$;
-- +goose StatementEnd

-- ── Generate rollups + detect violations ────────────────────────────────────
-- Run the cascade over the full 90-day window to process all seeded readings.
CALL refresh_emissions(INTERVAL '90 days');


-- +goose Down

-- Reverse order: violations, rollups, readings, policies, devices, assets, sites, org.
DELETE FROM emission_violations   WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM emission_rollup_hourly  WHERE entity_id IN (
    SELECT id FROM assets WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT id FROM sites WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT 'a0000000-0000-0000-0000-000000000001'::UUID
);
DELETE FROM emission_rollup_daily   WHERE entity_id IN (
    SELECT id FROM assets WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT id FROM sites WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT 'a0000000-0000-0000-0000-000000000001'::UUID
);
DELETE FROM emission_rollup_monthly WHERE entity_id IN (
    SELECT id FROM assets WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT id FROM sites WHERE org_id = 'a0000000-0000-0000-0000-000000000001'
    UNION SELECT 'a0000000-0000-0000-0000-000000000001'::UUID
);
DELETE FROM asset_emission_readings WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM emission_policies       WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM recorder_devices        WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM assets                  WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM sites                   WHERE org_id = 'a0000000-0000-0000-0000-000000000001';
DELETE FROM organizations           WHERE id    = 'a0000000-0000-0000-0000-000000000001';

-- Drop partitions that were dynamically created by the up migration.
-- Only drop months NOT already created by 002 (which covers 2026_05, 2026_06, 2026_07).
-- +goose StatementBegin
DO $$
DECLARE
    v_start DATE := date_trunc('month', now() - INTERVAL '60 days')::DATE;
    v_end   DATE := date_trunc('month', now() + INTERVAL '1 month')::DATE;
    v_month DATE;
    v_name  TEXT;
BEGIN
    v_month := v_start;
    WHILE v_month < '2026-05-01'::DATE LOOP
        v_name := 'asset_emission_readings_' || to_char(v_month, 'YYYY_MM');
        IF EXISTS (SELECT 1 FROM pg_class WHERE relname = v_name) THEN
            EXECUTE format('DROP TABLE %I', v_name);
        END IF;

        v_name := 'emission_rollup_hourly_' || to_char(v_month, 'YYYY_MM');
        IF EXISTS (SELECT 1 FROM pg_class WHERE relname = v_name) THEN
            EXECUTE format('DROP TABLE %I', v_name);
        END IF;

        v_month := v_month + INTERVAL '1 month';
    END LOOP;
END;
$$;
-- +goose StatementEnd
