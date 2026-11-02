ALTER TABLE outbox_events ADD COLUMN aggregate_version BIGINT NOT NULL DEFAULT 1 CHECK (aggregate_version > 0);

CREATE INDEX outbox_events_aggregate_version_idx
    ON outbox_events (aggregate_id, aggregate_version, created_at, id)
    WHERE published_at IS NULL;
