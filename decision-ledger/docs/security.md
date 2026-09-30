# Security Notes and Threat Model

## Assets

- Architecture decision content, which may reveal internal systems, controls, or future plans.
- Tool inputs and outputs passed between the MCP host, model, and server.
- The integrity of stored decision records and their lifecycle status.

## Trust Boundaries

- MCP arguments are untrusted, even when a host performs validation.
- Model-generated tool calls can be mistaken, manipulated by prompt injection, or requested without sufficient user context.
- Resource text and tool results may be sent to a model by the host.

## Current Controls

- Zod schemas validate tool inputs and reject unknown fields.
- The application uses parameterized SQL; runtime database access uses a restricted role, separate from the migration owner.
- The MCP container runs as a non-root user and PostgreSQL is not exposed to the public internet by this local Compose setup.
- Tool descriptions distinguish read operations from record creation.
- stdio diagnostics are sent to stderr so logs cannot corrupt the protocol stream.

## Known Gaps

- No user authentication, authorization, tenant isolation, or approval workflow.
- No durable audit log or tamper-evident history.
- No rate limiting, resource limits, or production-grade availability controls.
- Local Compose defaults are intentionally weak and are not production secrets.
- The database volume is not backed up or encrypted by this sample.

## Safe Use

Run only in a trusted local development environment. Change the default credentials before any shared use. Do not expose the server to a network, add secrets to records, or treat model-generated content as approved architecture. A production deployment requires an explicit threat model, host identity, per-operation authorization, human approval for writes, durable audit logging, encrypted and tested backups, and data-retention controls.

Report suspected vulnerabilities privately to the maintainers rather than opening a public issue with exploit details.
