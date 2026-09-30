# Architecture

## Context

Teams need a durable, reviewable record of architectural choices. Decision Ledger models one decision record and exposes three operations through an MCP server so an AI host can read and propose records through an explicit protocol boundary.

## Structure

```text
MCP host
   | stdio / MCP
   v
MCP adapter (src/mcp/server.ts)
   | application calls
   v
DecisionService (src/application/decision-service.ts)
   | DecisionRepository port
   v
PostgresDecisionRepository (src/infrastructure/)
   | parameterized SQL via pg Pool
   v
PostgreSQL (Compose volume in local development)
```

`src/contracts.ts` is the executable contract source. Zod validates untrusted inputs, infers TypeScript types, and generates the checked-in JSON Schema artifacts. The application layer owns use cases; the adapter translates MCP requests and results; the repository port keeps storage replaceable. The PostgreSQL schema is evolved only through versioned migrations. The in-memory repository remains available for fast isolated tests.

## Invariants

- Decision IDs are UUIDs and creation timestamps are UTC ISO 8601 values.
- A record has a bounded title, context, decision, and consequences, plus one lifecycle status.
- Unknown record fields are rejected at the contract boundary.
- A missing record is represented by a typed application error and an actionable MCP result.
- MCP stdio output is reserved for protocol messages; diagnostics go to stderr.

## Design Choices

- **MCP over an HTTP API:** this example teaches the AI integration boundary directly. It does not claim MCP replaces REST for general service-to-service APIs.
- **PostgreSQL adapter:** persists records across process restarts and uses parameterized queries; migrations use a separate owner connection from the least-privilege app role.
- **Compose deployment:** PostgreSQL is health-checked; the stdio MCP server runs as a non-root container process and is launched on demand by the host.
- **Local-only topology:** the sample has one database instance and no HA, replication, automated backups, or multi-region recovery.
- **Generated contracts:** checked-in JSON schemas support review and consumers that do not execute TypeScript; CI detects drift from the Zod source.

See [ADR 0001](adr/0001-mcp-as-the-interaction-boundary.md) for the protocol decision and alternatives considered.
