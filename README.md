# Synechron Engineering Training

This repository is a hands-on training resource for the Synechron team in Ontario, Canada. It brings together engineering projects and learning materials that the team can explore, run, test, and extend.

Each project lives in its own directory and includes project-specific documentation and setup instructions. The repository will grow as additional training projects are added.

## Projects

### Order Platform

A Go microservices reference for a product catalog and order workflow. It is being built to demonstrate service-owned data, versioned domain events, Kafka-based asynchronous integration, idempotency, and production-grade testing practices.

- [Project README and implementation status](order-platform/README.md)
- [Architecture overview](order-platform/docs/architecture.md)
- [Engineering practices](order-platform/docs/engineering-practices.md)
- [Testing strategy](order-platform/docs/testing.md)
- [Security notes and threat model](order-platform/docs/security.md)

### Decision Ledger

A contract-first architecture decision service that stores Architecture Decision Records (ADRs) and exposes its use cases through the Model Context Protocol (MCP). It demonstrates executable contracts, layered architecture, PostgreSQL persistence, automated testing, CI, and security trade-offs.

- [Project README and setup instructions](decision-ledger/README.md)
- [Architecture overview](decision-ledger/docs/architecture.md)
- [Engineering practices](decision-ledger/docs/engineering-practices.md)
- [Security notes and threat model](decision-ledger/docs/security.md)
- [Testing guide](decision-ledger/docs/testing.md)
- [ADR 0001: MCP as the interaction boundary](decision-ledger/docs/adr/0001-mcp-as-the-interaction-boundary.md)

## Working with This Repository

Start with the README in the project you want to explore. Follow that project's prerequisites, setup steps, and security guidance. Keep project-specific code and documentation in its own directory so multiple training projects can coexist here.

These projects are for learning and engineering practice. Review each project's limitations before using it beyond a local training environment.