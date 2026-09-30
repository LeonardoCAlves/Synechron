# Engineering Practices

This repository is designed to make the team's reasoning inspectable, not to maximize ceremony.

## Change Lifecycle

1. State the user or system outcome and measurable acceptance criteria.
2. Identify affected contracts, invariants, failure modes, and data sensitivity before implementation.
3. Write or update the executable contract first; regenerate checked-in schemas in the same change.
4. Implement the smallest use case behind an application boundary. Keep protocol and persistence details in adapters.
5. Add tests at the lowest layer that proves the behavior, plus a boundary test when serialization or protocol behavior is involved.
6. Review compatibility, security, observability, and rollback or migration implications.
7. Run `npm run check`; record any intentionally deferred production work.

## Design Review Questions

- Which invariant is enforced, and at which boundary?
- What is the failure contract for invalid input, missing data, timeouts, and partial completion?
- Which dependency can be replaced without changing the domain use case?
- Is this change backward compatible for persisted records and MCP clients?
- What information is exposed to a model or tool host, and is every field necessary?
- How would the team observe, roll back, and recover this change in production?

## Quality Policy

- `strict` TypeScript and exact optional property semantics are enabled.
- Public records and tool arguments are schema-validated; unknown fields are rejected.
- Generated contract changes are reviewed alongside source changes.
- The pull request must pass formatting, contract freshness, type checking, tests, and build.
- Dependency updates are reviewed for runtime support, license, maintenance, and known advisories.

## Production Readiness Is a Separate Decision

This reference deliberately omits durable storage, identity, access control, audit trails, rate limiting, secret management, and operational deployment. A team must design and test those controls before adapting the sample to real organizational records.
