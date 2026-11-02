# Security Notes and Threat Model

## Assets

- Product and order records, including commercially sensitive prices and customer information that may be introduced in later exercises.
- Kafka messages, consumer offsets, outbox/inbox records, and service credentials.
- Integrity and ordering of product and order lifecycle events.

## Trust boundaries

- HTTP request bodies and headers are untrusted, including idempotency keys.
- Kafka messages may be duplicated, delayed, replayed, malformed, or produced by a misconfigured principal.
- Each service is a separate trust and failure boundary even when local development shares a host.

## Planned controls

- Strict request and event validation, bounded payloads, parameterized SQL, and explicit error mapping.
- Separate database users/databases and Kafka principals per service.
- Secret injection at runtime; no credentials in source, logs, events, or checked-in local files.
- Least-privilege Kafka topic ACLs and PostgreSQL grants.
- Structured logs and metrics that exclude customer payloads and secrets.

## Known gaps

The training services now expose HTTP and Kafka endpoints, but they have no authentication, authorization, tenant isolation, or rate limiting. Local Compose uses single-node plaintext Kafka and disposable local credentials. Production encryption/key management, audit retention, production ACLs, deployment controls, and backup/recovery procedures are not implemented.

## Safe use and production gate

Do not put real customer, payment-card, credential, or confidential business data in this project. Before production adaptation, define service identity, per-operation authorization, tenant isolation, encryption and key rotation, audit and retention policy, Kafka/PostgreSQL backup and restore, resource limits, incident response, security review, and deployment/rollback ownership. Restrict access and retention for Kafka dead-letter topics because they preserve source event payloads.
