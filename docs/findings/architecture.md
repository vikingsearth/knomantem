# Knomantem -- Technical Architecture

This document describes the architecture of Knomantem, a knowledge management system
designed for teams that need structured capture, retrieval, and maintenance of
organisational knowledge.

---

## 1. Tech Stack Summary

| Layer       | Technology                          | Version / Notes              |
|-------------|-------------------------------------|------------------------------|
| Backend     | Go + Echo                           | Echo v4                      |
| Database    | PostgreSQL                          | 16, driver: pgx v5          |
| Search      | Bleve (embedded full-text search)   | In-process, no external dep  |
| Frontend    | Flutter + AppFlowy Editor           | Riverpod 3.0 for state mgmt |
| Auth        | JWT tokens + Casbin RBAC            |                              |
| API         | REST, documented with OpenAPI 3.0   |                              |

---

## 2. System Architecture Diagram

```
 ┌──────────────────────────────────────────────────────────────────────────────┐
 │                              FLUTTER CLIENTS                                │
 │                                                                              │
 │   ┌──────────────┐     ┌──────────────┐     ┌──────────────┐               │
 │   │   Desktop     │     │     Web      │     │    Mobile    │               │
 │   │  (macOS/Win/  │     │   (WASM /    │     │  (iOS /      │               │
 │   │   Linux)      │     │   JS)        │     │   Android)   │               │
 │   └──────┬───────┘     └──────┬───────┘     └──────┬───────┘               │
 │          │                    │                     │                        │
 │          │  AppFlowy Editor   │  Riverpod 3.0       │                        │
 └──────────┼────────────────────┼─────────────────────┼────────────────────────┘
            │                    │                     │
            │    HTTPS / WSS     │    HTTPS / WSS      │   HTTPS / WSS
            ▼                    ▼                     ▼
 ┌──────────────────────────────────────────────────────────────────────────────┐
 │                          REVERSE PROXY (Caddy / Nginx)                      │
 │                                                                              │
 │   - TLS termination          - Rate limiting                                │
 │   - Static asset serving     - Request routing                              │
 │   - HTTP/2 + WebSocket       - Gzip / Brotli compression                   │
 └────────────────────────────────────┬─────────────────────────────────────────┘
                                      │
                      ┌───────────────┴───────────────┐
                      │         HTTP / WS              │
                      ▼                                ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │                        GO API SERVER (Echo v4)                         │
 │                                                                        │
 │  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  ┌────────────┐  │
 │  │  Handlers   │  │ Middleware  │  │  Services    │  │  Domain    │  │
 │  │  (REST +    │  │ (JWT auth,  │  │  (business   │  │  (entities │  │
 │  │   WebSocket)│  │  Casbin     │  │   logic,     │  │   + repo   │  │
 │  │             │  │  RBAC,      │  │   orchestr.) │  │   ifaces)  │  │
 │  │             │  │  logging,   │  │              │  │            │  │
 │  │             │  │  CORS)      │  │              │  │            │  │
 │  └──────┬──────┘  └─────────────┘  └──────┬───────┘  └────────────┘  │
 │         │                                  │                          │
 │         │            ┌─────────────────────┤                          │
 │         │            │                     │                          │
 │         ▼            ▼                     ▼                          │
 │  ┌─────────────────────────┐   ┌─────────────────────────┐           │
 │  │     Repository Layer    │   │     Repository Layer    │           │
 │  │    (PostgreSQL / pgx)   │   │       (Bleve)           │           │
 │  └────────────┬────────────┘   └────────────┬────────────┘           │
 └───────────────┼─────────────────────────────┼────────────────────────┘
                 │                              │
                 ▼                              ▼
 ┌───────────────────────────┐   ┌───────────────────────────┐
 │      PostgreSQL 16        │   │     Bleve Search Index    │
 │                           │   │                           │
 │  - Pages (JSONB content)  │   │  - Full-text search       │
 │  - Users & teams          │   │  - Stored on local FS     │
 │  - Knowledge graph edges  │   │  - Rebuilt from DB on     │
 │  - Tags & metadata        │   │    startup if missing     │
 │  - Freshness scores       │   │                           │
 │  - Audit log              │   │                           │
 └───────────────────────────┘   └───────────────────────────┘

 ┌──────────────────────────────────────────────────────────────────────────────┐
 │                          BACKGROUND WORKERS                                 │
 │                                                                              │
 │   ┌────────────────────┐  ┌────────────────────┐  ┌─────────────────────┐   │
 │   │  Freshness Checker │  │   Search Indexer   │  │    AI Service       │   │
 │   │                    │  │                    │  │                     │   │
 │   │  - Periodic scan   │  │  - Watches for DB  │  │  - Summarisation    │   │
 │   │  - Decay scoring   │  │    change events   │  │  - Tag suggestions  │   │
 │   │  - Notification    │  │  - Updates Bleve   │  │  - Relation         │   │
 │   │    triggers        │  │    index           │  │    extraction       │   │
 │   └────────────────────┘  └────────────────────┘  └─────────────────────┘   │
 │                                                                              │
 │   All workers run in-process as goroutines inside the same Go binary.       │
 └──────────────────────────────────────────────────────────────────────────────┘

 ┌──────────────────────────────────────────────────────────────────────────────┐
 │                        WEBSOCKET (Presence Only)                            │
 │                                                                              │
 │   Client ◄──── WSS ────► Echo WS Handler ──► Presence Hub (in-memory)      │
 │                                                                              │
 │   - Who is viewing a page       - Cursor position (optional)                │
 │   - Online/offline status       - Heartbeat-based cleanup                   │
 └──────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Clean Architecture Layers

Knomantem follows a clean architecture with strict dependency rules: outer layers
depend on inner layers, never the reverse. All cross-layer communication passes
through interfaces defined in the domain layer.

```
  ┌──────────────────────────────────────────────┐
  │              Handler Layer                   │  ← Outermost
  │   (HTTP routes, WebSocket, serialization)    │
  ├──────────────────────────────────────────────┤
  │              Service Layer                   │
  │   (Business logic, orchestration)            │
  ├──────────────────────────────────────────────┤
  │              Domain Layer                    │  ← Innermost
  │   (Entities, value objects, repo interfaces) │
  ├──────────────────────────────────────────────┤
  │             Repository Layer                 │  ← Outermost
  │   (PostgreSQL, Bleve, external APIs)         │
  └──────────────────────────────────────────────┘

  Dependency direction:  Handler → Service → Domain ← Repository
