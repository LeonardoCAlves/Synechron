---
description: TypeScript implementation and test conventions for this repository.
applyTo: 'src/**/*.ts,tests/**/*.ts,scripts/**/*.ts'
---

- Keep strict compiler settings enabled; model optional fields explicitly.
- Keep domain and application modules free of MCP SDK imports.
- Reuse schemas and inferred types from `src/contracts.ts`; avoid parallel handwritten interfaces.
- Prefer typed errors for expected application outcomes and preserve actionable boundary responses.
- Test externally observable behavior and contract edge cases.
