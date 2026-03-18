# Knomantem: Technology Stack Decisions

**Status:** Draft
**Date:** 2026-03-18
**Author:** Knomantem Core Team

---

## Executive Summary

Knomantem is a knowledge management system that combines graph visualization, relational data modeling, AI-powered maintenance, content freshness tracking, full-text search, and team collaboration into a single product. No current competitor delivers all of these capabilities together.

This document records the technology choices made for the initial build, the alternatives that were evaluated, and the reasoning behind each decision. The overarching design principles are:

1. **Single-binary simplicity** — minimize external service dependencies to keep deployment and operations straightforward.
2. **Proven foundations** — choose mature, well-supported technologies over novel ones wherever possible.
3. **Extensibility** — ensure every choice can be swapped or scaled out later without rewriting the core system.
4. **Cross-platform reach** — deliver desktop, web, and mobile experiences from a single codebase.

---

## Decision Table (Quick Reference)

| Concern                 | Choice                              | Key Reason                                      |
| ----------------------- | ----------------------------------- | ----------------------------------------------- |
| Backend framework       | Go + Echo v4                        | Single binary, excellent concurrency, idiomatic |
| Database                | PostgreSQL 16 + pgx v5              | JSONB + relational + full-text search            |
| Search engine           | Bleve (embedded)                    | Pure Go, no external service                    |
| Markdown processing     | Goldmark                            | CommonMark-compliant, extensible AST            |
| Internal document model | Custom JSON AST (ProseMirror-style) | Rich blocks, CRDT-ready, editor-compatible      |
| Frontend                | Flutter + AppFlowy Editor           | Cross-platform, purpose-built KM editor         |
| State management        | Riverpod 3.0                        | Type-safe, testable, modern                     |
| Authentication          | JWT                                 | Stateless, standard, cross-service              |
| Authorization           | Casbin (RBAC/ABAC)                  | Flexible policy engine, Go-native               |
| Real-time               | WebSocket (presence + cursors)      | Lightweight MVP; CRDT deferred to Phase 2       |
| API style               | REST + OpenAPI 3.0                  | Simpler than GraphQL for MVP, better caching    |
| Deployment              | Single binary + PostgreSQL          | ~50 MB Docker image, one external dependency    |
| License                 | BSL 1.1                             | Self-host friendly, protects cloud offering     |

---

## Detailed Decisions

### 1. Backend: Go with Echo v4

**Decision:** Use Go as the backend language and Echo v4 as the HTTP framework.

**Alternatives Considered:**

| Alternative  | Reason for Rejection                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------- |
| Go + Gin     | 48% market share, but Echo offers better middleware chaining, more idiomatic context handling, built-in OpenAPI support, and cleaner error handling patterns. |
| Go + Fiber   | Built on fasthttp rather than net/http, which breaks compatibility with the broader Go HTTP ecosystem.  |
| Rust + Axum  | Steeper learning curve, slower iteration speed, smaller ecosystem for KM-adjacent libraries.            |
| Node + Nest  | No single-binary story; runtime dependency; weaker concurrency model for CPU-bound graph operations.    |

**Justification:**

- Go compiles to a single, statically linked binary — no runtime, no dependency tree to manage in production.
- Goroutines and channels provide a natural model for concurrent graph traversal, background AI maintenance jobs, and real-time WebSocket handling.
- Fast compilation keeps the development feedback loop tight.
- The standard library covers HTTP, JSON, crypto, and testing without third-party dependencies.
- Echo v4 adds a thin, idiomatic layer: centralized error handling via `echo.HTTPError`, composable middleware, route grouping, and first-class OpenAPI annotation support.

**Risks:**

- Echo has a smaller community than Gin; fewer Stack Overflow answers and tutorials.
- Go's type system (no sum types, limited generics) can lead to verbose code for complex domain models.

**Dependencies:** Go 1.23+, Echo v4.

---

### 2. Database: PostgreSQL 16 with pgx v5

