# Testing Strategy

## Layers

| Layer       | Current evidence                                         | Main risk covered                                        |
| ----------- | -------------------------------------------------------- | -------------------------------------------------------- |
| Contract    | `tests/contracts.test.ts`                                | Invalid, incomplete, or unknown data crossing a boundary |
| Application | `tests/decision-service.test.ts`                         | Defaults, filtering, metadata, and not-found semantics   |
| Adapter     | MCP SDK schema registration and build                    | Protocol wiring and TypeScript compatibility             |
| Repository  | `tests/postgres-decision-repository.integration.test.ts` | SQL mapping, filtering, and persistence across instances |
| CI          | `.github/workflows/ci.yml`                               | Reproducible install and required quality gates          |

## Required Local Gate

Run `npm run check` for formatting, generated contract freshness, strict type checking, unit tests, and a production build. Run `npm run test:integration` with `TEST_DATABASE_URL` set to a migrated PostgreSQL database. CI provisions PostgreSQL, applies migrations, runs the integration tests, and builds the Docker image.

## Workshop Test Exercises

- Add a test for each boundary condition before implementing the behavior.
- Add a contract test proving extra properties are rejected.
- Replace the in-memory adapter with a fault-injecting fake and test repository failures.
- Exercise tool discovery, invalid tool input, and a not-found result with the MCP Inspector or an in-memory MCP client.
- Add a compatibility test before changing a published JSON Schema.
- For a persistent adapter, test restart durability, concurrent writes, and crash recovery.

## Test Design Rules

- Prefer observable behavior over private implementation details.
- Keep tests deterministic; inject clocks and ID generators if assertions need exact values.
- Do not contact real external systems in unit tests.
- Treat protocol serialization and generated contracts as compatibility surfaces.
