# Architecture

## Goal

Build a shopping assistant that answers product and policy questions from approved e-commerce sources, returns verifiable citations, and abstains when retrieval does not provide enough evidence. The training goal is as much about dependable software boundaries as it is about retrieval-augmented generation.

## Current Boundary

The first milestone contains only the Java application core in `src/main/java/com/synechron/training/commercerag`:

1. Validate the question and store/tenant scope.
2. Ask a `Retriever` for a bounded set of passages.
3. Reject passages returned for a different scope and cap the context sent onward.
4. Ask an `AnswerGenerator` to produce an answer and identify supporting passage IDs.
5. Return citations only for retrieved passages; abstain if there is no evidence or no valid citation.

The `Retriever` and `AnswerGenerator` interfaces are adapter boundaries, not implementations. Unit tests use fakes so the use-case contract is testable without a model, database, network, or credentials. The core targets Java 25 and uses records for immutable data contracts; Spring Boot and Spring AI are introduced with the HTTP and provider-adapter milestones rather than coupled to this first use case.

## Target Architecture

- **Sources:** product descriptions, product attributes, shipping terms, returns policies, and care guides. Each source has ownership, scope, version, status, and update time.
- **Ingestion:** validate and normalize source records, split content with metadata-preserving chunking, compute stable content hashes, and enqueue idempotent indexing work.
- **Index:** PostgreSQL with `pgvector` for dense retrieval and full-text search for lexical retrieval. Apply scope and publication filters before returning candidates; fuse ranked results and optionally rerank a small candidate set.
- **Application:** use-case services depend on retrieval, generation, clock, and telemetry ports. Provider and storage details stay in adapters.
- **Generation:** use a model gateway with explicit timeout, token and cost budgets, structured output, and a prompt that treats retrieved text as untrusted data.
- **Evaluation:** a versioned golden dataset tests retrieval relevance, citation coverage, abstention, groundedness, latency, and cost independently of production traffic.
- **Delivery:** a versioned HTTP API and separate ingestion worker expose bounded operations, health checks, and observable failure states.

The first implementation should use a single deployable service with a separately runnable worker. Split services only when ownership or scaling evidence supports it. The existing Order Platform may provide a source catalog for a later integration milestone, but this project remains runnable and testable on its own.

## Invariants

- A query cannot retrieve across its authorized scope.
- Every returned citation points to a passage supplied to the generator for that answer.
- Missing or unusable evidence produces an explicit abstention, never a success-shaped guess.
- Reprocessing an unchanged source version does not create duplicate active chunks.
- Unpublishing or deleting a source removes its chunks from retrieval within a documented freshness bound.
- Retrieval and generation have explicit deadlines and bounded resource budgets.

See [ADR 0001](adr/0001-provider-independent-rag-core.md) for the initial boundary decision and the [roadmap](../README.md#roadmap) for implementation milestones.