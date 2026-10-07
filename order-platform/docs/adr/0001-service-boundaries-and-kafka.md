# ADR 0001: Establish Catalog and Order Services with Kafka Events

- Status: Accepted
- Date: 2026-10-07
- Decision owners: Training project

## Context

The training repository needs a second project in a different language that exercises service ownership, asynchronous integration, and distributed failure handling beyond the single-process Decision Ledger.

## Decision

Use Go to implement independently deployable catalog and order services. Each service owns its data and publishes/consumes versioned events through Kafka. PostgreSQL transactional outboxes and consumer inbox/deduplication records provide reliable at-least-once processing.

## Alternatives considered

- **Modular monolith:** simpler transactions and operations, but does not exercise inter-service contracts or delivery failures.
- **Synchronous HTTP-only integration:** easy to understand, but couples order availability and latency to catalog availability and does not demonstrate asynchronous projection consistency.
- **Direct database sharing:** rejected because it violates service ownership and couples schema evolution.
- **Kafka exactly-once as a business guarantee:** rejected; transport-level transactions do not make a database update and business side effect atomic.

## Consequences

- The local and CI environments require Kafka and PostgreSQL.
- Catalog projection updates are eventually consistent; clients may need retryable responses while products are not yet projected.
- Consumers must be idempotent, event schemas must be compatible, and duplicate delivery is expected.
- Operating multiple services and Kafka increases complexity and is intentionally part of the exercise.
- This decision does not claim production readiness; authentication, authorization, tenant isolation, and operational controls are separate work.
