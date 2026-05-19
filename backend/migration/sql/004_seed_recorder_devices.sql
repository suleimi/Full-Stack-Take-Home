-- +goose Up
-- Seed a set of global field devices (org_id IS NULL) so the emission
-- simulator has devices available for any organization out of the box.
INSERT INTO recorder_devices (type, org_id, device_metadata) VALUES
    ('field_device', NULL, '{"model": "FD-100", "serial": "FD-2026-001"}'),
    ('field_device', NULL, '{"model": "FD-100", "serial": "FD-2026-002"}'),
    ('field_device', NULL, '{"model": "FD-200", "serial": "FD-2026-003"}');

-- +goose Down
DELETE FROM recorder_devices
WHERE type = 'field_device'
  AND device_metadata->>'serial' IN ('FD-2026-001', 'FD-2026-002', 'FD-2026-003');