**Decision:** Use PostgreSQL 16 as the primary data store, accessed via the pgx v5 driver.

**Alternatives Considered:**

| Alternative | Reason for Rejection                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------ |
| SQLite      | Excellent for single-user or embedded use, but limited write concurrency makes it unsuitable for team collaboration at MVP scale. Remains a candidate for a future personal/small-team mode. |
| MongoDB     | Document-native, but loses relational integrity for spaces, permissions, and user management. Operational overhead of a separate data system with no clear advantage given PostgreSQL's JSONB. |
| CockroachDB | Distributed PostgreSQL-compatible, but overkill for the initial deployment target and adds operational complexity. |

**Justification:**

- **Hybrid model:** Relational tables for structural entities (spaces, pages, users, roles, permissions) and JSONB columns for semi-structured data (document content, graph edge metadata, AI maintenance logs).
- **Full-text search:** PostgreSQL's `tsvector`/`tsquery` serves as a reliable fallback search path alongside Bleve.
- **pgx v5** is a native PostgreSQL driver — not a `database/sql` wrapper. It provides direct access to PostgreSQL-specific features: JSONB marshaling, `COPY` for bulk operations, connection pooling, prepared statement caching, and listen/notify for real-time event triggers.
- PostgreSQL 16 brings improved logical replication, better JSONB performance, and parallel query enhancements.

**Risks:**

- PostgreSQL is the single external runtime dependency. If it goes down, Knomantem goes down.
- JSONB queries can become slow without proper indexing (`GIN` indexes on JSONB paths).
- Schema migrations on JSONB content require application-level versioning.

**Dependencies:** PostgreSQL 16+, pgx v5, golang-migrate for schema migrations.

---

### 3. Search: Bleve (Embedded)

**Decision:** Use Bleve as an embedded full-text search engine compiled into the Knomantem binary.

**Alternatives Considered:**

| Alternative            | Reason for Rejection                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------------------- |
| MeiliSearch            | Excellent search quality and developer experience, but requires a separate service — violates the single-binary deployment goal. |
| Typesense              | Similar trade-off to MeiliSearch: external process, separate scaling concern.                     |
| Elasticsearch/OpenSearch | Heavy operational burden, JVM dependency, far too complex for the initial deployment target.     |
| PostgreSQL FTS only    | Adequate for basic search, but lacks faceted search, custom analyzers, and relevance tuning.      |

**Justification:**

- Bleve is a pure Go library. It compiles directly into the binary — zero external search infrastructure.
- Supports faceted search (filter by space, author, date range), custom analyzers (stemming, stop words), and field boosting.
- Index strategy: pages, content blocks, metadata, and tags are indexed. Freshness scores are incorporated as a ranking signal so that recently updated content surfaces higher.
- The Bleve index is stored on disk alongside the application data directory.

**Risks:**

- Bleve is single-node only. If Knomantem needs horizontal scaling, search must be extracted to an external engine.
- Index size grows with content volume; very large deployments (100K+ pages) may hit performance limits.
- Bleve's community is smaller than Elasticsearch's; fewer analyzers and language packs are available out of the box.

**Dependencies:** Bleve v2.

**Future Path:** If scale demands it, Bleve can be replaced with MeiliSearch or Typesense behind the same internal search interface. The search abstraction layer should be designed with this migration in mind.

---

### 4. Markdown Processing: Goldmark

**Decision:** Use Goldmark for Markdown import and export.

**Alternatives Considered:**

| Alternative   | Reason for Rejection                                                        |
| ------------- | --------------------------------------------------------------------------- |
| Blackfriday   | Not CommonMark-compliant; less predictable parsing behavior.                |
| goldmark-fenced-code only | Too narrow; we need the full CommonMark AST for reliable round-tripping. |

**Justification:**

