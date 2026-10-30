# PostgreSQL Persistence

The catalog and order services each use a distinct PostgreSQL database and credentials. Both read `DATABASE_URL`; configure that variable with the service-specific connection string before starting the process. The service runs its embedded schema migration at startup and fails explicitly if it cannot connect or migrate.

The service database role needs schema-creation privileges for the initial migration and DML privileges for the service's tables. Use dedicated databases and least-privilege credentials; do not point both services at one database.

## Owned data

- The catalog database owns `catalog_products`, catalog `idempotency_requests`, and `outbox_events`.
- The order database owns `orders`, `order_lines`, order `idempotency_requests`, `catalog_product_projection`, and `outbox_events`.
- Neither service reads or writes the other service's tables. The order projection is owned by the order service and will be populated by the Kafka consumer in the event-flow milestone.
- Order lines store immutable product name, unit price, and currency snapshots. Catalog edits therefore do not change existing orders.

Each database has its own `schema_migrations` history. Migration version 1 creates the service tables and pending-event index. Migrations use a PostgreSQL advisory transaction lock and commit as one unit; the migration runner is safe to invoke at each process startup.

## Transactional outbox

Creating a product or an order commits the aggregate, idempotency record, and versioned event envelope in the same PostgreSQL transaction. If any insert fails, all writes roll back. The event is stored as JSONB with its ID, type, aggregate ID, schema version, and occurrence time, together with pending-delivery metadata.

The outbox guarantees the event is not lost between the database commit and eventual publication. It does not publish to Kafka yet; publisher retries, marking delivery, and operational handling belong to the event-flow milestone. Until then, outbox records remain pending.

## PostgreSQL integration tests

Set `ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL` and/or `ORDER_PLATFORM_ORDERS_TEST_DATABASE_URL` to disposable PostgreSQL databases and run `go test ./internal/postgres`. Each enabled test creates a cryptographically random isolated schema, runs the migrations twice, validates persistence/idempotency/outbox atomicity, and drops only its own schema during cleanup. Unset variables cause the corresponding integration test to skip. The test role must be allowed to create and drop schemas.

Never use production or shared databases for integration tests. Configure TLS appropriately for the target environment; `sslmode=disable` is suitable only for isolated local development.
