# Security Notes

Commerce RAG is a training reference and is not approved for customer or production data.

## Threats to Address

- **Cross-scope retrieval:** derive the authorized store/tenant scope from authenticated identity, not from an unchecked request field. Enforce it in the data query and again at the application boundary.
- **Prompt injection in sources:** treat catalog and policy text as untrusted input. Instructions embedded in retrieved content must never override system policy or tool authorization.
- **Unsupported claims:** cite only retrieved passages, abstain when evidence is absent, and evaluate groundedness. A citation alone does not prove that the cited text entails the answer.
- **Sensitive data leakage:** minimize data sent to embedding and generation providers; define redaction, retention, region, and provider-training settings before any real data use.
- **Unsafe source links:** allow-list citation schemes/hosts and escape display content; do not render retrieved HTML as trusted markup.
- **Resource exhaustion:** cap query size, retrieved passage count, chunk size, model tokens, concurrent work, retries, and per-request cost.
- **Index lifecycle:** ensure updates, unpublishing, deletion, and scope changes propagate to every lexical and vector index within a defined freshness target.

## Required Before Production

Design and verify authentication, authorization, tenant isolation, secrets management, encryption, audit/retention policy, provider data-processing terms, abuse controls, backups, incident response, and deployment-specific observability. Threat-model ingestion and query paths separately. Never log raw customer questions, full retrieved passages, credentials, or provider payloads by default.