- Goldmark is fully CommonMark-compliant, which ensures predictable parsing of user-supplied Markdown.
- The library is extensible via AST transformations — custom node types, renderers, and parsers can be plugged in without forking.
- Pure Go, maintained by yuin, widely adopted in the Go ecosystem (Hugo, Gitea, etc.).
- Used at two points in the pipeline:
  - **Import:** Goldmark parses `.md` files into its AST, which is then converted to Knomantem's internal JSON AST.
  - **Export:** The internal JSON AST is walked, and Goldmark's renderer emits standards-compliant Markdown.

**Risks:**

- Goldmark's AST and Knomantem's JSON AST are structurally different; the conversion layer must handle edge cases (nested lists, inline HTML, footnotes).
- CommonMark does not cover all of Knomantem's block types (callouts, databases, embeds). These must be serialized as fenced code blocks or HTML comments during export and re-parsed on import.

**Dependencies:** Goldmark v1.7+.

---

### 5. Internal Document Format: Custom JSON AST (ProseMirror-Inspired)

**Decision:** Store document content as a custom JSON AST modeled on ProseMirror's schema.

**Alternatives Considered:**

| Alternative       | Reason for Rejection                                                                                |
| ----------------- | --------------------------------------------------------------------------------------------------- |
| Raw Markdown      | Cannot represent rich block types (tables with formulas, embeds, database views, callouts). No natural path to real-time collaborative editing. |
| HTML              | Verbose, hard to diff, not designed for structured editing.                                         |
| Portable Text     | Sanity.io-specific; smaller ecosystem; less alignment with Flutter editor options.                   |
| Notion-style block model | Proprietary; no open specification to build against.                                          |

**Justification:**

- ProseMirror's document model is well-understood, battle-tested (used by Atlassian, The New York Times, and others), and thoroughly documented.
- The schema maps cleanly to Flutter's AppFlowy Editor, which expects a document-as-blocks structure with inline content and marks.
- The structure — `Document -> Block[] -> InlineContent[] with Mark[]` — supports operational transforms (OT) and CRDTs, providing a clear path to real-time collaboration.
- Stored as JSONB in PostgreSQL, which allows both whole-document retrieval and targeted queries against individual blocks.

**Risks:**

- A custom AST means a custom specification to maintain, document, and version.
- Conversion fidelity between Markdown and the JSON AST will be a persistent source of edge-case bugs.
- Schema evolution must be handled carefully — old documents need to remain readable as the schema grows.

**Dependencies:** None (custom implementation). Informed by the ProseMirror specification.

---

### 6. Frontend: Flutter with AppFlowy Editor

**Decision:** Build the frontend in Flutter using the AppFlowy Editor widget for the document editing experience.

**Alternatives Considered:**

| Alternative        | Reason for Rejection                                                                                 |
| ------------------ | ---------------------------------------------------------------------------------------------------- |
| React + TipTap     | Strong editor ecosystem, but limits deployment to web-only or requires Electron for desktop (large bundle, high memory). |
| Tauri + SolidJS    | Promising for desktop, but immature mobile story and no production-proven KM editor component.        |
| Swift/Kotlin native | Maximum platform fidelity, but doubles (or triples) the frontend codebase.                          |

**Justification:**

- Flutter provides genuine cross-platform output — desktop (Windows, macOS, Linux), web, and mobile (iOS, Android) — from a single Dart codebase.
- AppFlowy Editor is purpose-built for knowledge management: it supports nested documents, slash commands, customizable block types, and drag-and-drop reordering. AppFlowy itself proves that Flutter is viable for this category.
- Riverpod 3.0 for state management: type-safe providers, testable without widget trees, excellent DevTools integration.
- Material 3 design system for consistent, accessible UI components.

**Risks:**

- Flutter web performance is improving but still lags behind native JS frameworks for complex, text-heavy UIs.
- AppFlowy Editor is tightly coupled to AppFlowy's roadmap; breaking changes upstream could require significant adaptation.
- Dart's ecosystem is smaller than JavaScript's; fewer third-party packages for niche needs.

**Dependencies:** Flutter 3.27+, AppFlowy Editor, Riverpod 3.0, Material 3.

---

### 7. Authentication and Authorization: JWT + Casbin

