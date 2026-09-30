---
name: contract-first-change
description: 'Use when adding or changing a Decision Ledger capability, MCP tool, resource, prompt, or decision record field. Guides contract-first implementation, compatibility review, tests, and generated JSON Schema updates.'
---

# Contract-First Change

## Procedure

1. State the behavior and acceptance criteria. Identify whether the change affects a domain invariant, persisted data, or an MCP client contract.
2. Update or add the Zod schema and inferred type in `src/contracts.ts`. Reject unknown properties unless there is a documented compatibility reason not to.
3. Update `mcpToolContracts` when adding or changing a tool. Keep descriptions specific about effects, inputs, and failure conditions.
4. Implement the use case in `src/application/`; keep protocol and storage concerns behind their existing boundaries.
5. Add behavior-focused tests for valid input, invalid input, and the relevant failure path.
6. Regenerate checked-in artifacts with `npm run contracts:generate` and verify them with `npm run contracts:check`.
7. Review compatibility, authorization, privacy, observability, and production-readiness implications.
8. Run `npm run check` and report any gate that could not be run.

## Compatibility Rules

- Treat tool names, input schemas, resource URIs, prompt names, and output shapes as public interfaces.
- Prefer additive changes. If a breaking change is required, explain the migration and record the decision in an ADR.
- Never describe an in-memory operation as durable.
