# Backend Analysis: Knomantem Go API Server

## Overview

The Go backend (`cmd/`, `internal/`, `pkg/`) follows clean architecture with strict dependency inversion. Echo v4 HTTP framework, PostgreSQL 16 via pgx v5, Bleve embedded search, JWT + Casbin RBAC. The codebase is the most complete and production-ready layer of the project.

---

## Architecture & Layering

```
Handler → Service → Domain ← Repository
```

This is properly enforced:
- `internal/domain/` has zero external imports (only stdlib + uuid)
- `internal/service/` depends on domain interfaces only
- `internal/handler/` depends on service layer only
- `internal/repository/postgres/` and `pkg/search/` implement domain interfaces

The separation is genuine, not nominal. Services are testable with mock repositories without spinning up a database.

---

## API Endpoints

**Auth** (`/auth/*`)
- POST `/auth/login` — JWT + refresh token issuance
- POST `/auth/register` — user creation + immediate login
- POST `/auth/refresh` — token rotation
- GET `/auth/me` — current user profile

**Spaces** (`/spaces/*`)
- Full CRUD: GET list, GET single, POST, PUT, DELETE

**Pages** (`/spaces/:id/pages`, `/pages/:id`)
- GET `/spaces/:id/pages` — list pages in space
- POST `/spaces/:id/pages` — create page
- GET `/pages/:id` — get page with full content
- PUT `/pages/:id` — update title/content/icon
- DELETE `/pages/:id` — delete page
- PUT `/pages/:id/move` — reparent/reorder
- GET `/pages/:id/versions` — version history

**Search** (`/search`)
- GET with query params: q, space, tags, freshness, from, to, sort, cursor, limit
- Returns results with freshness data and tag facets

**Freshness** (`/pages/:id/freshness`, `/freshness/*`)
- GET `/pages/:id/freshness` — current score + status
- POST `/pages/:id/freshness/verify` — manual verification
- PUT `/pages/:id/freshness/settings` — configure decay rate + review interval
- GET `/freshness/dashboard` — paginated list sorted by freshness with aggregate stats

**Graph** (`/pages/:id/graph`, `/graph/*`)
- GET `/pages/:id/graph` — direct neighbors
- POST `/pages/:id/graph/edges` — create typed edge
- GET `/graph/explore` — multi-hop traversal from root with depth + edge type filters

**Tags** (`/tags`, `/pages/:id/tags`)
- GET `/tags` — list with optional prefix filter
- POST `/tags` — create tag
- POST `/pages/:id/tags` — assign tags with confidence scores

**Presence** (WebSocket)
- WSS `/presence/:pageId` — join/leave page presence hub

---

## Services

| Service | Lines | Status |
|---|---|---|
| `auth_service.go` | ~150 | Production-ready |
| `page_service.go` | ~200 | Production-ready |
| `space_service.go` | ~120 | Production-ready |
| `freshness_service.go` | 194 | Production-ready |
| `graph_service.go` | 130 | Production-ready |
| `tag_service.go` | 43 | Minimal but correct |
| `search_service.go` | 27 | Thin — needs freshness-weight integration |
| `presence_service.go` | ~80 | Functional in-memory hub |

**`search_service.go` is the weakest link.** It's a pass-through with zero business logic. The freshness weighting that is a core differentiator is not implemented here. `SearchItem` domain type carries `Freshness FreshnessBrief` but the ranking formula ignores it.

---

## Database Schema

6 migrations cover:
1. `users` — id, email, display_name, password_hash, role, timestamps
2. `pages` — id, space_id, parent_id, title, content (JSONB), icon, position, is_template, freshness tracking columns, timestamps
3. `edges` — id, source_page_id, target_page_id, edge_type, metadata (JSONB), created_by, timestamps
4. `tags` / `page_tags` — tag catalog + junction table with confidence_score
5. `notifications` — user_id, type, page_id, message, read_at, timestamps
6. `audit_log` — actor_id, action, resource_type, resource_id, metadata, timestamps

**Freshness is a first-class citizen on the `pages` table**, not a separate table — this is a deliberate design choice that simplifies queries but makes the pages table wide. The `freshness_records` referenced in some service code may be a view or an older design decision.

**Graph edges are typed with 6 edge types** enforced at the application layer (not a DB constraint): `reference`, `depends_on`, `supersedes`, `related_to`, `child_of`, `backlink`.

---

## Background Workers

**`freshness_checker.go`** — runs every 6 hours:
- Queries pages with `score < 30` (stale threshold)
- Applies linear decay: `100 × (1 − rate × days / interval)`
- Fires notifications when score drops below threshold
- **Note:** the worker only processes already-stale pages (score < 30). Pages in the Aging range (30-70) do not get decay applied on each run — this may be a bug or a deliberate optimization.

**`search_indexer.go`** — watches for DB change events:
- Indexes new/updated pages in Bleve
- Rebuilds index from scratch on startup if missing

**`ai_service.go`** — referenced in the directory structure docs:
- Listed as a background worker for "summarisation, tag suggestion, relation extraction"
- **Does not exist as a file** in the current codebase — it is an aspirational placeholder in the architecture doc

---

## Production Readiness Assessment

**Production-ready:**
- Authentication flow (JWT + bcrypt + Casbin RBAC)
- All CRUD operations (pages, spaces, users, edges, tags)
- Freshness decay engine
- Full-text search (Bleve, basic BM25)
- Docker multi-stage build (~50MB image)
- Structured JSON logging via slog
- Graceful shutdown with context cancellation
- Environment-variable configuration (12-factor compliant)
- Rate limiting middleware

**Rough / incomplete:**
- Search ranking (no freshness weighting, no scoring formula)
- No implicit edge creation from page content (backlinks not auto-created)
- `ai_service.go` worker doesn't exist
- Freshness worker only processes `score < 30`, not Aging pages
- `FreshnessStatus` on the `Page` domain object exists but how it gets populated from the freshness record is not clear (may be a JOIN or a denormalized column)

**Absent entirely:**
- OpenAPI spec (referenced in architecture doc as `docs/openapi.yaml`, does not exist in codebase)
- Comments/discussion system
- Public page sharing
- Webhook/event system
- Audit log endpoints (table exists, no handler)

---

## Code Quality

Consistently high. Clean error handling (domain errors mapped to HTTP status codes), no logic in handlers, proper context propagation throughout. The repository layer uses pgx prepared statements and parameterized queries consistently (no SQL injection surface). Test coverage is sparse (~7 test files) but the existing tests are well-written.
