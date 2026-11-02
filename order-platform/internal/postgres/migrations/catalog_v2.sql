ALTER TABLE outbox_events ADD COLUMN lease_until TIMESTAMPTZ;
ALTER TABLE outbox_events ADD COLUMN dead_lettered_at TIMESTAMPTZ;
ALTER TABLE catalog_products ADD COLUMN aggregate_version BIGINT NOT NULL DEFAULT 1 CHECK (aggregate_version > 0);

UPDATE outbox_events
SET payload = jsonb_set(payload, '{aggregateVersion}', '1'::jsonb, true)
WHERE NOT (payload ? 'aggregateVersion');