```

### 3.1 Handler Layer

Responsible for accepting HTTP and WebSocket requests, validating input, calling
the appropriate service method, and serializing the response.

- **Depends on:** Service layer (via interfaces).
- **Contains:** Request/response DTOs, input validation, route registration.
- **Example file:** `internal/handler/page_handler.go`

Handlers never contain business logic. A handler method typically:

1. Binds and validates the request body or path parameters.
2. Calls a single service method.
3. Returns the result as JSON with the appropriate HTTP status code.

### 3.2 Service Layer

Orchestrates business operations that may span multiple repositories or require
cross-cutting logic (e.g., creating a page AND indexing it for search).

- **Depends on:** Domain layer (entities + repository interfaces).
- **Contains:** Transaction boundaries, authorization checks (Casbin), event
  publication for background workers.
- **Example file:** `internal/service/page_service.go`

### 3.3 Domain Layer

The core of the application. Contains pure Go structs (entities and value objects)
and interface definitions for repositories. This layer has zero external
dependencies -- no database drivers, no HTTP frameworks.

- **Depends on:** Nothing.
- **Contains:** Entity structs (`Page`, `User`, `Edge`, `Tag`), repository
  interfaces (`PageRepository`, `UserRepository`, `SearchRepository`), domain
  errors, and business rule validation.
- **Example files:** `internal/domain/page.go`, `internal/domain/repository.go`

### 3.4 Repository Layer

Concrete implementations of the repository interfaces defined in the domain layer.
Each implementation is specific to a storage technology.

- **Depends on:** Domain interfaces.
- **Contains:** SQL queries (pgx), Bleve index operations, mapping between
  database rows and domain entities.
- **Example file:** `internal/repository/postgres/page_repository.go`

---

## 4. Component Interactions

### 4.1 Creating a Page

```
Client (Flutter)
  │
  │  POST /api/v1/pages  { title, content (JSON), tags }
  ▼
