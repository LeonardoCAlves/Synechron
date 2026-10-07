# Order Platform

Order Platform is a Go training project for advanced software engineering. It will model a product catalog and order workflow as independently deployable microservices integrated through versioned Kafka events.

## Project status

This is the foundation milestone. It establishes the service boundaries, initial domain invariants, event envelope, and engineering workflow. The HTTP services, PostgreSQL adapters, Kafka transport, local Compose environment, and CI pipeline are planned follow-up milestones; this foundation is not yet an end-to-end runnable platform.

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
- PostgreSQL and Kafka will be required for the local integration environment in a later milestone.

Run the implemented foundation checks from this directory:

```powershell
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

## Scope

This is an educational reference, not a production service. Do not use it for real customer or payment data. Authentication, authorization, tenant isolation, payment processing, operational deployment, and disaster recovery require separate design and validation before production adaptation.
