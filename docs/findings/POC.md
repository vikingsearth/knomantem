# Knomantem -- Proof of Concept Specification

**Version:** 1.0
**Date:** 2026-03-18
**Status:** Draft
**License:** Business Source License 1.1 (BSL 1.1)

---

## Table of Contents

1. [POC Objectives](#1-poc-objectives)
2. [Monorepo Directory Structure](#2-monorepo-directory-structure)
3. [PostgreSQL Schema](#3-postgresql-schema)
4. [REST API Endpoints](#4-rest-api-endpoints)
5. [UI Wireframes](#5-ui-wireframes)
6. [Success Criteria](#6-success-criteria)

---

## 1. POC Objectives

The Knomantem POC validates five core capabilities before committing to a full production build. Each objective has concrete, measurable acceptance criteria defined in Section 6.

### 1.1 Editor Capability

Prove that the AppFlowy Editor embedded in a Flutter application provides a viable rich-text editing experience with a custom JSON AST document model. The editor must support headings (H1-H3), bold, italic, strikethrough, inline code, code blocks, bullet lists, numbered lists, checklists, block quotes, dividers, and internal page links. Additionally, demonstrate bi-directional Markdown conversion: import Markdown files into the JSON AST via Goldmark on the backend, and export the JSON AST back to well-formed Markdown with minimal formatting loss.

### 1.2 Page Tree Navigation

Prove that a Spaces-to-nested-pages hierarchy can be rendered as a collapsible tree in the left sidebar, with drag-and-drop reordering and reparenting. The tree must support at least four levels of depth, lazy-load children on expansion, and persist position changes to the backend with optimistic UI updates.

### 1.3 Search

Prove that Bleve full-text search can index page content, titles, and tags, and return faceted search results in under 200 ms for a dataset of 1,000 pages. Faceted filtering must support filtering by space, tags, freshness status, and date range. Results must include highlighted match excerpts.

### 1.4 Freshness System

Prove that an automated freshness scoring system works end-to-end: pages receive an initial freshness score upon creation, the score decays over time via a configurable decay rate calculated by a background worker, users can manually verify a page (resetting its score to 100), and visual indicators (green/yellow/red badges) accurately reflect the current freshness state throughout the UI.

### 1.5 Architecture Soundness

Prove that Clean Architecture in Go with Echo v4 provides clean separation of concerns (handler -> service -> repository), that all business logic in the service layer is testable without a database dependency (via repository interfaces), and that API response times remain under 50 ms for non-search endpoints under normal load.

---

## 2. Monorepo Directory Structure

```
knomantem/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point: parse config, wire dependencies, start Echo server
│
├── internal/
│   ├── config/
│   │   └── config.go                  # Env-based config (reads .env, validates required vars)
│   │
│   ├── domain/
│   │   ├── page.go                    # Page entity + PageTree helper types
│   │   ├── space.go                   # Space entity
│   │   ├── user.go                    # User entity + Role enum
│   │   ├── freshness.go               # FreshnessRecord entity + FreshnessStatus enum
│   │   ├── graph_edge.go              # GraphEdge entity + EdgeType enum
│   │   ├── tag.go                     # Tag entity + PageTag junction type
│   │   ├── errors.go                  # Sentinel domain errors (ErrNotFound, ErrConflict, etc.)
│   │   └── repository.go             # All repository interfaces (one per aggregate root)
│   │
│   ├── handler/
│   │   ├── auth_handler.go            # POST login/register/refresh, GET me
│   │   ├── space_handler.go           # CRUD spaces
│   │   ├── page_handler.go            # CRUD pages, tree ops, move, import/export
│   │   ├── search_handler.go          # GET search with query params
│   │   ├── freshness_handler.go       # GET freshness, POST verify, PUT settings, GET dashboard
│   │   ├── graph_handler.go           # GET neighbors, POST edges, GET explore
│   │   ├── tag_handler.go             # GET/POST tags, POST page tags
│   │   └── middleware.go              # JWT auth middleware, request logging, CORS, rate limiting
│   │
│   ├── service/
│   │   ├── auth_service.go            # JWT generation/validation, password hashing (bcrypt)
│   │   ├── space_service.go           # Space CRUD business logic, slug generation
│   │   ├── page_service.go            # Page CRUD, tree operations, versioning on content change
│   │   ├── search_service.go          # Bleve index management, query building, facet extraction
│   │   ├── freshness_service.go       # Score calculation, decay logic, verification, dashboard aggregation
│   │   ├── graph_service.go           # Edge CRUD, neighbor queries, subgraph traversal
│   │   ├── tag_service.go             # Tag CRUD, page-tag association, bulk tagging
│   │   └── markdown_service.go        # Goldmark-based Markdown-to-JSON-AST and JSON-AST-to-Markdown
│   │
│   ├── repository/
│   │   └── postgres/
│   │       ├── space_repo.go          # SpaceRepository implementation (pgx v5)
│   │       ├── page_repo.go           # PageRepository implementation with tree queries
│   │       ├── user_repo.go           # UserRepository implementation
│   │       ├── freshness_repo.go      # FreshnessRepository implementation
│   │       ├── graph_repo.go          # GraphRepository implementation
│   │       ├── tag_repo.go            # TagRepository implementation
│   │       └── version_repo.go        # VersionRepository implementation
│   │
│   └── worker/
│       ├── freshness_worker.go        # Periodic freshness recalculation (runs every hour)
│       └── search_indexer.go          # Listens for page changes, updates Bleve index
│
├── pkg/
│   ├── search/
│   │   └── bleve.go                   # Bleve wrapper: index lifecycle, mapping config, query helpers
│   └── markdown/
│       ├── importer.go                # Markdown string -> JSON AST (AppFlowy document format)
│       └── exporter.go                # JSON AST -> Markdown string
│
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_spaces.sql
│   ├── 003_create_pages.sql
│   ├── 004_create_page_versions.sql
│   ├── 005_create_freshness.sql
│   ├── 006_create_graph_edges.sql
│   ├── 007_create_tags.sql
│   └── 008_create_permissions.sql
│
├── web/                                # Flutter application
│   ├── lib/
│   │   ├── main.dart                  # Bootstrap app, initialize providers
│   │   ├── app.dart                   # MaterialApp with theme, router
│   │   ├── router.dart                # GoRouter: route definitions, guards
│   │   │
│   │   ├── providers/
│   │   │   ├── auth_provider.dart     # Auth state, token storage, login/logout
│   │   │   ├── space_provider.dart    # Spaces list, selected space
│   │   │   ├── page_provider.dart     # Page tree, selected page, CRUD
│   │   │   ├── search_provider.dart   # Search query, results, filters
│   │   │   └── freshness_provider.dart # Freshness data, verify action
│   │   │
│   │   ├── models/
│   │   │   ├── space.dart             # Space data class + JSON serialization
│   │   │   ├── page.dart              # Page data class + JSON serialization
│   │   │   ├── user.dart              # User data class + JSON serialization
│   │   │   └── search_result.dart     # SearchResult data class + JSON serialization
│   │   │
│   │   ├── services/
│   │   │   └── api_service.dart       # HTTP client (dio), interceptors, base URL config
│   │   │
│   │   ├── screens/
│   │   │   ├── login_screen.dart      # Email/password login form
│   │   │   ├── home_screen.dart       # Dashboard: recent pages, freshness alerts, activity
│   │   │   ├── space_screen.dart      # Space view: page tree + content area
│   │   │   ├── page_editor_screen.dart # AppFlowy Editor + metadata sidebar
│   │   │   ├── search_screen.dart     # Search bar + filter chips + result cards
│   │   │   └── graph_screen.dart      # Interactive graph visualization
│   │   │
│   │   └── widgets/
│   │       ├── page_tree.dart         # Collapsible tree with drag-and-drop
│   │       ├── freshness_badge.dart   # Color-coded freshness indicator
│   │       ├── search_bar.dart        # Expandable search with suggestions
│   │       ├── graph_view.dart        # Force-directed graph canvas
│   │       └── editor_toolbar.dart    # Floating formatting toolbar
│   │
│   ├── pubspec.yaml                   # Flutter dependencies
│   └── test/                          # Widget and integration tests
│
├── docker-compose.yml                 # PostgreSQL 16, Go API, Flutter web (dev)
├── Dockerfile                         # Multi-stage: build Go binary, serve
├── Makefile                           # dev, build, test, migrate, seed targets
├── go.mod                             # Go module definition
├── go.sum                             # Go dependency checksums
├── .env.example                       # Template environment variables
├── LICENSE                            # BSL 1.1 license text
└── docs/                              # Documentation
```

---

## 3. PostgreSQL Schema

All tables use UUIDs as primary keys generated via `gen_random_uuid()`. Timestamps use `TIMESTAMPTZ` and default to `NOW()`. Foreign keys cascade on delete where appropriate.

### 3.1 Users

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url    TEXT,
    role          VARCHAR(50)  NOT NULL DEFAULT 'member'
                  CHECK (role IN ('admin', 'member', 'viewer')),
    settings      JSONB        NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
```

### 3.2 Spaces

```sql
CREATE TABLE spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    icon        VARCHAR(50),
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    settings    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_spaces_owner ON spaces (owner_id);
CREATE INDEX idx_spaces_slug  ON spaces (slug);
```

### 3.3 Pages

```sql
CREATE TABLE pages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID         NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id   UUID         REFERENCES pages(id) ON DELETE SET NULL,
    title       VARCHAR(500) NOT NULL,
    slug        VARCHAR(500) NOT NULL,
    content     JSONB        NOT NULL DEFAULT '{"type":"doc","content":[]}',
    position    INTEGER      NOT NULL DEFAULT 0,
    depth       INTEGER      NOT NULL DEFAULT 0,
    icon        VARCHAR(50),
    cover_image TEXT,
    is_template BOOLEAN      NOT NULL DEFAULT FALSE,
    created_by  UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by  UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_pages_space_slug UNIQUE (space_id, slug)
);

CREATE INDEX idx_pages_space    ON pages (space_id);
CREATE INDEX idx_pages_parent   ON pages (parent_id);
CREATE INDEX idx_pages_position ON pages (space_id, parent_id, position);
CREATE INDEX idx_pages_created  ON pages (created_by);
```

### 3.4 Page Versions

```sql
CREATE TABLE page_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id         UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    version         INTEGER      NOT NULL,
    title           VARCHAR(500) NOT NULL,
    content         JSONB        NOT NULL,
    change_summary  TEXT,
    created_by      UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_page_versions UNIQUE (page_id, version)
);

CREATE INDEX idx_page_versions_page ON page_versions (page_id, version DESC);
```

### 3.5 Freshness Records

```sql
CREATE TABLE freshness_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id             UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    owner_id            UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    freshness_score     DECIMAL(5,2) NOT NULL DEFAULT 100.00
                        CHECK (freshness_score >= 0 AND freshness_score <= 100),
    review_interval_days INTEGER     NOT NULL DEFAULT 30,
    last_reviewed_at    TIMESTAMPTZ,
    next_review_at      TIMESTAMPTZ,
    last_verified_by    UUID         REFERENCES users(id) ON DELETE SET NULL,
    last_verified_at    TIMESTAMPTZ,
    status              VARCHAR(20)  NOT NULL DEFAULT 'fresh'
                        CHECK (status IN ('fresh', 'aging', 'stale', 'unverified')),
    decay_rate          DECIMAL(5,4) NOT NULL DEFAULT 0.0333,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_freshness_page UNIQUE (page_id)
);

CREATE INDEX idx_freshness_status    ON freshness_records (status);
CREATE INDEX idx_freshness_next_review ON freshness_records (next_review_at);
CREATE INDEX idx_freshness_owner     ON freshness_records (owner_id);
```

**Freshness decay formula:**
```
new_score = current_score * (1 - decay_rate) ^ days_elapsed
```
Where `decay_rate` defaults to `0.0333` (score reaches ~50 after 20 days with no review, and ~0 after ~60 days). Status thresholds:
- `fresh`: score >= 70
- `aging`: score >= 40 AND score < 70
- `stale`: score < 40
- `unverified`: page has never been verified

### 3.6 Graph Edges

```sql
CREATE TABLE graph_edges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    target_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    edge_type       VARCHAR(50) NOT NULL DEFAULT 'reference'
                    CHECK (edge_type IN ('reference', 'parent', 'related', 'depends_on', 'derived_from')),
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_by      UUID        NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_graph_edge UNIQUE (source_page_id, target_page_id, edge_type),
    CONSTRAINT chk_no_self_link CHECK (source_page_id != target_page_id)
);

CREATE INDEX idx_graph_source ON graph_edges (source_page_id);
CREATE INDEX idx_graph_target ON graph_edges (target_page_id);
CREATE INDEX idx_graph_type   ON graph_edges (edge_type);
```

### 3.7 Tags and Page Tags

```sql
CREATE TABLE tags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    color           VARCHAR(7)   NOT NULL DEFAULT '#6B7280',
    is_ai_generated BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_name ON tags (name);

CREATE TABLE page_tags (
    page_id          UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id           UUID         NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    confidence_score DECIMAL(3,2) NOT NULL DEFAULT 1.00
                     CHECK (confidence_score >= 0 AND confidence_score <= 1),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    PRIMARY KEY (page_id, tag_id)
);

CREATE INDEX idx_page_tags_tag ON page_tags (tag_id);
```

### 3.8 Permissions

```sql
CREATE TABLE permissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type    VARCHAR(50) NOT NULL
                     CHECK (resource_type IN ('space', 'page')),
    resource_id      UUID        NOT NULL,
    permission_level VARCHAR(50) NOT NULL
                     CHECK (permission_level IN ('owner', 'editor', 'commenter', 'viewer')),
    granted_by       UUID        NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permission UNIQUE (user_id, resource_type, resource_id)
);

CREATE INDEX idx_permissions_user     ON permissions (user_id);
CREATE INDEX idx_permissions_resource ON permissions (resource_type, resource_id);
```

---

## 4. REST API Endpoints

All endpoints are prefixed with `/api/v1`. Authenticated endpoints require an `Authorization: Bearer <token>` header. All request and response bodies use `application/json`. Standard error responses follow the shape:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Page not found"
  }
}
```

Pagination follows cursor-based pagination where applicable:

```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "uuid-value",
    "has_more": true,
    "total": 42
  }
}
```

---

### 4.1 Auth

#### POST /api/v1/auth/register

Create a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "display_name": "Jane Doe",
  "password": "secureP@ss123"
}
```

**Response (201 Created):**
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "display_name": "Jane Doe",
      "avatar_url": null,
      "role": "member",
      "created_at": "2026-03-18T10:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
  }
}
```