page_handler.go
  │  Bind + validate request body
  │  Extract user from JWT context
  ▼
page_service.go
  │  Begin transaction
  │  Call PageRepository.Create(page)
  │  Call EdgeRepository.CreateImplicit(page)   ← auto-link mentioned pages
  │  Call SearchRepository.Index(page)          ← index in Bleve
  │  Commit transaction
  ▼
page_repository.go (postgres)
  │  INSERT INTO pages (id, title, content, ...) VALUES (...)
  │  content stored as JSONB
  ▼
Response: 201 Created { id, title, created_at, ... }
```

### 4.2 Searching for Content

```
Client (Flutter)
  │
  │  GET /api/v1/search?q=deployment+strategy&limit=20
  ▼
search_handler.go
  │  Parse query string, validate parameters
  ▼
search_service.go
  │  Call SearchRepository.Search(query, filters)
  │  Enrich results with metadata from PageRepository
  │  Apply Casbin permission filtering (user can only see authorized pages)
  ▼
search_repository.go (bleve)
  │  Execute Bleve query with boosting (title > body > tags)
  │  Return scored document IDs
  ▼
page_repository.go (postgres)
  │  SELECT id, title, snippet, freshness_score FROM pages WHERE id IN (...)
  ▼
Response: 200 OK [ { id, title, snippet, score, freshness }, ... ]
```

### 4.3 Freshness Check Background Job

```
Scheduler (goroutine, runs every 6 hours)
  │
  ▼
freshness_service.go
  │  SELECT pages WHERE last_reviewed_at < NOW() - freshness_interval
  │  For each stale page:
  │    ├── Calculate decay score based on age, edit frequency, view count
  │    ├── UPDATE pages SET freshness_score = <new_score>
  │    └── If score < threshold → create notification for page owner
  ▼
notification_service.go
  │  INSERT INTO notifications (user_id, type, page_id, message)
  │  Push via WebSocket if user is online
  ▼
Done. Next run in 6 hours.
```

### 4.4 Knowledge Graph Edge Creation

```
Client (Flutter)
  │
  │  POST /api/v1/edges  { source_page_id, target_page_id, relation_type }
  ▼
edge_handler.go
  │  Validate both page IDs exist
  │  Validate relation_type is in allowed enum
  ▼
edge_service.go
  │  Check for duplicate edge
  │  Check for cycles (if relation is hierarchical)
  │  Call EdgeRepository.Create(edge)
  │  Update search index metadata for both pages
  ▼
edge_repository.go (postgres)
  │  INSERT INTO edges (source_id, target_id, relation, created_by)
  ▼
Response: 201 Created { id, source_id, target_id, relation }
```

### 4.5 User Authentication Flow

```
Client (Flutter)
  │
  │  POST /api/v1/auth/login  { email, password }
  ▼
auth_handler.go
  │  Bind + validate credentials
  ▼
auth_service.go
  │  Call UserRepository.FindByEmail(email)
  │  Verify password hash (bcrypt)
  │  Generate JWT access token  (short-lived, 15 min)
  │  Generate refresh token     (long-lived, 7 days, stored in DB)
  │  Load Casbin roles for user
  ▼
Response: 200 OK { access_token, refresh_token, expires_at }

