# Kafka Event Flow

The catalog service writes a product and a `catalog.product.created.v1` envelope to its PostgreSQL outbox in one transaction. Its publisher leases pending outbox rows and sends envelopes to `catalog.product-events.v1`, keyed by aggregate ID. The order service consumes that topic with the stable group `order-service.catalog-projection.v1`.

The order consumer validates the envelope and product snapshot, then records the event in `inbox_events` and updates its catalog projection in one PostgreSQL transaction. It commits the Kafka offset only after that transaction succeeds. A crash between database commit and offset commit causes redelivery; the inbox primary key prevents applying the same event twice. Each envelope has a schema `version` and an `aggregateVersion`; the latter prevents older snapshots from replacing a newer projection.

The order service also publishes order-created envelopes from its own outbox to `order.events.v1`. No consumer for order events is defined by this milestone.

## Topics and configuration

| Environment variable | Default | Purpose |
| --- | --- | --- |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated bootstrap broker addresses |
| `CATALOG_EVENTS_TOPIC` | `catalog.product-events.v1` | Catalog events published by catalog service and consumed by order service |
| `ORDER_EVENTS_TOPIC` | `order.events.v1` | Order events published by order service |

Provision the source and dead-letter topics before production deployment. Local Compose enables topic auto-creation for training. Malformed, unsupported, or invalid catalog events are copied to `catalog.product-events.v1.dead-letter` with source topic/partition/offset and a failure reason in headers. The original Kafka offset is committed only after the dead-letter write succeeds. Transient database failures do not commit offsets; the consumer waits briefly and retries. Restrict access to the dead-letter topic because its values contain the original message payload.

## Outbox retry and recovery

Publishers claim batches using `FOR UPDATE SKIP LOCKED` leases so replicas can work independently. Kafka sends are synchronous and acknowledged by all in-sync replicas. A successful send is followed by marking the row published. A process crash between those operations can cause duplicate publication; consumers must remain idempotent.

Delivery failures use bounded exponential backoff with jitter. After 12 failed attempts, an event is marked `dead_lettered_at` and is no longer retried automatically. Inspect only metadata and the error when diagnosing an event; avoid copying payloads into tickets or logs. Authorized operators can inspect the event and selectively requeue it after correcting the cause:

An unpublished lower-version event blocks later outbox events for the same aggregate, including when that earlier event has been dead-lettered. This preserves Kafka partition order; recover the earlier event before releasing later updates for that aggregate.

```sql
SELECT id, event_type, aggregate_id, attempts, last_error, created_at
FROM outbox_events
WHERE dead_lettered_at IS NOT NULL AND published_at IS NULL
ORDER BY created_at;

UPDATE outbox_events
SET attempts = 0,
    last_error = NULL,
    dead_lettered_at = NULL,
    lease_until = NULL,
    next_attempt_at = now()
WHERE id = 'reviewed-event-id'
  AND published_at IS NULL
  AND dead_lettered_at IS NOT NULL;
```

Run the update only for an explicitly reviewed event in the owning service database. For consumer poison messages, inspect the dead-letter headers and payload under the organization's retention policy; after correcting the contract or data, replay the event to the source topic with the original aggregate key. Kafka offsets are not reset as part of routine recovery.

## Delivery guarantees and limits

- Database writes and outbox inserts are atomic; Kafka publication is at least once, not exactly once.
- Inbox deduplication, projection changes, and Kafka offset commits follow database-then-offset ordering.
- Aggregate IDs are Kafka keys to preserve partition ordering; aggregate versions make snapshot projection updates monotonic when records arrive out of order.
- Local Compose runs a single broker with plaintext networking and replication factor 1. It is not representative of a production Kafka cluster.
- Authentication, TLS, ACL setup, dead-letter retention, replay authorization, partition sizing, and production alert thresholds remain deployment-specific work.