#### POST /api/v1/auth/login

Authenticate with email and password.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "secureP@ss123"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "display_name": "Jane Doe",
      "avatar_url": null,
      "role": "member",
      "created_at": "2026-03-18T10:00:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
  }
}
```

#### POST /api/v1/auth/refresh

Refresh an expired access token.

**Request Body:**
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g..."
}
```

**Response (200 OK):**
```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4..."
  }
}
```

#### GET /api/v1/auth/me

Return the authenticated user's profile. Requires authentication.

**Response (200 OK):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "display_name": "Jane Doe",
    "avatar_url": null,
    "role": "member",
    "settings": {},
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T10:00:00Z"
  }
}
```

---

### 4.2 Spaces

All space endpoints require authentication.

#### GET /api/v1/spaces

List all spaces the authenticated user has access to.

**Query Parameters:**
| Param  | Type   | Default | Description          |
|--------|--------|---------|----------------------|
| cursor | UUID   | -       | Pagination cursor    |
| limit  | int    | 20      | Items per page (max 100) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "Engineering",
      "slug": "engineering",
      "description": "Engineering team knowledge base",
      "icon": "rocket",
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "page_count": 42,
      "created_at": "2026-03-18T10:00:00Z",
      "updated_at": "2026-03-18T10:00:00Z"
    }
  ],
  "pagination": {
    "next_cursor": null,
    "has_more": false,
    "total": 1
  }
}
```