───── Subsequent requests ─────

Client
  │
  │  GET /api/v1/pages  Authorization: Bearer <access_token>
  ▼
jwt_middleware.go
  │  Validate token signature and expiry
  │  Extract user ID and roles
  │  Attach to Echo context
  ▼
casbin_middleware.go
  │  Check (user_role, resource, action) against Casbin policy
  │  Allow or reject with 403
  ▼
Handler proceeds normally
```

---

## 5. Key Design Decisions

### 5.1 Why Clean Architecture

- **Testability.** Each layer can be tested in isolation. Services are tested with
  mock repositories. Handlers are tested with mock services. Domain logic needs no
  mocks at all.
- **Swappable implementations.** The repository layer can be replaced without
  touching business logic. If Bleve is ever outgrown, swapping in Meilisearch or
  Elasticsearch requires only a new repository implementation.
- **Onboarding clarity.** New developers can locate code predictably: HTTP concerns
  live in handlers, business rules live in services, data access lives in
  repositories.

### 5.2 Why Embedded Search (Bleve) vs External (Elasticsearch)

- **Deployment simplicity.** Bleve runs in-process. There is no separate search
  cluster to provision, monitor, or version-match. The entire system is a single
  binary plus PostgreSQL.
- **Sufficient scale.** For the target use case (teams of 5--500 with thousands to
  low hundreds of thousands of documents), Bleve performs well. If the system
  eventually needs sharded search, migrating to an external engine is straightforward
  because search is behind a repository interface.
- **Lower operational cost.** No JVM tuning, no heap sizing, no split-brain
  concerns.

### 5.3 Why REST vs GraphQL

- **Simpler MVP.** REST endpoints are straightforward to implement, document
  (OpenAPI 3.0), and cache. The knowledge management domain does not have the deep
  nesting that makes GraphQL compelling.
- **Better HTTP caching.** GET requests for pages and search results are trivially
  cacheable at the reverse proxy layer. GraphQL POST requests are not.
- **Ecosystem maturity.** Echo v4 has battle-tested middleware for REST. OpenAPI
  code generation produces Go server stubs and Flutter client code.

### 5.4 Why JSONB for Document Content

- **Flexible schema.** Page content from AppFlowy Editor is a tree of blocks
  (headings, paragraphs, lists, code blocks, etc.). Storing this as JSONB avoids an
  EAV anti-pattern and preserves the full document structure.
- **PostgreSQL native.** JSONB supports indexing (`GIN`), partial updates
  (`jsonb_set`), and path queries (`@>`, `?`, `#>>`), so the database can
  participate in content queries without full deserialization.
- **Migration friendly.** When the editor schema evolves, old documents remain
  readable. Schema migrations happen at the application layer, not the database
  layer.

### 5.5 Why Background Workers for Freshness

- **Don't block user requests.** Freshness scoring involves scanning potentially
  thousands of pages and computing decay functions. Running this synchronously in a
  request handler would cause unacceptable latency.
- **Periodic is sufficient.** Knowledge freshness changes slowly (hours to days, not
  seconds). A background job every 6 hours is more than adequate.
- **Goroutines keep it simple.** Workers run as goroutines in the same process.
  There is no need for a separate job queue (Redis, RabbitMQ) at this scale.

### 5.6 Why WebSocket for Presence Only

- **Full CRDT is too complex for MVP.** Real-time collaborative editing with
  conflict-free replicated data types is a massive engineering undertaking
  (see: Yjs, Automerge). The MVP uses optimistic locking for concurrent edits
  instead.
- **Presence is high value, low cost.** Showing "Alice is viewing this page" or
  "Bob is editing" provides significant UX value with minimal server-side state
  (an in-memory map of page ID to connected user IDs).
- **Upgrade path.** The WebSocket infrastructure established for presence can later
  carry CRDT operations when collaborative editing is prioritized.

