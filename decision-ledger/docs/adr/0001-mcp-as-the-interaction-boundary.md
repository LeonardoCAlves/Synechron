# ADR 0001: Use MCP as the AI Interaction Boundary

- Status: Accepted
- Date: 2026-09-29
- Decision owners: Architecture group

## Context

The teaching project needs a small interface that an AI host can discover and call without coupling the domain model to a specific model vendor. It should also demonstrate that tools, resources, and prompts have distinct roles.

## Decision

Expose decision use cases through the Model Context Protocol using its TypeScript SDK and stdio transport. Keep domain and application code independent of MCP. Use JSON Schema artifacts generated from the Zod contract definitions.

## Alternatives Considered

- **HTTP REST only:** a strong general-purpose service boundary, but does not directly demonstrate AI host discovery and MCP capabilities.
- **Vendor-specific function calling:** simpler for one host, but couples the integration to one model API and its schema conventions.
- **Direct filesystem access from the agent:** easy to prototype, but gives no explicit operation contract or protocol-level boundary.

## Consequences

- MCP tools provide model-invoked operations; resources provide read-only context; prompts provide a reusable user-invoked workflow.
- The stdio transport is suitable for local hosts but is not a network deployment design.
- Contract changes must be reviewed as client-facing compatibility changes.
- MCP does not provide business authorization or user approval by itself; the host and production system must enforce those controls.