#### POST /api/v1/spaces

Create a new space. The authenticated user becomes the owner.

**Request Body:**
```json
{
  "name": "Engineering",
  "description": "Engineering team knowledge base",
  "icon": "rocket",
  "settings": {}
}
```

**Response (201 Created):**
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Engineering",
    "slug": "engineering",
    "description": "Engineering team knowledge base",
    "icon": "rocket",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "settings": {},
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T10:00:00Z"
  }
}
```

#### GET /api/v1/spaces/:id

Get a single space by ID.

**Response (200 OK):**
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Engineering",
    "slug": "engineering",
    "description": "Engineering team knowledge base",
    "icon": "rocket",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "settings": {},
    "page_count": 42,
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T10:00:00Z"
  }
}
```

#### PUT /api/v1/spaces/:id

Update a space. Requires owner or admin role.

**Request Body:**
```json
{
  "name": "Engineering (updated)",
  "description": "Updated description",
  "icon": "gear",
  "settings": { "default_freshness_interval": 14 }
}
```

**Response (200 OK):**
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "name": "Engineering (updated)",
    "slug": "engineering",
    "description": "Updated description",
    "icon": "gear",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "settings": { "default_freshness_interval": 14 },
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T11:00:00Z"
  }
}
```

#### DELETE /api/v1/spaces/:id

Delete a space and all its pages. Requires owner or admin role.

**Response (204 No Content)**

---

### 4.3 Pages

All page endpoints require authentication.

#### GET /api/v1/spaces/:spaceId/pages

Get the full page tree for a space. Returns a flat list with `parent_id` and `depth` fields that the client assembles into a tree, or optionally a nested tree structure.

**Query Parameters:**
| Param  | Type   | Default | Description               |
|--------|--------|---------|---------------------------|
| format | string | "flat"  | "flat" or "tree"          |
| depth  | int    | -1      | Max depth (-1 = unlimited)|

**Response (200 OK, format=flat):**
```json
{
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "space_id": "660e8400-e29b-41d4-a716-446655440000",
      "parent_id": null,
      "title": "Getting Started",
      "slug": "getting-started",
      "icon": "book",
      "position": 0,
      "depth": 0,
      "is_template": false,
      "has_children": true,
      "freshness_status": "fresh",
      "created_at": "2026-03-18T10:00:00Z",
      "updated_at": "2026-03-18T10:00:00Z"
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "space_id": "660e8400-e29b-41d4-a716-446655440000",
      "parent_id": "770e8400-e29b-41d4-a716-446655440000",
      "title": "Installation",
      "slug": "installation",
      "icon": null,
      "position": 0,
      "depth": 1,
      "is_template": false,
      "has_children": false,
      "freshness_status": "aging",
      "created_at": "2026-03-18T10:00:00Z",
      "updated_at": "2026-03-18T10:00:00Z"
    }
  ]
}
```

#### POST /api/v1/spaces/:spaceId/pages

Create a new page within a space.

**Request Body:**
```json
{
  "title": "New Page",
  "parent_id": "770e8400-e29b-41d4-a716-446655440000",
  "content": {
    "type": "doc",
    "content": [
      {
        "type": "heading",
        "attrs": { "level": 1 },
        "content": [{ "type": "text", "text": "New Page" }]
      }
    ]
  },
  "icon": "page",
  "position": 0,
  "is_template": false
}
```

**Response (201 Created):**
```json
{
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440000",
    "space_id": "660e8400-e29b-41d4-a716-446655440000",
    "parent_id": "770e8400-e29b-41d4-a716-446655440000",
    "title": "New Page",
    "slug": "new-page",
    "content": { "type": "doc", "content": [...] },
    "icon": "page",
    "position": 0,
    "depth": 1,
    "is_template": false,
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "updated_by": "550e8400-e29b-41d4-a716-446655440000",
    "freshness": {
      "score": 100.00,
      "status": "fresh"
    },
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T10:00:00Z"
  }
}
```

#### GET /api/v1/pages/:id

Get a single page with full content.

**Response (200 OK):**
```json
{
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "space_id": "660e8400-e29b-41d4-a716-446655440000",
    "parent_id": null,
    "title": "Getting Started",
    "slug": "getting-started",
    "content": {
      "type": "doc",
      "content": [
        {
          "type": "heading",
          "attrs": { "level": 1 },
          "content": [{ "type": "text", "text": "Getting Started" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "Welcome to Knomantem." }]
        }
      ]
    },
    "icon": "book",
    "cover_image": null,
    "position": 0,
    "depth": 0,
    "is_template": false,
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "updated_by": "550e8400-e29b-41d4-a716-446655440000",
    "freshness": {
      "score": 85.50,
      "status": "fresh",
      "last_verified_at": "2026-03-15T10:00:00Z",
      "next_review_at": "2026-04-14T10:00:00Z"
    },
    "tags": [
      { "id": "aa0e8400-...", "name": "onboarding", "color": "#10B981" }
    ],
    "version": 3,
    "created_at": "2026-03-18T10:00:00Z",
    "updated_at": "2026-03-18T10:00:00Z"
  }
}
```

#### PUT /api/v1/pages/:id

Update a page. Creates a new version automatically if content changes.

**Request Body:**
```json
{
  "title": "Getting Started (Revised)",
  "content": {
    "type": "doc",
    "content": [...]
  },
  "icon": "book",
  "cover_image": "https://example.com/cover.jpg",
  "change_summary": "Added installation instructions"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "title": "Getting Started (Revised)",
    "slug": "getting-started",
    "content": { "type": "doc", "content": [...] },
    "version": 4,
    "updated_at": "2026-03-18T11:00:00Z"
  }
}
```

#### DELETE /api/v1/pages/:id

Delete a page. Child pages are reparented to the deleted page's parent.

**Response (204 No Content)**

#### PUT /api/v1/pages/:id/move

Move a page to a new parent and/or position within the tree.

**Request Body:**
```json
{
  "parent_id": "770e8400-e29b-41d4-a716-446655440000",
  "position": 2
}
```

`parent_id` can be `null` to move to root level. `position` is the zero-based index among siblings.

**Response (200 OK):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "parent_id": "770e8400-e29b-41d4-a716-446655440000",
    "position": 2,
    "depth": 1,
    "updated_at": "2026-03-18T11:00:00Z"
  }
}
```