---

## 6. Directory Structure

```
knomantem/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point: config, DI, server start
│
├── internal/
│   ├── domain/
│   │   ├── page.go                  # Page entity, PageContent value object
│   │   ├── user.go                  # User entity, credentials
│   │   ├── edge.go                  # Knowledge graph edge entity
│   │   ├── tag.go                   # Tag entity
│   │   ├── notification.go          # Notification entity
│   │   ├── repository.go            # All repository interfaces
│   │   └── errors.go                # Domain-specific error types
│   │
│   ├── handler/
│   │   ├── page_handler.go          # CRUD for pages
│   │   ├── search_handler.go        # Search endpoints
│   │   ├── edge_handler.go          # Knowledge graph edge endpoints
│   │   ├── auth_handler.go          # Login, register, refresh
│   │   ├── user_handler.go          # User profile, preferences
│   │   ├── notification_handler.go  # Notification list, mark-read
│   │   ├── presence_handler.go      # WebSocket upgrade + presence
│   │   └── routes.go                # Route registration
│   │
│   ├── service/
│   │   ├── page_service.go          # Page business logic
│   │   ├── search_service.go        # Search orchestration
│   │   ├── edge_service.go          # Graph operations, cycle detection
│   │   ├── auth_service.go          # JWT issuance, password verification
│   │   ├── freshness_service.go     # Decay scoring, staleness detection
│   │   ├── notification_service.go  # Notification creation + delivery
│   │   └── presence_service.go      # In-memory presence hub
│   │
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── page_repository.go   # PageRepository (pgx)
│   │   │   ├── user_repository.go   # UserRepository (pgx)
│   │   │   ├── edge_repository.go   # EdgeRepository (pgx)
│   │   │   └── notification_repository.go
│   │   └── bleve/
│   │       └── search_repository.go # SearchRepository (Bleve)
│   │
│   ├── middleware/
│   │   ├── jwt.go                   # JWT validation middleware
│   │   ├── casbin.go                # Casbin RBAC enforcement
│   │   ├── logging.go               # Structured request logging
│   │   ├── cors.go                  # CORS configuration
│   │   └── ratelimit.go             # Per-user rate limiting
│   │
│   ├── config/
│   │   └── config.go                # Env-based configuration struct
│   │
│   └── worker/
│       ├── freshness_checker.go     # Periodic freshness background job
│       ├── search_indexer.go        # Watches DB changes, updates Bleve
│       └── ai_service.go           # Summarisation, tag suggestion
│
├── pkg/
│   ├── search/
│   │   └── bleve.go                 # Bleve index initialization + helpers
│   └── markdown/
│       └── render.go                # Markdown-to-plaintext for indexing
│
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_pages.sql
│   ├── 003_create_edges.sql
│   ├── 004_create_tags.sql
│   ├── 005_create_notifications.sql
│   └── 006_create_audit_log.sql
│
├── web/                             # Flutter application
│   ├── lib/
│   │   ├── main.dart
│   │   ├── features/
│   │   │   ├── pages/               # Page viewing and editing
│   │   │   ├── search/              # Search UI
│   │   │   ├── graph/               # Knowledge graph visualisation
│   │   │   └── auth/                # Login, registration
│   │   ├── providers/               # Riverpod 3.0 providers
│   │   ├── models/                  # Dart data classes
│   │   └── services/                # API client, WebSocket client
│   ├── pubspec.yaml
│   └── test/
│
├── docs/
│   ├── openapi.yaml                 # OpenAPI 3.0 specification
│   └── findings/
│       └── architecture.md          # This document
│
├── docker-compose.yml               # Dev environment (Go + PG + Caddy)
├── Dockerfile                       # Multi-stage build for Go binary
├── Makefile                         # Build, test, migrate, lint targets
├── go.mod
└── go.sum
```

---

## 7. Infrastructure and Deployment

### 7.1 Single Binary Deployment

