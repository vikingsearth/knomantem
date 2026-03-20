# Database Reference — pgx v5 + PostgreSQL 16

> Best practices for pgx v5 pool configuration, transactions, PostgreSQL-specific patterns, migrations, and production tuning.

---

## 1. pgxpool Configuration

`pgxpool.Pool` is the only concurrency-safe way to use pgx. A single `*pgx.Conn` must not be shared between goroutines.

```go
// internal/infrastructure/database.go

package infrastructure

import (
    "context"
    "fmt"
    "runtime"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("NewPool ParseConfig: %w", err)
    }

    // Connection counts.
    // Rule of thumb for API servers: max(4, 2 * runtime.NumCPU())
    // Never exceed the PostgreSQL max_connections minus DBA/monitoring headroom.
    cfg.MaxConns = int32(max(4, 2*runtime.NumCPU()))
    cfg.MinIdleConns = 2

    // Lifetime management — prevents long-lived connections accumulating state.
    cfg.MaxConnLifetime = 30 * time.Minute
    cfg.MaxConnLifetimeJitter = 5 * time.Minute   // prevents thundering herd on expiry
    cfg.MaxConnIdleTime = 5 * time.Minute
    cfg.HealthCheckPeriod = time.Minute

    // Called after a new connection is established.
    cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
        // Set session-level settings if needed.
        _, err := conn.Exec(ctx, "SET application_name = 'yourapp'")
        return err
    }

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("NewPool connect: %w", err)
    }

    // Verify the pool is healthy at startup.
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("NewPool ping: %w", err)
    }

    return pool, nil
}
```

### PgBouncer Compatibility

If your deployment uses PgBouncer in transaction-pooling mode, prepared statements are **not** safe because the backend connection may change between statements. Disable automatic statement caching:

```go
cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
// OR per-connection via DSN:
// postgres://...?default_query_exec_mode=exec
```

Do **not** use `QueryExecModeExec` without PgBouncer — you lose prepared statement reuse and performance.

---

## 2. Query Patterns

### Preferred: CollectRows with RowToStructByName

```go
const q = `
    SELECT id, title, body, owner_id, created_at, updated_at
    FROM documents
    WHERE owner_id = $1
    ORDER BY created_at DESC
    LIMIT $2 OFFSET $3
`

rows, err := r.pool.Query(ctx, q, ownerID, limit, offset)
if err != nil {
    return nil, fmt.Errorf("documentRepository.List: %w", err)
}
// CollectRows closes rows and returns the slice.
docs, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Document])
if err != nil {
    return nil, fmt.Errorf("documentRepository.List scan: %w", err)
}
return docs, nil
```

Note: `pgx.RowToStructByName` maps column names to struct fields using the `db` tag:
```go
type Document struct {
    ID        string    `db:"id"`
    Title     string    `db:"title"`
    Body      string    `db:"body"`
    OwnerID   string    `db:"owner_id"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

### Single Row — CollectExactlyOneRow

```go
rows, err := r.pool.Query(ctx, `SELECT ... FROM documents WHERE id = $1`, id)
if err != nil {
    return nil, fmt.Errorf("query: %w", err)
}

doc, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Document])
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.ErrNotFound
    }
    return nil, fmt.Errorf("scan: %w", err)
}
return &doc, nil
```

### Named Arguments

For complex queries with many parameters, named args improve readability and avoid positional mistakes:

```go
const q = `
    INSERT INTO documents (id, title, body, owner_id, created_at, updated_at)
    VALUES (@id, @title, @body, @owner_id, @created_at, @updated_at)
`

_, err := r.pool.Exec(ctx, q, pgx.NamedArgs{
    "id":         doc.ID,
    "title":      doc.Title,
    "body":       doc.Body,
    "owner_id":   doc.OwnerID,
    "created_at": doc.CreatedAt,
    "updated_at": doc.UpdatedAt,
})
```

---

## 3. Transaction Handling

### Pattern A: BeginFunc (Preferred for simple cases)

`pgx.BeginFunc` automatically rolls back on error and commits on success. Use for service-layer transactions.

```go
func (r *documentRepository) CreateWithTags(ctx context.Context, doc *domain.Document, tags []string) error {
    return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
        _, err := tx.Exec(ctx,
            `INSERT INTO documents (id, title, body, owner_id) VALUES ($1, $2, $3, $4)`,
            doc.ID, doc.Title, doc.Body, doc.OwnerID,
        )
        if err != nil {
            return fmt.Errorf("insert document: %w", err)
        }

        for _, tag := range tags {
            _, err = tx.Exec(ctx,
                `INSERT INTO document_tags (document_id, tag) VALUES ($1, $2)`,
                doc.ID, tag,
            )
            if err != nil {
                return fmt.Errorf("insert tag %q: %w", tag, err)
            }
        }
        return nil // commit
    })
}
```