**Decision:** Use JWT for authentication and Casbin for role-based (and attribute-based) access control.

**Alternatives Considered:**

| Alternative          | Reason for Rejection                                                                   |
| -------------------- | -------------------------------------------------------------------------------------- |
| Session cookies      | Stateful; requires server-side session store; complicates horizontal scaling.           |
| OAuth2 only (Keycloak) | Heavy external dependency for MVP; can be layered on later as an identity provider.  |
| OPA (Open Policy Agent) | More powerful than needed for MVP; Rego policy language has a steeper learning curve. |
| Hand-rolled RBAC     | Error-prone; Casbin is battle-tested and saves significant implementation effort.       |

**Justification:**

- **JWT:** Stateless tokens that work across services and are easy to validate without a database round-trip. Standard `Authorization: Bearer` header.
- **Casbin:** Go-native policy engine that supports RBAC, ABAC, and custom models via a declarative configuration file. Policies can be stored in PostgreSQL, making them queryable and auditable.
- **Model:** `User -> Role -> Permission` mapped onto `Space` and `Page` resources. Example roles: Owner, Editor, Viewer. Casbin evaluates policies at the middleware layer before handlers execute.

**Risks:**

- JWT tokens cannot be revoked without a denylist (adds statefulness). Mitigation: short-lived access tokens (15 min) with refresh token rotation.
- Casbin policy debugging can be opaque; tooling for policy visualization is limited.

**Dependencies:** golang-jwt v5, Casbin v2, casbin-pgx-adapter.

---

### 8. Real-Time: WebSocket for Presence and Cursors

**Decision:** Implement real-time features via WebSocket connections for presence indicators and cursor positions. Defer full CRDT-based collaborative editing to Phase 2.

**Alternatives Considered:**

| Alternative     | Reason for Rejection                                                                     |
| --------------- | ---------------------------------------------------------------------------------------- |
| Server-Sent Events | Unidirectional; cannot carry cursor position updates from client to server.           |
| Full CRDT (Yjs) | Correct long-term solution, but adds significant complexity to MVP scope.               |
| Polling         | High latency, wasteful of resources.                                                    |

**Justification:**

- WebSocket provides a bidirectional, low-latency channel suitable for presence ("who is viewing this page") and cursor/selection broadcasting.
- **MVP conflict model:** Optimistic locking with version vectors. When two users edit the same page, the second save receives a conflict notification and can merge or overwrite.
- This is explicitly a stepping stone. The JSON AST has been designed to support OT/CRDT operations, so the migration path to Phase 2 is clear.

**Phase 2 plan:** Integrate Yjs or Automerge for conflict-free real-time co-editing. The WebSocket transport layer built in MVP will be reused.

**Risks:**

- Optimistic locking will cause data loss if users do not handle conflict prompts carefully.
- WebSocket connections require sticky sessions or a pub/sub backend (e.g., PostgreSQL LISTEN/NOTIFY or Redis) for multi-instance deployments.

**Dependencies:** gorilla/websocket or nhooyr/websocket.

---

### 9. API: REST with OpenAPI 3.0

**Decision:** Expose a RESTful API documented with an OpenAPI 3.0 specification, versioned under `/api/v1/`.

**Alternatives Considered:**

| Alternative | Reason for Rejection                                                                         |
| ----------- | -------------------------------------------------------------------------------------------- |
| GraphQL     | More flexible for clients, but adds resolver complexity, caching difficulty, and a larger learning curve for contributors. Better suited for Phase 2 if client query patterns demand it. |
| gRPC        | Excellent for service-to-service communication, but poor browser support without a proxy layer. |
| tRPC        | TypeScript-specific; not applicable to a Go + Dart stack.                                    |

**Justification:**