The Go server compiles to a single statically linked binary. Bleve runs embedded,
so there is no search service to deploy separately. The only external runtime
dependency is PostgreSQL.

```
Production deployment requires exactly two processes:
  1. knomantem-server (Go binary)
  2. PostgreSQL 16
```

### 7.2 Docker Image

A multi-stage Docker build produces a minimal image.

```dockerfile
# Build stage
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /knomantem ./cmd/server

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /knomantem /usr/local/bin/knomantem
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["knomantem"]
```

Resulting image size: approximately 50 MB (Go binary ~30 MB + Alpine base ~20 MB).

### 7.3 Docker Compose (Development)

```yaml
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://knomantem:secret@db:5432/knomantem?sslmode=disable
      JWT_SECRET: dev-secret-change-in-production
      BLEVE_INDEX_PATH: /data/search.bleve
      LOG_LEVEL: debug
    volumes:
      - bleve_data:/data
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: knomantem
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: knomantem
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U knomantem"]
      interval: 5s
      timeout: 3s
      retries: 5

  caddy:
    image: caddy:2-alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data

volumes:
  pg_data:
  bleve_data:
  caddy_data:
```

### 7.4 Configuration via Environment Variables

All configuration is read from environment variables, following twelve-factor app
principles.

| Variable            | Description                       | Default               |
|---------------------|-----------------------------------|-----------------------|
| `DATABASE_URL`      | PostgreSQL connection string      | (required)            |
| `JWT_SECRET`        | HMAC secret for JWT signing       | (required)            |
| `JWT_EXPIRY`        | Access token lifetime             | `15m`                 |
| `REFRESH_EXPIRY`    | Refresh token lifetime            | `168h` (7 days)       |
| `BLEVE_INDEX_PATH`  | Filesystem path for Bleve index   | `./data/search.bleve` |
| `PORT`              | HTTP listen port                  | `8080`                |
| `LOG_LEVEL`         | Logging verbosity                 | `info`                |
| `CORS_ORIGINS`      | Comma-separated allowed origins   | `*`                   |
| `FRESHNESS_INTERVAL`| How often the freshness job runs  | `6h`                  |

### 7.5 Bleve Index Management

The Bleve search index is stored as a directory on the local filesystem. It is
treated as a derived data store: if the index directory is missing or corrupted on
startup, the server rebuilds it from PostgreSQL.

- **Backup:** Not required. The index is fully rebuildable from the database.
- **Persistence:** In Docker, mount a named volume at `BLEVE_INDEX_PATH` to avoid
  rebuilding on every container restart.
- **Rebuild trigger:** Delete the index directory and restart the server, or call
  `POST /api/v1/admin/reindex` (admin-only endpoint).

---

## 8. Cross-Cutting Concerns

### 8.1 Logging

Structured JSON logging via `slog` (Go standard library). Every request logs:
method, path, status, latency, user ID, and request ID.

### 8.2 Error Handling

Domain errors are defined in `internal/domain/errors.go` and mapped to HTTP status
codes in the handler layer. Services return domain errors; handlers translate them.

| Domain Error         | HTTP Status |
|----------------------|-------------|
| `ErrNotFound`        | 404         |
| `ErrConflict`        | 409         |
| `ErrValidation`      | 422         |
| `ErrUnauthorized`    | 401         |
| `ErrForbidden`       | 403         |

### 8.3 Testing Strategy

- **Domain layer:** Pure unit tests, no mocks needed.
- **Service layer:** Unit tests with mock repositories (generated via `mockgen`).
- **Handler layer:** Integration tests using `httptest` with mock services.
- **Repository layer:** Integration tests against a test PostgreSQL instance
  (managed via `testcontainers-go`).
- **End-to-end:** A small suite of E2E tests using Docker Compose to spin up the
  full stack.

---

*This document reflects the architecture as of initial design. It will be updated as
the system evolves.*