### Pattern B: Explicit Transaction (for service-layer orchestration)

When the transaction must span multiple repository calls, pass the transaction down. Use a `Querier` interface that both `*pgx.Conn`/`*pgxpool.Pool` and `pgx.Tx` satisfy.

```go
// internal/domain/repository.go

// Querier is satisfied by *pgxpool.Pool, *pgx.Conn, and pgx.Tx.
// Repositories that need to be transactional accept this interface.
type Querier interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

```go
// internal/service/document_service.go

func (s *documentService) TransferOwnership(ctx context.Context, docID, newOwnerID string) error {
    tx, err := s.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // safe to call after Commit; no-ops on closed tx

    if err := s.docRepo.WithTx(tx).UpdateOwner(ctx, docID, newOwnerID); err != nil {
        return fmt.Errorf("update owner: %w", err)
    }
    if err := s.auditRepo.WithTx(tx).LogOwnerChange(ctx, docID, newOwnerID); err != nil {
        return fmt.Errorf("audit log: %w", err)
    }

    return tx.Commit(ctx)
}
```

```go
// Repository with transaction support.
type documentRepository struct {
    db domain.Querier
}

func (r *documentRepository) WithTx(tx pgx.Tx) *documentRepository {
    return &documentRepository{db: tx}
}
```

### Isolation Levels

```go
// Serializable for strong consistency (e.g., financial transfers).
tx, err := pool.BeginTx(ctx, pgx.TxOptions{
    IsoLevel: pgx.Serializable,
})

// Read committed is the PostgreSQL default and usually sufficient.
tx, err := pool.BeginTx(ctx, pgx.TxOptions{
    IsoLevel: pgx.ReadCommitted,
})
```

---

## 4. Batch Queries

Use batches when you need to execute multiple independent queries in a single round trip.

```go
func (r *documentRepository) DeleteMany(ctx context.Context, ids []string) error {
    batch := &pgx.Batch{}
    for _, id := range ids {
        batch.Queue(`DELETE FROM documents WHERE id = $1`, id)
    }

    results := r.pool.SendBatch(ctx, batch)
    defer results.Close()

    for i := range ids {
        _, err := results.Exec()
        if err != nil {
            return fmt.Errorf("delete[%d]: %w", i, err)
        }
    }
    return results.Close()
}
```

### Bulk Insert: CopyFrom

`CopyFrom` is orders of magnitude faster than multi-row INSERT for large datasets (even as few as 5 rows).

```go
func (r *documentRepository) BulkInsert(ctx context.Context, docs []*domain.Document) error {
    _, err := r.pool.CopyFrom(
        ctx,
        pgx.Identifier{"documents"},
        []string{"id", "title", "body", "owner_id", "created_at", "updated_at"},
        pgx.CopyFromSlice(len(docs), func(i int) ([]any, error) {
            d := docs[i]
            return []any{d.ID, d.Title, d.Body, d.OwnerID, d.CreatedAt, d.UpdatedAt}, nil
        }),
    )
    return err
}
```

---

## 5. PostgreSQL-Specific Patterns

### JSONB

```sql
-- Schema
ALTER TABLE documents ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';

-- Query with JSONB containment
SELECT * FROM documents WHERE metadata @> '{"status": "published"}';

-- Query with JSONB path
SELECT * FROM documents WHERE metadata->>'author_id' = $1;
```

```go
// In Go — use map[string]any or a typed struct with pgtype.
import "github.com/jackc/pgx/v5/pgtype"

type DocumentMetadata struct {
    Status   string `json:"status"`
    AuthorID string `json:"author_id"`
}

// Read JSONB into a struct.
var meta DocumentMetadata
err := row.Scan(&meta) // pgx v5 can scan JSON directly into a struct
```

For custom JSONB types, implement `pgtype.ValueScanner` and `pgtype.ValueEncoder`, or use `pgx.RowToAddrOfStructByName` with `pgtype.Text` intermediary.

### CTEs (Common Table Expressions)

Use CTEs for complex multi-step queries to keep SQL readable and avoid subquery nesting.

```sql
-- Get documents with their tag counts.
WITH doc_tags AS (
    SELECT document_id, COUNT(*) AS tag_count
    FROM document_tags
    GROUP BY document_id
)
SELECT d.*, COALESCE(dt.tag_count, 0) AS tag_count
FROM documents d
LEFT JOIN doc_tags dt ON d.id = dt.document_id
WHERE d.owner_id = $1
ORDER BY d.created_at DESC;
```

### Advisory Locks

Use advisory locks to prevent concurrent processing of the same resource (e.g., background job deduplication).

```go
// Session-level advisory lock (auto-released on connection close).
func (r *jobRepository) TryLockJob(ctx context.Context, jobID int64) (bool, error) {
    var locked bool
    err := r.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, jobID).Scan(&locked)
    return locked, err
}

