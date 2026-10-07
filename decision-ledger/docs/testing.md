# Testing Strategy

## Layers

| Layer       | Current evidence                                         | Main risk covered                                                                         |
| ----------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Contract    | `tests/contracts.test.ts`                                | Invalid, incomplete, or unknown data crossing a boundary                                  |
| Application | `tests/decision-service.test.ts`                         | Defaults, filtering, metadata, and not-found semantics                                    |
| Adapter     | `tests/mcp-server.test.ts`                               | Discovery, validation, tools, resources, prompts, and protocol errors                     |
| Repository  | `tests/postgres-decision-repository.integration.test.ts` | SQL mapping, filtering, concurrent writes, process-exit durability, and surfaced failures |
| CI          | `.github/workflows/ci.yml`                               | Reproducible install and required quality gates                                           |

## Required Local Gate

Run `npm run check` for formatting, generated contract freshness, strict type checking, unit tests, and a production build. Run `npm run test:integration` with `TEST_DATABASE_URL` set to a migrated PostgreSQL database. CI provisions PostgreSQL, applies migrations, runs the integration tests, and builds the Docker image.

## Remaining Workshop Exercises

- Add a test for each boundary condition before implementing the behavior.
- Add a contract test proving extra properties are rejected.
- Replace the in-memory adapter with a fault-injecting fake and test repository failures.
- Add a compatibility test before changing a published JSON Schema.
- Test database crash recovery and failover in an isolated disposable environment before production use.

The automated MCP adapter tests use the SDK's in-memory transport; the MCP Inspector remains useful for a manual host-level walkthrough.

## Test Design Rules

- Prefer observable behavior over private implementation details.
- Keep tests deterministic; inject clocks and ID generators if assertions need exact values.
- Do not contact real external systems in unit tests.
- Treat protocol serialization and generated contracts as compatibility surfaces.