- REST is well-understood, broadly supported, and trivially cacheable.
- OpenAPI 3.0 specification is generated from code annotations (using Echo's built-in support or swaggo/swag), ensuring documentation stays in sync with implementation.
- Versioned endpoints (`/api/v1/spaces`, `/api/v1/pages/{id}`) allow non-breaking evolution.
- Standard HTTP semantics map cleanly to Knomantem's domain: `GET /pages/{id}`, `POST /pages`, `PATCH /pages/{id}`, `DELETE /pages/{id}`.

**Risks:**

- REST can lead to over-fetching or under-fetching for complex UIs (the classic GraphQL argument). Mitigation: use `?fields=` sparse fieldsets and `?include=` for related resources.
- API versioning strategy must be decided early; URL-based versioning (`/v1/`) is simple but can lead to version proliferation.

**Dependencies:** Echo v4 (routing + OpenAPI), oapi-codegen or swaggo/swag (spec generation).

---

### 10. Deployment: Single Binary + PostgreSQL

**Decision:** Ship Knomantem as a single compiled binary with PostgreSQL as the only external runtime dependency.

**Justification:**

- Go's static compilation produces one artifact. Bleve search is embedded. No JVM, no Node runtime, no sidecar services.
- The Docker image is approximately 50 MB (scratch or distroless base + Go binary + static assets).
- Installation: download the binary, point it at a PostgreSQL connection string, run it.
- Configuration via environment variables and/or a TOML config file.

**Future:**

- **SQLite mode:** For personal use or small teams, replace PostgreSQL with an embedded SQLite database. This would make Knomantem fully self-contained — zero external dependencies.
- **Horizontal scaling:** Add Redis or NATS for pub/sub (WebSocket fan-out), an external search engine, and a load balancer. The architecture should not assume single-instance, but it should not require multi-instance.

**Risks:**

- Single binary means the application, search index, and all background jobs share one process. A runaway search indexing job can starve HTTP handlers of CPU.
- No built-in HA story for MVP; PostgreSQL streaming replication is the recommended path for database redundancy.

**Dependencies:** Docker (optional), PostgreSQL 16+.

---

### 11. License: Business Source License 1.1 (BSL 1.1)

**Decision:** Release Knomantem under the Business Source License 1.1.

**Alternatives Considered:**

| Alternative    | Reason for Rejection                                                                              |
| -------------- | ------------------------------------------------------------------------------------------------- |
| MIT / Apache 2.0 | Fully permissive; allows competitors to offer a hosted version without contributing back.       |
| AGPL 3.0       | Strong copyleft deters enterprise adoption and commercial integrations.                           |
| SSPL            | MongoDB's license; not OSI-approved; legally contentious in some jurisdictions.                  |
| Proprietary    | Prevents community contributions and self-hosting, both of which are strategic goals.             |

**Justification:**

- **Free to use, self-host, and modify** for any purpose — internal corporate use, personal use, education, research.
- **Restriction:** Third parties cannot offer Knomantem as a commercial hosted/managed service (e.g., "Knomantem Cloud" run by a competitor).
- **Automatic conversion:** After 4 years, the code converts to Apache 2.0, becoming fully permissive.
- **Precedent:** BSL 1.1 is used by MariaDB, Sentry, CockroachDB, HashiCorp (Terraform, Vault), and others. It is well-understood by legal teams.
- **Monetization path:** The license protects the option to offer an official hosted cloud service as a commercial product while keeping the self-hosted version free.

**Risks:**

- BSL is not OSI-approved "open source." Some community members and organizations have policies against non-OSI licenses.
- The 4-year conversion window may be too long (or too short) depending on commercial traction.

**Dependencies:** None (legal decision).

---

## Technology Version Pinning

| Technology       | Minimum Version | Pinned/Tested Version | Notes                                    |
| ---------------- | --------------- | --------------------- | ---------------------------------------- |
| Go               | 1.23            | 1.23.x                | Required for improved generics and rangefunc |
| Echo             | 4.12            | 4.12.x                | Latest v4 stable                         |
| PostgreSQL       | 16.0            | 16.x                  | JSONB performance improvements           |
| pgx              | 5.6             | 5.6.x                 | Native PG driver                         |
| Bleve            | 2.4             | 2.4.x                 | Embedded search                          |
| Goldmark         | 1.7             | 1.7.x                 | CommonMark parser                        |
| Flutter          | 3.27            | 3.27.x                | Stable channel                           |
| AppFlowy Editor  | 3.x             | Latest stable          | KM-focused editor widget                 |
| Riverpod         | 3.0             | 3.0.x                 | State management                         |
| Casbin           | 2.x             | 2.x                   | RBAC/ABAC engine                         |
| golang-jwt       | 5.x             | 5.x                   | JWT implementation                       |
| golang-migrate   | 4.x             | 4.x                   | Database migrations                      |
| Docker base      | —               | gcr.io/distroless/static | Minimal container image               |

---

## Deployment Architecture Summary

```
┌─────────────────────────────────────────────────────┐
│                   Client Devices                     │
│  ┌─────────┐  ┌─────────┐  ┌──────┐  ┌───────────┐ │
│  │ Desktop │  │  Web    │  │ iOS  │  │  Android  │ │
│  │ Flutter │  │ Flutter │  │ Flut.│  │  Flutter  │ │
│  └────┬────┘  └────┬────┘  └──┬───┘  └─────┬─────┘ │
└───────┼────────────┼──────────┼─────────────┼───────┘
        │            │          │             │
        └────────────┴─────┬────┴─────────────┘
                           │
                    HTTPS + WSS
                           │
              ┌────────────┴────────────┐
              │    Knomantem Binary      │
              │  ┌────────────────────┐  │
              │  │   Echo v4 Router   │  │
              │  │  ┌──────────────┐  │  │
              │  │  │  REST API    │  │  │
              │  │  │  /api/v1/*   │  │  │
              │  │  └──────────────┘  │  │
              │  │  ┌──────────────┐  │  │
              │  │  │  WebSocket   │  │  │
              │  │  │  /ws         │  │  │
              │  │  └──────────────┘  │  │
              │  └────────────────────┘  │
              │                          │
              │  ┌────────────────────┐  │
              │  │   Casbin (RBAC)    │  │
              │  └────────────────────┘  │
              │                          │
              │  ┌────────────────────┐  │
              │  │   Bleve Search     │  │
              │  │   (embedded)       │  │
              │  └────────────────────┘  │
              │                          │
              │  ┌────────────────────┐  │
              │  │  Background Jobs   │  │
              │  │  - AI maintenance  │  │
              │  │  - Search indexing  │  │
              │  │  - Freshness check │  │
              │  └────────────────────┘  │
              └────────────┬────────────┘
                           │
                      TCP :5432
                           │
              ┌────────────┴────────────┐
              │    PostgreSQL 16         │
              │  ┌────────────────────┐  │
              │  │  Relational tables │  │
              │  │  - spaces          │  │
              │  │  - pages           │  │
              │  │  - users           │  │
              │  │  - roles           │  │
              │  │  - permissions     │  │
              │  └────────────────────┘  │
              │  ┌────────────────────┐  │
              │  │  JSONB columns     │  │
              │  │  - page content    │  │
              │  │  - graph edges     │  │
              │  │  - AI logs         │  │
              │  └────────────────────┘  │
              │  ┌────────────────────┐  │
              │  │  Full-text search  │  │
              │  │  (fallback)        │  │
              │  └────────────────────┘  │
              └─────────────────────────┘
```

**Minimal production deployment:**
- 1 server / VM / container running the Knomantem binary
- 1 PostgreSQL 16 instance (managed or self-hosted)
- Reverse proxy (Caddy/Nginx) for TLS termination (optional but recommended)

**Docker Compose (typical):**
- `knomantem` service: single binary, ~50 MB image
- `postgres` service: official PostgreSQL 16 image
- Volume mounts: PostgreSQL data, Bleve index directory, config file

---

*This document is a living record. Decisions may be revisited as the project matures, user feedback arrives, and scale requirements evolve. Each change should be recorded with a date and rationale.*
