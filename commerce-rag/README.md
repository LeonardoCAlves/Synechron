# Commerce RAG

Commerce RAG is an advanced software engineering training project for a shopping assistant grounded in an e-commerce catalog and operational knowledge such as shipping, returns, and product-care policies.

The first milestone is intentionally provider-independent. It implements the application boundary for retrieval-augmented answers, checks retrieved scope, constrains citations to retrieved passages, and abstains when evidence is missing. The project does not yet include a real embedding model, vector database, LLM provider, or HTTP service.

## Prerequisites

- JDK 25 or later
- Maven 3.9 or later

## Run the Current Milestone

```powershell
mvn test
```

The unit suite uses fakes and does not require model credentials, a database, or external services. Maven may download build dependencies on the first run.

## Current Capabilities

- A retrieval port and answer-generation port with typed application contracts
- Defense-in-depth filtering by store/tenant scope
- Bounded context passed to the generator
- Citation allow-listing against retrieved passages
- Fail-closed abstention when evidence or valid citations are absent
- Unit tests for invalid input, scope leakage, citation behavior, and no-evidence cases

## Roadmap

1. Add versioned catalog and policy ingestion with deterministic chunk IDs, content hashes, and idempotent re-indexing.
2. Add PostgreSQL with `pgvector` and full-text search for hybrid retrieval, plus migrations and disposable integration tests.
3. Add Spring Boot HTTP delivery and Spring AI embedding/generation adapters with deadlines, bounded retries, token/cost budgets, and structured outputs.
4. Add a background ingestion worker, including readiness, observability, and graceful shutdown.
5. Add a golden evaluation set for retrieval quality, citation coverage, groundedness review, latency, and cost regression.
6. Add local Compose operations and CI gates for unit, integration, contract, evaluation, and security checks.

The current core targets Java 25 with Maven and JUnit. Spring Boot and Spring AI are reserved for the delivery and provider-adapter milestones, keeping the application behavior testable without infrastructure. The project is complete only when these milestones are implemented and verified. See the [architecture](docs/architecture.md), [testing strategy](docs/testing.md), and [security notes](docs/security.md) for the intended boundaries and acceptance criteria.

## Scope

This is a training project, not a production shopping or customer-support system. Do not send customer data, payment information, credentials, or confidential business content to model providers. Production use requires an explicit identity, authorization, privacy, retention, incident response, and operational review.