# Testing Strategy

## Layers

| Layer | Evidence expected | Main risk covered |
| --- | --- | --- |
| Application | Fast unit tests with fake ports | Scope isolation, bounded context, citation allow-list, abstention |
| Ingestion | Property and contract tests | Stable chunk IDs, metadata preservation, idempotency, deletion semantics |
| Retrieval | Disposable PostgreSQL integration tests | Scope filters, hybrid ranking, migrations, index consistency |
| Provider adapters | Contract tests with recorded/fake responses | Timeouts, malformed output, retry limits, provider error mapping |
| Evaluation | Versioned golden dataset and report | Recall@k, MRR, citation coverage, abstention, regressions |
| API/system | HTTP contract and Compose smoke tests | Authentication boundary, request limits, health, end-to-end behavior |

## Current Checks

From this directory, with JDK 25+ and Maven 3.9+:

```powershell
mvn test
```

The current tests do not call an LLM or database. Keep retrieval relevance metrics separate from generation quality: a fluent answer cannot compensate for irrelevant retrieved passages, and a good retrieval score does not prove that a generated answer is grounded.

## Evaluation Requirements

- Keep queries, expected source IDs, relevant passages, and expected abstention behavior in a reviewed, versioned dataset.
- Report retrieval metrics (at minimum Recall@k and MRR) independently of answer checks.
- Check that citations refer to retrieved, authorized, published sources.
- Review groundedness and answer usefulness with a documented rubric; do not treat an LLM judge as a correctness oracle.
- Track p50/p95 latency, token usage, estimated cost, timeout rate, and abstention rate.
- Establish baseline thresholds before enabling a model, embedding, chunking, or ranking change.
- Use synthetic or public fixtures only; never commit customer conversations or proprietary documents.