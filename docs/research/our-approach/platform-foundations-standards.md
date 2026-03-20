# Platform Foundations: Standards & Decisions

> Research conducted 2026-03-20. Applies to the knomantem Go backend (Echo v4, PostgreSQL 16, pgx v5).

---

## 1. OpenAPI 3.0 for Go / Echo Backends

### Hand-written YAML vs. swaggo annotations

Two mainstream approaches exist for producing an OpenAPI spec from a Go backend:

**swaggo/swag** — generates `openapi.yaml` / `swagger.json` from Go doc-comment annotations (`// @Summary`, `// @Param`, etc.) embedded in handler files. The generator is run as a pre-build step (`swag init`). Advantages: spec stays close to the code, less risk of drift. Disadvantages: annotation syntax is verbose and error-prone, generated output is hard to diff, and the tool imposes a specific project structure. It also does not support OpenAPI 3.1 (only 3.0).

**Hand-written YAML** — a single `docs/openapi.yaml` owned by the team and validated with a linter (e.g., `spectral`, `vacuum`, or the Swagger Editor). Advantages: full control over schema design, easy to review in PRs, can be written before implementation to drive contract-first development, no code-gen step in CI. Disadvantages: must be updated manually when handlers change; drift is the main risk.

**Our decision: hand-written YAML.** The codebase already has clearly separated handler interfaces, so the spec can track the `routes.go` inventory. A `make lint-api` step using `spectral lint docs/openapi.yaml` keeps it honest.

### Request / response schema conventions

- All request bodies use `application/json`.
- All responses use `application/json` with a consistent envelope:
  - Single-item endpoints: `{ "data": { ... } }` or the object directly (our handlers use the direct form for simplicity).
  - List endpoints with pagination: `{ "data": [...], "pagination": { "next_cursor": "...", "has_more": true, "total": 42 } }`.
- Error responses use `{ "error": { "code": "NOT_FOUND", "message": "..." } }`.
- Timestamps are ISO-8601 UTC strings (`2006-01-02T15:04:05Z`).
- UUIDs are `type: string, format: uuid`.
- Optional nullable fields use `nullable: true` (OpenAPI 3.0) rather than `oneOf` with null (which is 3.1).

### JWT Bearer auth documentation

Security scheme definition:

```yaml
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

Applied globally or per-operation via:

```yaml
security:
  - bearerAuth: []