func (r *jobRepository) UnlockJob(ctx context.Context, jobID int64) error {
    _, err := r.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, jobID)
    return err
}

// Transaction-level advisory lock (auto-released on transaction commit/rollback).
func (r *jobRepository) TryLockJobTx(ctx context.Context, tx pgx.Tx, jobID int64) (bool, error) {
    var locked bool
    err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, jobID).Scan(&locked)
    return locked, err
}
```

---

## 6. Migration Strategy with golang-migrate

### Directory Layout

```
migrations/
  000001_create_users.up.sql
  000001_create_users.down.sql
  000002_create_documents.up.sql
  000002_create_documents.down.sql
```

Name files with a zero-padded integer prefix, not a timestamp, for readability. Always write the `.down.sql` — even if you never plan to roll back in production, it is essential for test teardown.

### Running Migrations at Startup

```go
// internal/infrastructure/migrate.go

package infrastructure

import (
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn, migrationsPath string) error {
    m, err := migrate.New("file://"+migrationsPath, dsn)
    if err != nil {
        return fmt.Errorf("RunMigrations new: %w", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("RunMigrations up: %w", err)
    }
    return nil
}
```

Use the `pgx/v5` database driver for golang-migrate to stay consistent with the rest of the codebase:
```
github.com/golang-migrate/migrate/v4/database/pgx/v5
```

### Safe Migration Practices

- Run migrations in a separate step before deploying the new binary (blue-green / rolling deploy compatibility).
- Use `IF NOT EXISTS` and `IF EXISTS` in migration SQL to make them idempotent where possible.
- Never modify an existing migration file once it has been run in any environment — create a new migration instead.
- Lock the `schema_migrations` table is handled automatically by golang-migrate.

---

## 7. Production Connection Pool Tuning

### Starting Estimates

| Scenario | MaxConns | MinIdleConns |
|---|---|---|
| Development / local | 5 | 0 |
| Single API server, 4 CPUs | 10–15 | 2 |
| Multiple API replicas | `floor(pg_max_conns / replicas) - 5` | 2 |
| High-throughput (batch processing) | 20–30 | 5 |

The PostgreSQL side: `max_connections` defaults to 100. Reserve ~10 for superuser/DBA connections. Then divide the remainder across all application instances.

```sql
-- Check current connections.
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

-- Check pool waits (a high count means MaxConns is too low).
SELECT * FROM pg_stat_activity WHERE wait_event_type = 'Client';
```

### Observing Pool Health

```go
// Log pool stats periodically from a background goroutine.
func LogPoolStats(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            stats := pool.Stat()
            logger.Info("pgxpool stats",
                "total_conns",       stats.TotalConns(),
                "acquired_conns",    stats.AcquiredConns(),
                "idle_conns",        stats.IdleConns(),
                "constructing_conns", stats.ConstructingConns(),
                "max_conns",         stats.MaxConns(),
            )
        }
    }
}
```

### Parameterized Queries — Always

pgx v5 never interpolates parameters into SQL strings. All query arguments are transmitted as binary protocol parameters to PostgreSQL. This makes SQL injection impossible at the driver level.

**Do this:**
```go
r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
```

**Never this:**
```go
// NEVER — SQL injection vulnerability.
r.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM documents WHERE id = '%s'`, id))
```

### Null Handling

Use `pgtype` types or pointer types for nullable columns.

```go
import "github.com/jackc/pgx/v5/pgtype"

type Document struct {
    ID          string       `db:"id"`
    Description pgtype.Text  `db:"description"` // nullable VARCHAR
    ArchivedAt  pgtype.Timestamptz `db:"archived_at"` // nullable TIMESTAMPTZ
}

// Check if value is present.
if doc.Description.Valid {
    fmt.Println(doc.Description.String)
}
```

Or use Go pointers — pgx v5 scans NULL into a nil pointer:
```go
type Document struct {
    Description *string    `db:"description"`
    ArchivedAt  *time.Time `db:"archived_at"`
}
```
