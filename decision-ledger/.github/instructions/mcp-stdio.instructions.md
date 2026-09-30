---
description: Rules for implementing and reviewing the MCP stdio adapter.
applyTo: 'src/mcp/**/*.ts'
---

- Keep stdout exclusively for MCP protocol traffic. Send diagnostics to stderr.
- Keep protocol translation in this adapter; delegate business rules to application services.
- Do not launch MCP stdio through a command that writes banners or logs to stdout; stdout is the JSON-RPC channel.
- Validate identifiers and all client-controlled arguments, including resource-template variables.
- Return useful, bounded error messages and never include secrets or stack traces in tool results.
- Treat tool descriptions and generated schemas as a public compatibility surface.
- Require explicit product-level authorization and human approval before adding sensitive writes.
