# Order Platform

Order Platform is a Go training project for advanced software engineering. It will model a product catalog and order workflow as independently deployable microservices integrated through versioned Kafka events.

## Project status

The project has completed its foundation, service-boundaries, and PostgreSQL durability milestones. Catalog and order services now persist to separate PostgreSQL databases, and each write commits its state and versioned outbox event atomically. Kafka transport, a local Compose environment, and CI remain follow-up milestones; events are recorded but are not yet published.

## Architecture direction

- `catalog-service` owns product definitions and publishes product lifecycle events.
- `order-service` owns orders and maintains its own read projection of catalog products from Kafka events.
- Each service owns its persistence and schema; local development may share a PostgreSQL server but must use separate databases and credentials.
- PostgreSQL transactional outboxes publish events at least once. Consumers use an inbox/deduplication record and commit Kafka offsets only after their database transaction commits.
- Orders snapshot product name and price at placement time; catalog changes do not rewrite historical orders.
- Services expose versioned HTTP APIs and versioned event contracts. Kafka is asynchronous integration, not a distributed transaction coordinator.

See the [architecture](docs/architecture.md), [engineering practices](docs/engineering-practices.md), [testing strategy](docs/testing.md), and [threat model](docs/security.md).

## Requirements and checks

- Go 1.27 or later
- PostgreSQL is required to run either service. Kafka will be required when event transport is implemented.

Run the implemented checks from this directory:

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test ./...
go build ./...
```

Run either service with `go run ./cmd/catalog-service` (127.0.0.1:8081) or `go run ./cmd/order-service` (127.0.0.1:8082). Set `HTTP_ADDR` and `DATABASE_URL` to configure the listen address and service-owned PostgreSQL database. The order service starts with an empty catalog projection; order placement becomes available after a projection is populated by the Kafka consumer in a later milestone.

The current HTTP routes and request/response semantics are described in [service boundaries](docs/service-boundaries.md).
Database ownership, migrations, outbox records, and integration-test setup are described in [persistence](docs/persistence.md).

## Scope

This is an educational reference, not a production service. Do not use it for real customer or payment data. Authentication, authorization, tenant isolation, payment processing, operational deployment, and disaster recovery require separate design and validation before production adaptation.
