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

PostgreSQL/Kafka integration tests must use disposable local infrastructure and never truncate or mutate a shared environment. Every test that creates infrastructure must clean it up even when assertions fail.

## Distributed behavior to prove

- The same idempotency key and payload return the original result; a changed payload with that key conflicts.
- A crash between database commit and Kafka publish does not lose the outbox event.
- A crash after Kafka publish but before marking the outbox row may duplicate delivery; the consumer applies the effect once.
- A consumer failure before database commit does not commit the Kafka offset.
- Replaying an event and delivering it out of order do not corrupt the product projection.
- Concurrent requests do not create duplicate orders or violate inventory/business invariants.
