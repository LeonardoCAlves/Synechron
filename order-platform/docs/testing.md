# Testing Strategy

## Layers

| Layer | Evidence expected | Main risk covered |
| --- | --- | --- |
| Domain | Pure Go unit tests | Product/order invariants, money arithmetic, lifecycle rules |
| Application | Port-driven service tests | Idempotency, not-found, conflict, and failure semantics |
| HTTP | Handler tests with `httptest` | Validation, status codes, serialization, request limits |
| Event contract | Golden/schema compatibility tests | Versioning, required fields, unknown fields, additive changes |
| PostgreSQL | Disposable integration database | Constraints, migrations, outbox/inbox atomicity, concurrency |
| Kafka | Disposable broker integration tests | Redelivery, offset commit ordering, partitioning, replay |
| System | Compose smoke tests | Service discovery, health, end-to-end eventual consistency |

## Required local checks

```powershell
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

The service-boundaries milestone includes `httptest` coverage for JSON validation, request-size limits, HTTP status/error contracts, idempotency replay and conflicts, product snapshots, and orders waiting for a catalog projection.

PostgreSQL/Kafka integration tests must use disposable local infrastructure and never truncate or mutate a shared environment. Every test that creates infrastructure must clean it up even when assertions fail. PostgreSQL adapter tests use dedicated connection-string environment variables, create a unique schema per test, and drop only that schema during cleanup.

```powershell
$env:ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL = "postgres://user:password@localhost:5432/catalog_test?sslmode=disable"
$env:ORDER_PLATFORM_ORDERS_TEST_DATABASE_URL = "postgres://user:password@localhost:5432/orders_test?sslmode=disable"
go test ./internal/postgres
```

The database URLs above must point to disposable test databases where the test user can create and drop schemas. If either variable is unset, its integration test is skipped.

## Distributed behavior to prove

- The same idempotency key and payload return the original result; a changed payload with that key conflicts.
- A crash between database commit and Kafka publish does not lose the outbox event.
- A crash after Kafka publish but before marking the outbox row may duplicate delivery; the consumer applies the effect once.
- A consumer failure before database commit does not commit the Kafka offset.
- Replaying an event and delivering it out of order do not corrupt the product projection.
- Concurrent requests do not create duplicate orders or violate inventory/business invariants.
