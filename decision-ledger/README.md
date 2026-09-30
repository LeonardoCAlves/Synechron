# Decision Ledger

Decision Ledger is a small, executable reference project for advanced software engineering practice. It records Architecture Decision Records (ADRs) and exposes the use cases through the Model Context Protocol (MCP).

The project is intentionally small in domain scope and broad in engineering surface: executable contracts, ports and adapters, failure semantics, layered tests, CI, ADRs, a threat model, and repository-level agent workflows are all visible in one place.

## Prerequisites

- Node.js 22 or later
- npm 9 or later
- Docker Desktop with Docker Compose
- VS Code with GitHub Copilot Chat for the repository instructions, custom agents, prompts, and skills

## Run It

```powershell
Copy-Item .env.example .env
# Replace both example passwords in .env before starting the stack.
npm ci
docker compose up -d postgres
$env:DATABASE_URL = 'postgresql://decision_ledger_owner:change-this-local-owner-password@localhost:5432/decision_ledger'
npm run db:migrate
npm run check
$env:TEST_DATABASE_URL = $env:DATABASE_URL
npm run test:integration
```

The MCP server uses stdio and persists records in PostgreSQL. In VS Code, open the workspace's MCP server list and start `decision-ledger`; `.vscode/mcp.json` launches the MCP container, and Compose waits for PostgreSQL health before applying migrations and starting the server. The app runs as a non-root user and connects with a restricted database role; migrations use a separate owner connection. The default credentials are for local development only.

To run the PostgreSQL-backed MCP process directly from the terminal after building and migrating, use `node dist/index.js` with `DATABASE_URL` set. For editor/host use, prefer the Docker Compose launch configuration: npm's script banner must not be written to MCP stdout.

To inspect the protocol without an AI host:

```powershell
npx @modelcontextprotocol/inspector docker compose run --rm -T mcp-server
```

## Capabilities

- Tools: `list-decisions`, `get-decision`, and `create-decision`
- Resources: `decisions://all` and the `decisions://{id}` resource template
- Prompt: `review-decision`, which asks for a structured review of a stored ADR
- Contracts: generated JSON Schema for decision records and MCP tool inputs under `contracts/`
- Quality gate: formatting, generated-contract freshness, strict TypeScript, unit tests, PostgreSQL integration tests, production build, and Docker image build in CI

## Engineering Tour

1. Read [the architecture](docs/architecture.md) and [ADR 0001](docs/adr/0001-mcp-as-the-interaction-boundary.md) before changing the boundary.
2. Trace `src/contracts.ts` into `src/application/decision-service.ts`, then into the repository adapter and MCP server.
3. Run `npm run check` and inspect the CI definition in `.github/workflows/ci.yml`.
4. Use the `contract-first-change` skill for a feature exercise; ask the `architecture-reviewer` custom agent to challenge the design.
5. Discuss the production gaps in [the threat model](docs/security.md) and the deliberate trade-offs in [the engineering guide](docs/engineering-practices.md).

See [testing strategy](docs/testing.md) for the validation layers and suggested failure cases. All repository documentation and agent assets are written in English for use as team training material.

## Project Map

```text
src/                 Domain, application use cases, adapters, and MCP entry point
contracts/           Generated public JSON Schemas
database/            PostgreSQL initialization and restricted app-role setup
migrations/          Versioned PostgreSQL schema migrations
tests/               Unit, contract, and PostgreSQL integration tests
Dockerfile           Multi-stage, non-root MCP server image
compose.yaml         Local PostgreSQL and MCP server orchestration
docs/                Architecture, engineering practices, ADRs, and test strategy
.github/             CI, review assets, agent instructions, custom agents, and skills
.vscode/mcp.json     Local MCP server registration
```

## Scope and Limitations

This project is a teaching reference, not a production service. It has durable local persistence but no user authentication, business authorization, tenant isolation, approval workflow, rate limiting, backup policy, or audit retention. Do not expose it to untrusted clients or store sensitive decision records in it.
