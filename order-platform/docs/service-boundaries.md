# Service Boundaries and HTTP API v1

The catalog and order services expose independent HTTP APIs. The application layer depends on repository/projection interfaces backed by service-owned PostgreSQL adapters.

## Running locally

From the repository root:

```powershell
go run ./cmd/catalog-service
go run ./cmd/order-service
```

Set `DATABASE_URL` to the connection string for the service's own database before starting it; the catalog and order services must use separate databases and credentials. The catalog service listens on `127.0.0.1:8081`; the order service listens on `127.0.0.1:8082`. Run them in separate terminals. Set `HTTP_ADDR` to override either address. Both expose `GET /healthz`. Each process applies its own embedded schema migration on startup.

The order service reads only its local catalog projection, which its Kafka consumer populates asynchronously. Until a requested product is projected, order creation returns `503 catalog_projection_unavailable` with `Retry-After: 1`.

## Endpoints

All request and response bodies use JSON. Unknown fields, malformed JSON, and trailing JSON values are rejected. Request bodies are limited to 1 MiB. Error responses use the stable shape:

```json
{"error":{"code":"invalid_request","message":"request body must contain one valid JSON object with supported fields"}}
```

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/products` | Create a product |
| `GET` | `/api/v1/products/{id}` | Retrieve a product |
| `POST` | `/api/v1/orders` | Place an order using the local catalog projection |
| `GET` | `/api/v1/orders/{id}` | Retrieve an order |
| `GET` | `/healthz` | Process liveness |
| `GET` | `/readyz` | PostgreSQL and Kafka dependency readiness |
| `GET` | `/metrics` | Prometheus text-format event/outbox metrics |

Product creation accepts `name`, `description`, `priceCents`, and `currency`; the service generates the product ID. Order creation accepts product IDs and quantities only—the order service obtains names, prices, and currencies from its own projection and stores those values as immutable order-line snapshots.

Both `POST` endpoints require an `Idempotency-Key` header of 1–128 bytes. Repeating the same key with the same normalized request returns the original resource and response status. Reusing the key with a different request returns `409 idempotency_conflict`. Keys are scoped to the service's database and persist with the created resource.

## Failure semantics

- Invalid JSON, unsupported fields, invalid domain values, and missing/invalid idempotency keys return `400`.
- Unsupported content types return `415`; request bodies over 1 MiB return `413`.
- Missing resources return `404`; an idempotency-key payload mismatch or inactive product returns `409`.
- A product missing from the order service's local projection returns `503` so callers can retry while asynchronous projection catches up.
- Unexpected errors return a generic `500` response; internal error details are not exposed.

Authentication, authorization, tenant isolation, and rate limiting are not implemented. Do not expose these training services to untrusted networks or use them with real customer data.
