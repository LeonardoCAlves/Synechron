# Engineering Practices

## Change lifecycle

1. State the outcome, measurable acceptance criteria, and affected service/API/event contracts.
2. Identify invariants, data sensitivity, consistency requirements, retry behavior, and partial-failure modes.
3. Update executable HTTP/event contracts and compatibility tests before changing behavior.
4. Implement the use case in its owning service. Keep HTTP, Kafka, and PostgreSQL concerns in adapters.
5. Test domain and application behavior without external systems, then test serialization and persistence at the boundary.
6. Review duplicate delivery, ordering, idempotency, timeout, observability, migration, rollback, and replay implications.
7. Run formatting, static analysis, unit tests, integration tests, and the build; record any deferred production work.

## Go quality policy

- Use the Go standard library unless a dependency has a clear, reviewed benefit.
- Prefer small interfaces at the consumer boundary and explicit dependency construction.
- Pass `context.Context` through I/O and honor cancellation and deadlines.
- Return errors with useful context; do not discard errors or convert failures into success-shaped defaults.
- Use `errors.Is`/`errors.As` for error contracts and structured logs without secrets or sensitive payloads.
- Keep public HTTP/event records validated; reject unknown fields where forward compatibility permits.
- Run `gofmt`, `go vet`, unit tests, race tests, and dependency/security review in CI.

## Distributed-systems review

- Assume messages are delivered at least once. Define deduplication keys and retention.
- Never publish an event independently of the database transaction that commits its source state.
- Commit a Kafka offset only after the consumer's database transaction commits.
- Define ordering scope, replay behavior, poison-message handling, and schema compatibility.
- Avoid synchronous cross-service calls on critical write paths unless the consistency requirement justifies them.
- Use timeouts, bounded retries with jitter, backpressure, and observable dead-letter/recovery procedures.

## Production readiness is separate

This training reference does not provide organizational identity, access control, tenant isolation, secret management, production deployment, or disaster recovery. Those controls require explicit design and verification before real data is introduced.
