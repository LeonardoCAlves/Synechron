# Contributing

## Before Opening a Pull Request

- Read [the architecture](docs/architecture.md), [engineering practices](docs/engineering-practices.md), and [security notes](docs/security.md).
- For behavior changes, update tests and the executable contract in the same pull request.
- Regenerate JSON Schema artifacts with `npm run contracts:generate`.
- Run `npm run check` and include the result in the pull request description.
- Document significant boundary or persistence decisions as an ADR.

## Pull Request Expectations

Describe the problem, the chosen approach, alternatives considered, compatibility impact, and security impact. Keep generated artifacts in sync. Do not include secrets or real confidential decision records in fixtures.

## Development

Use Node.js 22 or later, Docker Compose, and install dependencies with `npm ci`. The MCP server is local stdio-only; use the VS Code configuration or MCP Inspector to exercise it. Run PostgreSQL integration tests against a migrated test database; never point them at a shared or production database because the suite truncates its `decisions` table.