```

Public endpoints (register, login, refresh) override with `security: []`.

### Pagination pattern

This backend uses opaque cursor-based pagination (not offset). The pattern:

- Query params: `cursor` (string, opaque, from previous response) and `limit` (integer, 1–100, default 20).
- Response wrapper includes `next_cursor` (null when no more pages), `has_more` (boolean), `total` (integer, total matching records).
- Cursors encode enough state to reproduce the next page (typically an encoded timestamp + ID pair).

Offset-based pagination is intentionally avoided: it suffers from the "missing row" problem on live datasets and does not scale past ~10k rows.

---

## 2. Go Service Layer Testing

### Table-driven tests

The Go testing community (including the standard library itself) strongly favours table-driven tests for exhaustive coverage of a single function with many input combinations. The pattern:

```go
cases := []struct {
    name  string
    input SomeInput
    want  SomeOutput
    err   error
}{
    {"happy path", ..., ..., nil},
    {"missing field", ..., ..., ErrValidation},
}
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        got, err := svc.DoThing(ctx, tc.input)
        // assert
    })
}
```

Use `t.Run` so each case appears as a named sub-test in `go test -v` output and can be run in isolation with `-run TestFoo/happy_path`.

### Mock generation: mockery vs mockgen vs hand-written

| Tool | Pros | Cons |
|---|---|---|
| **mockery v2** | Interface-aware, generates readable mocks, supports `RETURNS` / `CALLED` assertions, integrates with testify/mock | Requires code-gen step, adds testify dependency |
| **mockgen (google/mock)** | Mature, strict mode catches unused calls, good gomock integration | Verbose generated code, two syntax variants |
| **Hand-written stubs** | Zero dependencies, full control, readable, no code-gen | More boilerplate per interface, easy to forget a method when interface grows |

**Our decision: hand-written stubs/mocks.** The service interfaces are small (5–10 methods). The pattern used in `auth_service_test.go` and `page_service_test.go` — a `mockXxxRepo` struct with a `map` for state and optional function overrides (e.g. `createFn func(...)`) — hits the right balance of readability and flexibility. When an interface grows beyond ~12 methods, reconsider mockery.

### Test naming conventions

Follow `Test<TypeName>_<MethodName>_<Scenario>`:

- `TestAuthService_Register_Success`
- `TestAuthService_Register_DuplicateEmail`
- `TestFreshnessService_RunDecay_ScoreClampsToZero`

This makes `go test -v` output self-documenting and allows targeted runs with `-run`.

### What to test vs. what to skip

**Test:**
- Business logic (domain rules, calculations, state transitions).
- Error paths: not-found, validation, conflict, unauthorized.
- Side effects: verify that dependent repository methods were called (e.g., search was indexed after create).
- Boundary conditions: score clamping, empty inputs, zero-value UUIDs.

**Skip / defer:**
- Repository implementations (integration tests with a real DB, not unit tests).
- HTTP handler binding/routing (covered by integration tests or curl smoke tests).
- Token parsing internals of third-party libraries.
- Concurrency edge cases in the presence hub (require a dedicated race-condition test suite).

### Pattern in use

Each service test file in this repo follows this structure:

1. **Hand-written mock structs** implementing the relevant repository interface.
2. **Optional function-hook fields** (`createFn`, `updateFn`) on mocks for injecting errors without subclassing.
3. **Helper constructors** (`newAuthSvc`, `newPageSvc`) that wire the mock into the real service.
4. **Test functions** grouped by method, named by scenario, using plain `testing.T` assertions (no testify required).

---

## 3. pgvector in PostgreSQL

### Adding pgvector to an existing schema

pgvector is a PostgreSQL extension. It must be installed on the server (`CREATE EXTENSION IF NOT EXISTS vector;`) before any `vector` columns or operators can be used. On managed services (RDS, Supabase, Neon, Cloud SQL) it is pre-installed; on self-hosted Postgres it requires the OS package `postgresql-<version>-pgvector`.

**Migration approach:**

```sql
-- up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE page_embeddings (
    page_id    UUID PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    embedding  vector(384),
    model      VARCHAR(100) NOT NULL DEFAULT 'all-MiniLM-L6-v2',
    indexed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ON page_embeddings USING hnsw (embedding vector_cosine_ops);

-- down
DROP TABLE IF EXISTS page_embeddings;
-- Only drop extension if no other tables use it (check pg_depend or be conservative):
-- DROP EXTENSION IF EXISTS vector;
```

The down migration intentionally omits `DROP EXTENSION` in the typical case to avoid breaking other tables that may have been added by concurrent migrations. It is safe to drop it only in a clean-room dev setup.

### Index types: ivfflat vs hnsw

| | **ivfflat** | **hnsw** |
|---|---|---|
| Build time | Fast | Slow (builds graph incrementally) |
| Query speed | Good | Excellent |
| Memory | Low | Higher (graph lives in RAM) |
| Recall accuracy | Good (tunable via `probes`) | Very high (tunable via `ef_search`) |
| Suitable for | Batch-indexed, large static datasets | Real-time inserts, high-recall requirements |
| Minimum rows to build | Needs `lists` rows before it helps | Works immediately |

**Our decision: HNSW.** Knowledge-management pages are inserted and updated individually in near-real-time (not batch). HNSW provides better recall (~0.98+) and handles incremental inserts well. The embedding dimension is small (384 for MiniLM), so the memory overhead is modest even for millions of pages.

Operator classes for cosine similarity: `vector_cosine_ops`. For dot-product: `vector_ip_ops`. For L2: `vector_l2_ops`. Cosine similarity is the standard for text embeddings because it is magnitude-invariant.

### Typical embedding dimensions

| Model | Dimensions | Notes |
|---|---|---|
| `all-MiniLM-L6-v2` (sentence-transformers) | **384** | Best size/quality trade-off for semantic search; runs on CPU |
| `text-embedding-3-small` (OpenAI) | **1536** | Default; can be reduced to 256/512 with matryoshka |
| `text-embedding-3-large` (OpenAI) | **3072** | Highest quality, highest cost |
| `mistral-embed` (Mistral) | **1024** | Good multilingual quality |
| `nomic-embed-text` (Nomic) | **768** | Strong open-source alternative |

**Our decision: 384 (MiniLM-L6-v2).** The model runs locally without GPU, has Apache-2.0 licence, and 384 dimensions keeps the HNSW index small. If quality proves insufficient, migrating to 1536 (OpenAI) requires only a schema change and re-indexing.

### Storing and querying with pgx v5

pgvector provides a `pgvector-go` driver extension, but for pgx v5 the recommended approach is:

1. Represent the vector as `pgtype.Array[float32]` or, if using the `pgvector-go` library, as `pgvector.Vector`.
2. Register the codec on the connection pool:
   ```go
   import "github.com/pgvector/pgvector-go"
   conn.TypeMap().RegisterType(&pgtype.Type{
       Name:  "vector",
       OID:   pgvector.VectorOID,
       Codec: pgvector.VectorCodec{},
   })
   ```
3. Query for nearest neighbours:
   ```sql
   SELECT page_id, embedding <=> $1 AS distance
   FROM page_embeddings
   ORDER BY embedding <=> $1
   LIMIT $2;
   ```
   (`<=>` is cosine distance; `<#>` is negative inner product; `<->` is L2.)

For a repository stub that does not yet have a real implementation, it is acceptable to define the interface and leave the pgx implementation for the embedding sprint.
