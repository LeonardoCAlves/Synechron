# Architecture

## Context

The project demonstrates how to design independently deployable catalog and order services without sharing a database or coupling one service to another's persistence model.

## Target topology

```text
Client
  | HTTP
  +----------------------> catalog-service ----> catalog database
  |
  +----------------------> order-service ------> order database
                               |                       ^
                               | transactional outbox  | inbox + catalog projection
                               v                       |
                             Kafka --------------------+
```

The catalog publishes product-created and product-updated events. The order service consumes them into a local catalog projection and uses that projection when accepting orders. Orders capture a product name and unit price snapshot so historical records remain stable when the catalog changes.

## Boundaries and invariants

- A service is the sole owner of its tables and migration lifecycle.
- A service never reads another service's database. Cross-service data is obtained through an API or an owned projection populated by events.
- Product names and prices are validated by the catalog owner. Money is represented as integer minor units, never floating-point values.
- An order contains at least one line; each quantity is positive and bounded; each unit price is non-negative; the total must not overflow.
- Repeated product identifiers within one order are rejected; they are never silently double-counted.
- Events have a stable ID, type, schema version, occurrence time, and aggregate ID. Event payloads are additive-compatible within a version.
- The transactional outbox closes the database/Kafka dual-write gap. It provides at-least-once delivery, so consumers must deduplicate.
- A consumer commits its Kafka offset only after the projection and inbox record commit together in its database transaction.

## Reliability and failure semantics

- HTTP APIs validate input, cap request size, apply deadlines, and return stable machine-readable error responses.
- Idempotency keys are scoped to an operation and persist the request fingerprint and original result. Reusing a key with a different request is a conflict.
- Outbox retries use bounded exponential backoff with jitter, expose oldest-event age and retry metrics, and route repeatedly failing events to an explicit operational recovery path.
- Consumers tolerate duplicate and out-of-order events. Projection updates use event versions or aggregate sequence numbers where ordering matters.
- Distributed workflows use explicit states and compensation/recovery rules; Kafka transactions are not treated as a substitute for a business saga.
- Readiness checks verify dependencies; liveness checks do not turn temporary dependency failure into restart loops.

## Delivery milestones

1. **Foundation (current):** domain invariants, event envelope, architecture, and engineering standards.
2. **Service boundaries:** versioned HTTP contracts and application use cases, with in-memory ports and boundary tests.
3. **Durability:** service-owned PostgreSQL schemas, migrations, transactional outbox, and integration tests.
4. **Event flow:** Kafka producer/consumer adapters, inbox deduplication, catalog projection, retry and replay behavior.
5. **Local operations:** Compose environment, health/readiness, structured logs, metrics, and end-to-end smoke tests.
6. **Release readiness:** CI, dependency/security review, compatibility checks, operational runbooks, and an explicit production-readiness review.

See [ADR 0001](adr/0001-service-boundaries-and-kafka.md) for the initial decision and trade-offs.
