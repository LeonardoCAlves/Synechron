# Testing Strategy

## Layers

| Layer | Evidence expected | Main risk covered |
| --- | --- | --- |
| Domain | Pure Go unit tests | Product/order invariants, money arithmetic, lifecycle rules |
| Application | Port-driven service tests | Idempotency, not-found, conflict, and failure semantics |
| HTTP | Handler tests with `httptest` | Validation, status codes, serialization, request limits |
| Event contract | Golden/schema compatibility tests | Versioning, required fields, unknown fields, additive changes |
| PostgreSQL | Disposable integration database | Constraints, migrations, outbox/inbox atomicity, concurrency |
| Kafka | Disposable broker smoke test | Catalog publication, projection update, retryable eventual consistency |
| System | Compose smoke tests | Service discovery, health, end-to-end eventual consistency |

## Required local checks

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test ./...
go test -race ./...
```

HTTP boundary tests cover JSON validation, request-size limits, HTTP status/error contracts, idempotency replay and conflicts, readiness versus liveness, product snapshots, and orders waiting for a catalog projection. Pure event-flow tests cover bounded retry delays; PostgreSQL integration tests verify inbox deduplication and monotonic projection updates.

PostgreSQL/Kafka integration tests must use disposable local infrastructure and never truncate or mutate a shared environment. Every test that creates infrastructure must clean it up even when assertions fail. PostgreSQL adapter tests use dedicated connection-string environment variables, create a unique schema per test, and drop only that schema during cleanup.

```powershell
$env:ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL = "postgres://user:password@localhost:5432/catalog_test?sslmode=disable"
$env:ORDER_PLATFORM_ORDERS_TEST_DATABASE_URL = "postgres://user:password@localhost:5432/orders_test?sslmode=disable"
go test ./internal/postgres
```

The database URLs above must point to disposable test databases where the test user can create and drop schemas. If either variable is unset, its integration test is skipped. Kafka event propagation is validated by the Compose smoke test: it creates a catalog product and retries order placement until the consumer has committed its projection. See [local operations](operations.md) for running the smoke test and CI flow.

## Distributed behavior to prove

- The same idempotency key and payload return the original result; a changed payload with that key conflicts.
- A crash between database commit and Kafka publish does not lose the outbox event.
- A crash after Kafka publish but before marking the outbox row may duplicate delivery; the consumer applies the effect once.
- The consumer commits Kafka offsets only after the inbox and projection transaction commits; the Compose smoke test exercises the successful path.
- Replaying an event and delivering it out of order do not corrupt the product projection.
- Concurrent requests do not create duplicate orders or violate inventory/business invariants.
