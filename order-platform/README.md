# Order Platform

Order Platform is a Go training project for advanced software engineering. It will model a product catalog and order workflow as independently deployable microservices integrated through versioned Kafka events.

## Project status

The project has implemented its foundation, service-boundaries, PostgreSQL durability, event-flow, local-operations, and CI milestones. Catalog and order services persist to separate PostgreSQL databases; outbox events are published to Kafka and the order service consumes catalog events into an inbox-deduplicated projection. The local Compose stack and CI smoke test exercise the end-to-end flow. Environment-specific production security and operational approval remain out of scope.

## Architecture direction

- `catalog-service` owns product definitions and publishes product lifecycle events.
- `order-service` owns orders and maintains its own read projection of catalog products from Kafka events.
- Each service owns its persistence and schema; local development may share a PostgreSQL server but must use separate databases and credentials.
- PostgreSQL transactional outboxes publish events at least once. Consumers use an inbox/deduplication record and commit Kafka offsets only after their database transaction commits.
- Orders snapshot product name and price at placement time; catalog changes do not rewrite historical orders.
- Services expose versioned HTTP APIs and versioned event contracts. Kafka is asynchronous integration, not a distributed transaction coordinator.

See the [architecture](docs/architecture.md), [engineering practices](docs/engineering-practices.md), [testing strategy](docs/testing.md), and [threat model](docs/security.md).

## Requirements and checks

- Go 1.27.3 or later and Docker Compose v2 are required for the full local stack.

Run the implemented checks from this directory:

```powershell
go mod download
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
go vet ./...
go test -race -cover ./...
go build ./...
```

Run the complete local stack from the repository root with `docker compose -f order-platform/compose.yaml up --build --wait`. It provides PostgreSQL, Kafka, catalog service, and order service. Run `bash order-platform/scripts/smoke-test.sh` to verify catalog-to-order event propagation. The individual services can also be run with `go run ./cmd/catalog-service` and `go run ./cmd/order-service` after configuring their separate `DATABASE_URL` values and Kafka broker/topic variables.

The HTTP routes and request/response semantics are described in [service boundaries](docs/service-boundaries.md). Database ownership and migrations are documented in [persistence](docs/persistence.md), Kafka delivery in [event flow](docs/event-flow.md), and local operations/release checks in [operations](docs/operations.md).

## Scope

This is an educational reference, not a production service. Do not use it for real customer or payment data. Authentication, authorization, tenant isolation, payment processing, operational deployment, and disaster recovery require separate design and validation before production adaptation.