#### GET /api/v1/pages/:id/versions

List all versions of a page, ordered by version number descending.

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "bb0e8400-...",
      "page_id": "770e8400-...",
      "version": 4,
      "title": "Getting Started (Revised)",
      "change_summary": "Added installation instructions",
      "created_by": {
        "id": "550e8400-...",
        "display_name": "Jane Doe"
      },
      "created_at": "2026-03-18T11:00:00Z"
    },
    {
      "id": "cc0e8400-...",
      "page_id": "770e8400-...",
      "version": 3,
      "title": "Getting Started",
      "change_summary": "Fixed typos",
      "created_by": {
        "id": "550e8400-...",
        "display_name": "Jane Doe"
      },
      "created_at": "2026-03-17T09:00:00Z"
    }
  ]
}
```

#### GET /api/v1/pages/:id/versions/:version

Get the full content of a specific page version.

**Response (200 OK):**
```json
{
  "data": {
    "id": "cc0e8400-...",
    "page_id": "770e8400-...",
    "version": 3,
    "title": "Getting Started",
    "content": { "type": "doc", "content": [...] },
    "change_summary": "Fixed typos",
    "created_by": {
      "id": "550e8400-...",
      "display_name": "Jane Doe"
    },
    "created_at": "2026-03-17T09:00:00Z"
  }
}
```

#### POST /api/v1/pages/:id/import/markdown

Import a Markdown file and replace the page content with the parsed result. The Markdown is converted to the JSON AST format via Goldmark on the backend.

**Request Body:**
```json
{
  "markdown": "# Heading\n\nThis is a paragraph with **bold** and *italic* text.\n\n- Item 1\n- Item 2\n\n```go\nfmt.Println(\"Hello\")\n```"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "id": "770e8400-...",
    "title": "Heading",
    "content": {
      "type": "doc",
      "content": [
        {
          "type": "heading",
          "attrs": { "level": 1 },
          "content": [{ "type": "text", "text": "Heading" }]
        },
        {
          "type": "paragraph",
          "content": [
            { "type": "text", "text": "This is a paragraph with " },
            { "type": "text", "marks": [{ "type": "bold" }], "text": "bold" },
            { "type": "text", "text": " and " },
            { "type": "text", "marks": [{ "type": "italic" }], "text": "italic" },
            { "type": "text", "text": " text." }
          ]
        },
        {
          "type": "bullet_list",
          "content": [
            { "type": "list_item", "content": [{ "type": "text", "text": "Item 1" }] },
            { "type": "list_item", "content": [{ "type": "text", "text": "Item 2" }] }
          ]
        },
        {
          "type": "code_block",
          "attrs": { "language": "go" },
          "content": [{ "type": "text", "text": "fmt.Println(\"Hello\")" }]
        }
      ]
    },
    "version": 5,
    "updated_at": "2026-03-18T12:00:00Z"
  }
}
```

#### GET /api/v1/pages/:id/export/markdown

Export the page content as a Markdown string.

**Response (200 OK):**
```json
{
  "data": {
    "page_id": "770e8400-...",
    "title": "Getting Started",
    "markdown": "# Getting Started\n\nWelcome to Knomantem.\n\n## Installation\n\nRun the following command:\n\n```bash\nmake dev\n```\n",
    "exported_at": "2026-03-18T12:00:00Z"
  }
}
```

---

### 4.4 Search

#### GET /api/v1/search

Full-text search across all pages the authenticated user can access. Uses Bleve for indexing and querying.

**Query Parameters:**
| Param     | Type   | Default     | Description                                      |
|-----------|--------|-------------|--------------------------------------------------|
| q         | string | (required)  | Search query                                     |
| space     | UUID   | -           | Filter to a specific space                       |
| tags      | string | -           | Comma-separated tag names                        |
| freshness | string | -           | Filter by status: "fresh", "aging", "stale"      |
| from      | string | -           | Start date (ISO 8601) for updated_at range       |
| to        | string | -           | End date (ISO 8601) for updated_at range         |
| sort      | string | "relevance" | Sort by: "relevance", "updated", "freshness"     |
| cursor    | string | -           | Pagination cursor                                |
| limit     | int    | 20          | Items per page (max 100)                         |

**Response (200 OK):**
```json
{
  "data": {
    "results": [
      {
        "page_id": "770e8400-...",
        "title": "Getting Started",
        "excerpt": "Welcome to <mark>Knomantem</mark>. This guide covers...",
        "space": {
          "id": "660e8400-...",
          "name": "Engineering",
          "icon": "rocket"
        },
        "freshness": {
          "score": 85.50,
          "status": "fresh"
        },
        "tags": [
          { "name": "onboarding", "color": "#10B981" }
        ],
        "score": 0.95,
        "updated_at": "2026-03-18T10:00:00Z"
      }
    ],
    "facets": {
      "spaces": [
        { "id": "660e8400-...", "name": "Engineering", "count": 12 }
      ],
      "tags": [
        { "name": "onboarding", "count": 5 },
        { "name": "architecture", "count": 3 }
      ],
      "freshness": [
        { "status": "fresh", "count": 8 },
        { "status": "aging", "count": 3 },
        { "status": "stale", "count": 1 }
      ]
    },
    "query_time_ms": 45,
    "total": 12
  },
  "pagination": {
    "next_cursor": "dd0e8400-...",
    "has_more": false,
    "total": 12
  }
}
```

---

### 4.5 Freshness

#### GET /api/v1/pages/:id/freshness

Get the freshness record for a specific page.

**Response (200 OK):**
```json
{
  "data": {
    "id": "ee0e8400-...",
    "page_id": "770e8400-...",
    "owner_id": "550e8400-...",
    "freshness_score": 72.30,
    "review_interval_days": 30,
    "last_reviewed_at": "2026-03-10T10:00:00Z",
    "next_review_at": "2026-04-09T10:00:00Z",
    "last_verified_by": {
      "id": "550e8400-...",
      "display_name": "Jane Doe"
    },
    "last_verified_at": "2026-03-10T10:00:00Z",
    "status": "fresh",
    "decay_rate": 0.0333,
    "created_at": "2026-03-01T10:00:00Z",
    "updated_at": "2026-03-18T06:00:00Z"
  }
}
```

#### POST /api/v1/pages/:id/freshness/verify

Mark a page as verified. Resets freshness score to 100 and recalculates next review date.

**Request Body:**
```json
{
  "notes": "Reviewed and confirmed all steps are current"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "page_id": "770e8400-...",
    "freshness_score": 100.00,
    "status": "fresh",
    "last_verified_by": {
      "id": "550e8400-...",
      "display_name": "Jane Doe"
    },
    "last_verified_at": "2026-03-18T12:00:00Z",
    "next_review_at": "2026-04-17T12:00:00Z",
    "updated_at": "2026-03-18T12:00:00Z"
  }
}
```

#### PUT /api/v1/pages/:id/freshness/settings

Update freshness settings for a page.

**Request Body:**
```json
{
  "review_interval_days": 14,
  "decay_rate": 0.05,
  "owner_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (200 OK):**
```json
{
  "data": {
    "page_id": "770e8400-...",
    "review_interval_days": 14,
    "decay_rate": 0.05,
    "owner_id": "550e8400-...",
    "next_review_at": "2026-04-01T12:00:00Z",
    "updated_at": "2026-03-18T12:00:00Z"
  }
}
```

#### GET /api/v1/freshness/dashboard

Get an overview of freshness across all pages the user can access. Useful for identifying stale content that needs review.

**Query Parameters:**
| Param  | Type   | Default | Description                                |
|--------|--------|---------|--------------------------------------------|
| status | string | -       | Filter by: "fresh", "aging", "stale"       |
| sort   | string | "score" | Sort by: "score", "next_review", "updated" |
| cursor | UUID   | -       | Pagination cursor                          |
| limit  | int    | 20      | Items per page (max 100)                   |

**Response (200 OK):**
```json
{
  "data": {
    "summary": {
      "total_pages": 150,
      "fresh": 95,
      "aging": 35,
      "stale": 20,
      "average_score": 68.5
    },
    "pages": [
      {
        "page_id": "770e8400-...",
        "title": "Deployment Guide",
        "space": { "id": "660e8400-...", "name": "Engineering" },
        "freshness_score": 15.20,
        "status": "stale",
        "last_verified_at": "2026-01-05T10:00:00Z",
        "next_review_at": "2026-02-04T10:00:00Z",
        "owner": { "id": "550e8400-...", "display_name": "Jane Doe" }
      }
    ]
  },
  "pagination": {
    "next_cursor": "ff0e8400-...",
    "has_more": true,
    "total": 20
  }
}
```

---

### 4.6 Graph

#### GET /api/v1/pages/:id/graph

Get the immediate graph neighbors (directly connected pages) for a page.

**Query Parameters:**
| Param     | Type   | Default | Description                                     |
|-----------|--------|---------|-------------------------------------------------|
| edge_type | string | -       | Filter by edge type                             |
| direction | string | "both"  | "outgoing", "incoming", or "both"               |

**Response (200 OK):**
```json
{
  "data": {
    "center": {
      "id": "770e8400-...",
      "title": "Getting Started",
      "freshness_status": "fresh"
    },
    "edges": [
      {
        "id": "gg0e8400-...",
        "source": {
          "id": "770e8400-...",
          "title": "Getting Started",
          "freshness_status": "fresh"
        },
        "target": {
          "id": "880e8400-...",
          "title": "Installation",
          "freshness_status": "aging"
        },
        "edge_type": "reference",
        "metadata": {},
        "created_at": "2026-03-18T10:00:00Z"
      }
    ],
    "node_count": 5,
    "edge_count": 4
  }
}
```

#### POST /api/v1/pages/:id/graph/edges

Create a new graph edge from the specified page to a target page.

**Request Body:**
```json
{
  "target_page_id": "880e8400-e29b-41d4-a716-446655440000",
  "edge_type": "reference",
  "metadata": {
    "context": "Mentioned in installation section"
  }
}
```

**Response (201 Created):**
```json
{
  "data": {
    "id": "hh0e8400-...",
    "source_page_id": "770e8400-...",
    "target_page_id": "880e8400-...",
    "edge_type": "reference",
    "metadata": { "context": "Mentioned in installation section" },
    "created_by": "550e8400-...",
    "created_at": "2026-03-18T12:00:00Z"
  }
}
```

#### GET /api/v1/graph/explore

Explore the graph starting from a root node, expanding to a specified depth. Returns a subgraph suitable for visualization.

**Query Parameters:**
| Param     | Type   | Default | Description                          |
|-----------|--------|---------|--------------------------------------|
| root      | UUID   | (required) | Starting page ID                  |
| depth     | int    | 2       | How many hops from root (max 5)      |
| edge_type | string | -       | Filter by edge type                  |
| limit     | int    | 100     | Max nodes to return                  |

**Response (200 OK):**
```json
{
  "data": {
    "nodes": [
      {
        "id": "770e8400-...",
        "title": "Getting Started",
        "space_id": "660e8400-...",
        "freshness_status": "fresh",
        "connection_count": 5,
        "depth_from_root": 0
      },
      {
        "id": "880e8400-...",
        "title": "Installation",
        "space_id": "660e8400-...",
        "freshness_status": "aging",
        "connection_count": 2,
        "depth_from_root": 1
      }
    ],
    "edges": [
      {
        "source_id": "770e8400-...",
        "target_id": "880e8400-...",
        "edge_type": "reference"
      }
    ],
    "total_nodes": 2,
    "total_edges": 1,
    "truncated": false
  }
}
```

---

### 4.7 Tags

#### GET /api/v1/tags

List all tags, optionally filtered by name prefix.

**Query Parameters:**
| Param  | Type   | Default | Description            |
|--------|--------|---------|------------------------|
| q      | string | -       | Search by name prefix  |
| limit  | int    | 50      | Items per page         |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "ii0e8400-...",
      "name": "onboarding",
      "color": "#10B981",
      "is_ai_generated": false,
      "page_count": 12,
      "created_at": "2026-03-18T10:00:00Z"
    }
  ]
}
```

#### POST /api/v1/tags

Create a new tag.

**Request Body:**
```json
{
  "name": "architecture",
  "color": "#6366F1"
}
```

**Response (201 Created):**
```json
{
  "data": {
    "id": "jj0e8400-...",
    "name": "architecture",
    "color": "#6366F1",
    "is_ai_generated": false,
    "created_at": "2026-03-18T12:00:00Z"
  }
}
```

#### POST /api/v1/pages/:id/tags

Associate tags with a page. Accepts an array to allow bulk tagging.

**Request Body:**
```json
{
  "tags": [
    { "tag_id": "ii0e8400-...", "confidence_score": 1.0 },
    { "tag_id": "jj0e8400-...", "confidence_score": 0.85 }
  ]
}
```

**Response (200 OK):**
```json
{
  "data": {
    "page_id": "770e8400-...",
    "tags": [
      {
        "id": "ii0e8400-...",
        "name": "onboarding",
        "color": "#10B981",
        "confidence_score": 1.0
      },
      {
        "id": "jj0e8400-...",
        "name": "architecture",
        "color": "#6366F1",
        "confidence_score": 0.85
      }
    ]
  }
}
```

---

## 5. UI Wireframes

### 5.1 Home / Dashboard

```
+-----------------------------------------------------------------------+
|  [=] Knomantem                          [Search...]        [@] Jane   |
+------------------+----------------------------------------------------+
|                  |                                                     |
|  SPACES          |   RECENT PAGES                                     |
|  ----------------+                                                     |
|  > Engineering   |   +------------------------------------------+     |
|  > Design        |   | [book] Getting Started          [GREEN]  |     |
|  > Product       |   | Engineering . Updated 2h ago              |     |
|    Marketing     |   +------------------------------------------+     |
|                  |   | [page] API Reference            [YELLOW] |     |
|  [+ New Space]   |   | Engineering . Updated 3d ago              |     |
|                  |   +------------------------------------------+     |
|  ----------------+   | [page] Deployment Guide         [RED]    |     |
|                  |   | Engineering . Updated 45d ago             |     |
|  FAVORITES       |   +------------------------------------------+     |
|  Getting Started |                                                     |
|  API Reference   |   FRESHNESS ALERTS                                 |
|                  |   ------------------------------------------------ |
|  ----------------+   ! 5 pages need review                            |
|                  |   +------------------------------------------+     |
|  TAGS            |   | [!] Deployment Guide         Score: 15%  |     |
|  # onboarding    |   |     Last verified 45 days ago  [Verify]   |     |
|  # architecture  |   +------------------------------------------+     |
|  # api           |   | [!] CI/CD Pipeline           Score: 32%  |     |
|                  |   |     Last verified 28 days ago  [Verify]   |     |
|                  |   +------------------------------------------+     |
|                  |                                                     |
|                  |   ACTIVITY                                          |
|                  |   ------------------------------------------------ |
|                  |   Jane edited "Getting Started" - 2h ago            |
|                  |   Alex created "New Feature Spec" - 5h ago          |
|                  |   Jane verified "API Reference" - 1d ago            |
|                  |                                                     |
+------------------+----------------------------------------------------+
```

**Behavior Details:**

- The left sidebar is always visible and 240px wide. It contains the spaces list (collapsible), a favorites section (pages the user has starred), and a tag cloud for quick filtering.
- "Recent Pages" shows the last 10 pages the user viewed or edited, each displayed as a card with the page title, parent space name, last updated timestamp, and a freshness badge (colored dot: green for fresh, yellow for aging, red for stale).
- "Freshness Alerts" lists pages with a freshness status of "stale" or "aging" that the current user owns or has edit access to. Each alert has a one-click "Verify" button that triggers the verify endpoint.
- "Activity" is a reverse-chronological feed of recent actions (edits, creates, verifications) across all spaces the user can access.

---

### 5.2 Space View

```
+-----------------------------------------------------------------------+
|  [=] Knomantem                          [Search...]        [@] Jane   |
+------------------+----------------------------------------------------+
|                  |  Engineering > Getting Started                      |
|  SPACES          |  ------------------------------------------------  |
|  ----------------+                                                     |
|  v Engineering   |  +-----------------------------------------------+ |
|    > Getting     |  |                                               | |
|      Started     |  |  # Getting Started              [GREEN 85%]  | |
|      v Guides    |  |                                               | |
|        Install   |  |  Welcome to the Engineering knowledge base.   | |
|        Config    |  |  This space contains all technical            | |
|        Deploy    |  |  documentation for the team.                  | |
|    > API Ref     |  |                                               | |
|    > Runbooks    |  |  ## Quick Links                               | |
|  > Design        |  |  - [[Installation Guide]]                     | |
|  > Product       |  |  - [[Configuration]]                          | |
|                  |  |  - [[Deployment Guide]]                       | |
|  [+ New Page]    |  |                                               | |
|                  |  |  ## Recent Updates                             | |
|                  |  |  The deployment process has been updated to    | |
|                  |  |  use the new CI/CD pipeline...                 | |
|                  |  |                                               | |
|                  |  +-----------------------------------------------+ |
|                  |                                                     |
+------------------+----------------------------------------------------+
```

**Behavior Details:**

- The left panel shows the page tree for the currently selected space. Nodes are collapsible and expandable by clicking the arrow icon. The tree supports drag-and-drop: dragging a node over another node makes it a child; dragging between nodes reorders siblings. Drop targets are indicated with a blue horizontal line (between siblings) or a blue highlight (reparenting).
- The tree lazy-loads children when a node is first expanded (for spaces with hundreds of pages). Expanded state is persisted locally.
- The right panel shows a read-only preview of the selected page. Double-clicking the content area or clicking an "Edit" button opens the full page editor (Section 5.3).
- Breadcrumb navigation at the top shows: Space Name > Parent Page > Current Page. Each segment is clickable.
- The "New Page" button creates a new page as a sibling of the currently selected page. Holding Shift creates a child page instead.
- Each tree node displays the page icon, title (truncated to fit), and a small freshness dot (colored circle).

---

### 5.3 Page Editor

```
+-----------------------------------------------------------------------+
|  [=] Knomantem                          [Search...]        [@] Jane   |
+------------------+---------------------------------+------------------+
|                  |  Engineering > Guides > Install  |                  |
|  PAGE TREE       |  [book] Installation Guide       |  METADATA        |
|  (same as 5.2)   |  [GREEN 92%] v3 . Saved 2m ago   |                  |
|                  |---------------------------------|  Tags:           |
|                  |                                  |  [onboarding]    |
|                  |  # Installation Guide            |  [infrastructure]|
|                  |                                  |  [+ Add tag]     |
|                  |  ## Prerequisites                |                  |
|                  |                                  |  ---------------  |
|                  |  Before installing, ensure you   |  Freshness:      |
|                  |  have the following:             |  Score: 92%      |
|                  |                                  |  Status: Fresh   |
|                  |  - Go 1.22+                      |  Interval: 30d   |
|                  |  - PostgreSQL 16                 |  Next review:    |
|                  |  - Docker                        |  Apr 14, 2026    |
|                  |                                  |  [Verify Now]    |
|                  |  ## Steps              +---------+                  |
|                  |                        | B I S   |  ---------------  |
|                  |  1. Clone the repo:    | H1 H2   |  Linked Pages:   |
|                  |                        | [] - 1.  |  > Configuration |
|                  |  ```bash               | <> ""    |  > Deploy Guide  |
|                  |  git clone ...         +---------+  [+ Add link]    |
|                  |  ```                             |                  |
|                  |                                  |  ---------------  |
|                  |  2. Run setup:                   |  Version History |
|                  |                                  |  v3 - 2h ago     |
|                  |  ```bash                         |  v2 - 1d ago     |
|                  |  make setup                      |  v1 - 3d ago     |
|                  |  ```                             |  [View all]      |
|                  |                                  |                  |
+------------------+---------------------------------+------------------+
```

**Behavior Details:**

- The editor area occupies the center column and uses the AppFlowy Editor widget. Content is editable in place with a WYSIWYG experience.
- A floating toolbar appears when text is selected, offering: Bold, Italic, Strikethrough, Inline Code, Link, Highlight, and Text Color. The toolbar is positioned above the selection and follows the cursor.
- Slash commands are triggered by typing `/` at the beginning of a line or after a space. A dropdown palette appears with options: Heading 1-3, Bullet List, Numbered List, Checklist, Code Block, Block Quote, Divider, Image, Page Link, Table. Typing filters the list. Pressing Enter inserts the selected block.
- Page links (typed as `[[Page Title]]`) trigger an autocomplete dropdown that searches page titles across the current space. Selecting a page creates a graph edge of type "reference" automatically.
- The header shows: page icon (clickable to change), page title (editable inline), freshness badge, version number, and last saved time. Auto-save triggers 2 seconds after the user stops typing.
- The right sidebar (300px wide, collapsible) shows metadata panels:
  - **Tags**: Assigned tags as colored chips. "Add tag" opens an autocomplete dropdown.
  - **Freshness**: Current score, status, review interval, next review date, and a "Verify Now" button.
  - **Linked Pages**: Pages connected via graph edges. Click to navigate. "Add link" opens a page search dialog.
  - **Version History**: Recent versions with timestamps and author names. Clicking a version opens a diff view.
- Keyboard shortcuts: Ctrl+B (bold), Ctrl+I (italic), Ctrl+S (force save), Ctrl+K (insert link), Ctrl+Shift+H (toggle heading).

---

### 5.4 Search Results

```
+-----------------------------------------------------------------------+
|  [=] Knomantem                          [Search...]        [@] Jane   |
+------------------+----------------------------------------------------+
|                  |                                                     |
|  SPACES          |  Search: "deployment pipeline"                      |
|  (sidebar)       |  ------------------------------------------------  |
|                  |  Filters:                                           |
|                  |  [Engineering v] [# infrastructure] [Fresh v]      |
|                  |  [Mar 1 - Mar 18]  [x Clear all]                   |
|                  |                                                     |
|                  |  12 results (45ms)    Sort: [Relevance v]           |
|                  |  ------------------------------------------------  |
|                  |                                                     |
|                  |  +-----------------------------------------------+ |
|                  |  | [page] Deployment Guide              [RED]    | |
|                  |  | Engineering                                    | |
|                  |  | ...the <b>deployment</b> <b>pipeline</b> has   | |
|                  |  | been updated to use GitHub Actions for         | |
|                  |  | continuous <b>deployment</b>...                 | |
|                  |  | # infrastructure  # devops                     | |
|                  |  | Updated 45d ago                                | |
|                  |  +-----------------------------------------------+ |
|                  |                                                     |
|                  |  +-----------------------------------------------+ |
|                  |  | [page] CI/CD Pipeline Setup          [YELLOW] | |
|                  |  | Engineering                                    | |
|                  |  | ...configuring the <b>deployment</b>            | |
|                  |  | <b>pipeline</b> involves setting up the        | |
|                  |  | workflow file and secrets...                   | |
|                  |  | # infrastructure  # ci-cd                      | |
|                  |  | Updated 28d ago                                | |
|                  |  +-----------------------------------------------+ |
|                  |                                                     |
|                  |  +-----------------------------------------------+ |
|                  |  | [page] Release Process               [GREEN]  | |
|                  |  | Product                                        | |
|                  |  | ...after the build passes the                  | |
|                  |  | <b>pipeline</b>, the release manager triggers   | |
|                  |  | the <b>deployment</b>...                        | |
|                  |  | # releases  # process                          | |
|                  |  | Updated 2d ago                                 | |
|                  |  +-----------------------------------------------+ |
|                  |                                                     |
|                  |  [Load more...]                                     |
|                  |                                                     |
+------------------+----------------------------------------------------+
```

**Behavior Details:**

- The search bar at the top is pre-focused when navigating to this screen. It supports real-time search-as-you-type with a 300ms debounce.
- Filter chips below the search bar allow narrowing results. Available filters:
  - **Space**: Dropdown of all spaces the user can access.
  - **Tags**: Multi-select tag picker with autocomplete.
  - **Freshness**: Dropdown with "Fresh", "Aging", "Stale" options.
  - **Date range**: Date picker for filtering by `updated_at` range.
  - "Clear all" removes all active filters.
- Each result card shows:
  - Page icon and title (clickable, navigates to the page editor).
  - Space name (clickable, navigates to the space view).
  - Excerpt with search term matches highlighted using `<mark>` tags. Excerpts are ~150 characters centered around the first match.
  - Tags displayed as small colored chips.
  - Freshness badge (colored dot with status label).
  - "Updated X ago" relative timestamp.
- Sort options: "Relevance" (default, by Bleve score), "Last Updated" (most recent first), "Freshness" (lowest freshness first, for finding stale content).
- The facet sidebar (part of the search response) shows counts per space, per tag, and per freshness status to help users understand the distribution of results.
- Results use cursor-based pagination. "Load more" appends the next page of results.

---

### 5.5 Graph View

```
+-----------------------------------------------------------------------+
|  [=] Knomantem                          [Search...]        [@] Jane   |
+------------------+----------------------------------------------------+
|                  |  Graph: Getting Started (depth: 2)                  |
|  SPACES          |  ------------------------------------------------  |
|  (sidebar)       |  Filters: [All types v]  Depth: [2 v]              |
|                  |                                                     |
|                  |       (Config)------+                               |
|                  |         /YELLOW     |                               |
|                  |        /            | related                       |
|                  |  (Install)     reference                            |
|                  |    YELLOW \         |                               |
|                  |  reference \        v                               |
|                  |            [Getting Started]                        |
|                  |              GREEN / \                              |
|                  |          ref /     \ depends_on                     |
|                  |             /       \                               |
|                  |        (API Ref)   (Deploy)                         |
|                  |         GREEN        RED                            |
|                  |                       |                             |
|                  |                       | derived_from                |
|                  |                       v                             |
|                  |                   (Runbook)                         |
|                  |                    YELLOW                           |
|                  |                                                     |
|                  |  ------------------------------------------------  |
|                  |  Nodes: 6  |  Edges: 5  |  [Fit] [Zoom+] [Zoom-]   |
|                  |                                                     |
+------------------+----------------------------------------------------+
```

**Behavior Details:**

- The graph is rendered using a force-directed layout algorithm (e.g., using the `graphview` Flutter package or a custom Canvas-based renderer). Nodes repel each other; edges act as springs.
- **Node appearance:**
  - Each node is a rounded rectangle containing the page title (truncated to 20 characters).
  - Node size scales with connection count: more connections produce larger nodes (min 40px, max 100px diameter).
  - Node fill color indicates freshness: green (#10B981) for fresh, yellow (#F59E0B) for aging, red (#EF4444) for stale.
  - The center node (the page from which the graph was opened) has a thicker border and is slightly larger.
- **Edge appearance:**
  - Edges are curved lines connecting nodes.
  - Edge labels display the relationship type ("reference", "depends_on", "related", "derived_from").
  - Edge thickness can vary by type (e.g., "depends_on" is thicker than "reference").
  - Edge color is a neutral gray (#9CA3AF).
- **Interactions:**
  - Click a node to navigate to that page in the editor.
  - Hover over a node to see a tooltip with: full title, space name, freshness score, and connection count.
  - Drag nodes to reposition them (the physics simulation pauses while dragging and resumes on release).
  - Scroll to zoom in/out. Pinch-to-zoom on touch devices.
  - "Fit" button resets the zoom to fit all nodes in the viewport.
- **Filters:**
  - Edge type dropdown filters which edges are displayed. Options: "All types", "reference", "related", "depends_on", "derived_from".
  - Depth slider (1-5) controls how many hops from the center node are fetched.
- The status bar at the bottom shows the current node count, edge count, and zoom controls.

---

## 6. Success Criteria

Each criterion must be demonstrated and verified before the POC is considered complete.

| # | Capability | Criterion | Measurement |
|---|------------|-----------|-------------|
| 1 | Editor | Can create, edit, and save a document with headings (H1-H3), bold, italic, strikethrough, bullet lists, numbered lists, checklists, code blocks, block quotes, and internal page links | Manual test: create a document using every supported block type, save, reload, and confirm all formatting is preserved |
| 2 | Markdown | Can import a 1,000-line Markdown file and export it back to Markdown with less than 5% formatting loss | Automated test: import a reference Markdown file, export it, and run a diff. Formatting loss is measured as the percentage of lines that differ (excluding whitespace normalization) |
| 3 | Page Tree | Can create a space with at least 3 levels of nested pages and reorder them via drag-and-drop | Manual test: create pages at depths 0, 1, 2, and 3. Drag a depth-2 page to become a depth-1 sibling. Confirm the tree and database are consistent |
| 4 | Search Speed | Full-text search returns relevant results in under 200ms for a 1,000-page dataset | Automated test: seed 1,000 pages with realistic content, execute 10 different search queries, and confirm all return in under 200ms (measured server-side, excluding network latency) |
| 5 | Search Filters | Can filter search results by space, tags, and freshness status, with correct result counts | Manual test: apply each filter type individually and in combination. Confirm result counts match expected values |
| 6 | Freshness Decay | Pages display a correct freshness score that decays over time according to the configured decay rate | Automated test: create a page, advance the clock (or set `last_verified_at` in the past), run the freshness worker, and confirm the score matches the expected decay formula output within 0.1% tolerance |
| 7 | Freshness Verify | Can verify a page and see the freshness score reset to 100 with the next review date recalculated | Manual test: find a page with a decayed score, click "Verify Now", confirm the score is 100 and `next_review_at` is set to now + review_interval_days |
| 8 | Graph | Can visualize page connections in an interactive force-directed graph with at least 10 nodes and 15 edges | Manual test: create a network of 10+ pages with various edge types, open the graph view, confirm all nodes and edges render, and confirm click-to-navigate works |
| 9 | API Performance | All non-search API endpoints respond in under 50ms at the 95th percentile | Automated test: run a load test with 50 concurrent users for 60 seconds using a tool such as k6 or vegeta. Measure p95 latency for each endpoint group |
| 10 | Architecture | All business logic in the service layer is testable without a database dependency via repository interfaces | Code review: confirm every service constructor accepts interfaces (not concrete types). Write at least one unit test per service that uses a mock repository and passes |

---

## Appendix A: Technology Stack Summary

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Backend Runtime | Go | 1.22+ | API server |
| HTTP Framework | Echo | v4 | Routing, middleware, request handling |
| Database | PostgreSQL | 16 | Primary data store |
| Database Driver | pgx | v5 | PostgreSQL driver for Go |
| Search Engine | Bleve | v2 | Full-text search with faceted queries |
| Markdown Parser | Goldmark | v1.7+ | Markdown to AST and AST to Markdown |
| Auth | JWT (golang-jwt) | v5 | Token-based authentication |
| Authorization | Casbin | v2 | RBAC policy enforcement |
| Frontend Framework | Flutter | 3.x | Cross-platform UI |
| Editor | AppFlowy Editor | latest | Rich text editing with JSON AST |
| State Management | Riverpod | 3.0 | Reactive state management |
| Routing | GoRouter | latest | Declarative Flutter routing |
| HTTP Client | Dio | latest | HTTP client with interceptors |
| Containerization | Docker | latest | Development and deployment |
| Database Migrations | golang-migrate | v4 | Schema versioning |

## Appendix B: Environment Variables

```env
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgres://knomantem:knomantem@localhost:5432/knomantem?sslmode=disable

# JWT
JWT_SECRET=your-secret-key-min-32-chars-long
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Bleve
BLEVE_INDEX_PATH=./data/search.bleve

# Casbin
CASBIN_MODEL_PATH=./config/casbin_model.conf
CASBIN_POLICY_PATH=./config/casbin_policy.csv

# Freshness Worker
FRESHNESS_WORKER_INTERVAL=1h
FRESHNESS_DECAY_RATE=0.0333

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

## Appendix C: Docker Compose (Development)

```yaml
version: "3.9"

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: knomantem
      POSTGRES_PASSWORD: knomantem
      POSTGRES_DB: knomantem
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U knomantem"]
      interval: 5s
      timeout: 5s
      retries: 5

  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://knomantem:knomantem@db:5432/knomantem?sslmode=disable
      JWT_SECRET: dev-secret-key-change-in-production-32chars
      BLEVE_INDEX_PATH: /data/search.bleve
      ENV: development
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - blevedata:/data

volumes:
  pgdata:
  blevedata:
```

## Appendix D: Makefile Targets

```makefile
.PHONY: dev build test migrate seed lint

# Start development server with hot reload
dev:
	air -c .air.toml

# Build production binary
build:
	go build -o bin/server ./cmd/server

# Run all tests
test:
	go test ./... -v -cover

# Run database migrations
migrate:
	migrate -database "$(DATABASE_URL)" -path migrations up

# Rollback last migration
migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down 1

# Seed database with sample data (1000 pages for search testing)
seed:
	go run ./cmd/seed/main.go

# Run linter
lint:
	golangci-lint run ./...

# Start all services via Docker Compose
up:
	docker compose up -d

# Stop all services
down:
	docker compose down

# Run Flutter web app
web:
	cd web && flutter run -d chrome
```
