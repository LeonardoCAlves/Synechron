ALTER TABLE outbox_events ADD COLUMN lease_until TIMESTAMPTZ;
ALTER TABLE outbox_events ADD COLUMN dead_lettered_at TIMESTAMPTZ;
ALTER TABLE catalog_product_projection ADD COLUMN aggregate_version BIGINT NOT NULL DEFAULT 1 CHECK (aggregate_version > 0);

UPDATE outbox_events
SET payload = jsonb_set(payload, '{aggregateVersion}', '1'::jsonb, true)
WHERE NOT (payload ? 'aggregateVersion');

CREATE TABLE inbox_events (
    event_id TEXT PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    aggregate_version BIGINT NOT NULL CHECK (aggregate_version > 0),
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX inbox_events_aggregate_idx
    ON inbox_events (aggregate_id, aggregate_version);
