# Repository Instructions

- Keep all project code, documentation, tests, and agent assets in English.
- Preserve the domain/application/MCP/infrastructure boundaries described in [the architecture](../docs/architecture.md).
- Treat `src/contracts.ts` as the source of truth for decision and MCP tool schemas. Regenerate `contracts/` with `npm run contracts:generate` after contract changes.
- Treat `migrations/` as the only supported way to evolve the PostgreSQL schema; never modify a migration that may have been applied.
- Keep runtime database credentials separate from migration-owner credentials; use parameterized SQL.
- Validate untrusted inputs at boundaries; do not leak implementation details or secrets through MCP responses.
- Never write logs to stdout in the stdio server; use stderr.
- Add or update behavior-focused tests and run `npm run check` before declaring a change complete.
- This project persists to PostgreSQL but has no business authentication, authorization, or production backup strategy. Do not imply otherwise.

The MCP TypeScript SDK v2 uses `@modelcontextprotocol/server`, Zod v4, and stdio patterns documented at [the SDK server guide](https://ts.sdk.modelcontextprotocol.io/v2/get-started/first-server.html), [tools](https://ts.sdk.modelcontextprotocol.io/v2/servers/tools.html), [resources](https://ts.sdk.modelcontextprotocol.io/v2/servers/resources.html), and [prompts](https://ts.sdk.modelcontextprotocol.io/v2/servers/prompts.html). Follow the [official MCP specification](https://modelcontextprotocol.io/specification/latest) when changing protocol behavior.
