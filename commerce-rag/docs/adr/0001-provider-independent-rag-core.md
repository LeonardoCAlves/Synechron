# ADR 0001: Provider-Independent RAG Core

## Status

Accepted

## Context

RAG examples often couple business behavior directly to a model SDK and vector store. That makes it hard to test scope isolation, citation rules, failure behavior, or evaluation without network access and credentials. It also encourages the mistaken assumption that a model-generated citation is valid merely because it looks plausible.

## Decision

Keep the initial question-answering use case behind Java `Retriever` and `AnswerGenerator` interfaces. The application validates scope, bounds retrieved context, maps citation IDs back to retrieved passages, and abstains when it has no usable evidence. Provider and persistence adapters will be added in later milestones and tested behind these contracts.

## Consequences

- Core behavior can be tested quickly and deterministically with fakes.
- The application has a clear place to enforce defense-in-depth scope and citation checks.
- The current project does not yet perform semantic search or generate answers on its own; those are explicitly future milestones.
- Citation membership is only a minimum safety check. Entailment and answer quality require evaluation beyond this core.