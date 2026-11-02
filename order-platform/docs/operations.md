# Local Operations and Release Checklist

## Local Compose

Install Docker Compose v2 and Go 1.27 or later. From the repository root, start the platform:

```powershell
docker compose -f order-platform/compose.yaml up --build --wait
```

The local stack uses one PostgreSQL server with separate `catalog_service` and `orders_service` databases and service-specific credentials, a single-node Kafka broker, and both HTTP services. The Compose passwords are disposable local-development values; never reuse them outside this isolated environment. PostgreSQL data persists in the `postgres-data` volume. The APIs are bound to loopback at `127.0.0.1:8081` and `127.0.0.1:8082`; Kafka is reachable from the host at `127.0.0.1:9092`, and PostgreSQL at `127.0.0.1:55432`.

Check readiness and metrics with:

```powershell
Invoke-RestMethod http://127.0.0.1:8081/readyz
Invoke-RestMethod http://127.0.0.1:8082/readyz
Invoke-WebRequest http://127.0.0.1:8081/metrics
```

Run the catalog-to-order smoke test with `bash order-platform/scripts/smoke-test.sh` in a Bash environment. To stop the stack without deleting persistent local data:

```powershell
docker compose -f order-platform/compose.yaml down
```

Use `docker compose ... down --volumes` only when intentionally deleting the disposable local PostgreSQL volume.

When running a service directly on the host against the Compose databases, use PostgreSQL port `55432` and set `DATABASE_URL`, `KAFKA_BROKERS=localhost:9092`, and the service topic variables explicitly.

## CI

The GitHub Actions workflow formats, vets, tests with the race detector and coverage, builds, verifies dependencies, runs `govulncheck`, validates Compose, and executes a Compose-based catalog-to-order smoke test. PostgreSQL adapter tests create temporary schemas in a disposable CI service database. A failed Compose smoke run emits service logs; examine them without copying event payloads or credentials into public artifacts.

## Release checklist

- Record reviewed Go and Docker image versions, dependency vulnerability results, API contracts, and database migration versions.
- Verify separate service accounts, production-specific TLS, credentials injection, network policies, Kafka ACLs, and PostgreSQL grants.
- Apply migrations using a controlled rollout and validate backup/restore and rollback before deploying application instances.
- Choose Kafka replication, retention, partitions, consumer-group capacity, and dead-letter ownership for the target environment.
- Alert on readiness failures, oldest pending outbox age, retries/dead-lettered events, consumer lag, and dead-letter growth.
- Exercise poison-message recovery and outbox requeue with an audited operator identity.
- Review authentication, authorization, tenant isolation, rate limits, privacy/retention, threat model, incident response, and disaster recovery before handling real data.

Passing CI is not a production-readiness approval. The development stack uses plaintext single-node Kafka, local-only credentials, and no application authentication.
