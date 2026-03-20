# Go Architecture Reference

> Clean architecture patterns, dependency injection, error handling, and context propagation for this codebase.
> Stack: Go 1.22+, Echo v4, pgx v5, Casbin, golang-jwt v5, slog.

---

## 1. Clean Architecture Layer Boundaries

This codebase follows a strict layered dependency rule: outer layers depend on inner layers, never the reverse.

```
cmd/
  server/main.go          ← wires everything together
internal/
  handler/                ← HTTP layer (Echo handlers)
  service/                ← Business logic
  domain/                 ← Entities + repository interfaces (no imports of infra)
  repository/             ← Concrete DB implementations (pgx)
  middleware/             ← Echo middleware (auth, logging)
  infrastructure/         ← DB pool, external clients
```

**The cardinal rule:** `domain/` must not import `repository/`, `handler/`, or any infrastructure package. It owns the interfaces; everything else implements them.

### Domain Layer

```go
// internal/domain/document.go

package domain

import (
    "context"
    "time"
)

// Document is a pure domain entity — no DB tags, no JSON tags required here.
type Document struct {
    ID        string
    Title     string
    Body      string
    OwnerID   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// DocumentRepository is defined in the domain layer.
// Concrete implementations live in internal/repository/.
type DocumentRepository interface {
    GetByID(ctx context.Context, id string) (*Document, error)
    List(ctx context.Context, ownerID string, opts ListOptions) ([]*Document, error)
    Create(ctx context.Context, doc *Document) error
    Update(ctx context.Context, doc *Document) error
    Delete(ctx context.Context, id string) error
}

// DocumentService is the service interface — used by handlers, tested with mocks.
type DocumentService interface {
    GetDocument(ctx context.Context, id, callerID string) (*Document, error)
    CreateDocument(ctx context.Context, req CreateDocumentRequest) (*Document, error)
    DeleteDocument(ctx context.Context, id, callerID string) error
}

type ListOptions struct {
    Limit  int
    Offset int
}

type CreateDocumentRequest struct {
    Title   string
    Body    string
    OwnerID string
}
```

### Service Layer

```go
// internal/service/document_service.go

package service

import (
    "context"
    "fmt"
    "log/slog"

    "yourapp/internal/domain"
)

type documentService struct {
    repo   domain.DocumentRepository
    logger *slog.Logger
}

// NewDocumentService is the constructor — takes the interface, not the concrete type.
func NewDocumentService(repo domain.DocumentRepository, logger *slog.Logger) domain.DocumentService {
    return &documentService{repo: repo, logger: logger}
}

func (s *documentService) GetDocument(ctx context.Context, id, callerID string) (*domain.Document, error) {
    doc, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("documentService.GetDocument: %w", err)
    }
    if doc.OwnerID != callerID {
        return nil, domain.ErrForbidden
    }
    return doc, nil
}
```

### Repository Layer

```go
// internal/repository/document_postgres.go

package repository

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "yourapp/internal/domain"
)

type documentRepository struct {
    pool *pgxpool.Pool
}

func NewDocumentRepository(pool *pgxpool.Pool) domain.DocumentRepository {
    return &documentRepository{pool: pool}
}

func (r *documentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
    const q = `SELECT id, title, body, owner_id, created_at, updated_at FROM documents WHERE id = $1`

    rows, err := r.pool.Query(ctx, q, id)
    if err != nil {
        return nil, fmt.Errorf("documentRepository.GetByID query: %w", err)
    }

    doc, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Document])
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrNotFound
        }
        return nil, fmt.Errorf("documentRepository.GetByID scan: %w", err)
    }
    return &doc, nil
}
```

---

## 2. Dependency Injection

### Manual Constructor DI (Recommended for this codebase)

Manual DI via constructors is the simplest, most testable, and most readable approach. It requires no code generation and makes the dependency graph explicit.

```go
// cmd/server/main.go

package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "yourapp/internal/domain"
    "yourapp/internal/handler"
    "yourapp/internal/repository"
    "yourapp/internal/service"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
    if err != nil {
        logger.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer pool.Close()

    // Wire up the dependency graph bottom-up.
    docRepo    := repository.NewDocumentRepository(pool)
    docService := service.NewDocumentService(docRepo, logger)
    docHandler := handler.NewDocumentHandler(docService, logger)

    // Register routes.
    e := setupEcho(logger)
    docHandler.RegisterRoutes(e)

    if err := e.Start(":8080"); err != nil {
        logger.Error("server stopped", "error", err)
    }
}
```

### When to Use Wire or Fx

- **Wire (google/wire):** Use when the dependency graph becomes large (20+ types) and the manual wiring in main.go becomes unwieldy. Wire generates the wiring code at compile time — no reflection overhead.
- **Fx (uber-go/fx):** Use when you need lifecycle management, module grouping, or dynamic provider registration. Adds complexity; not recommended unless you need its features.

**Do this** — keep constructors accepting interfaces, not concrete types:
```go
// Good: accepts the interface
func NewDocumentService(repo domain.DocumentRepository, ...) domain.DocumentService
```

**Not this** — accepting concrete types couples layers:
```go
// Bad: couples service to repository implementation
func NewDocumentService(repo *repository.DocumentRepository, ...) *DocumentService
```

---

## 3. Error Handling

### Sentinel Errors in the Domain Layer

Define sentinel errors in the domain package so all layers can check them without creating circular imports.

```go
// internal/domain/errors.go

package domain

import "errors"

var (
    ErrNotFound        = errors.New("not found")
    ErrForbidden       = errors.New("forbidden")
    ErrConflict        = errors.New("conflict")
    ErrInvalidArgument = errors.New("invalid argument")
    ErrUnauthorized    = errors.New("unauthorized")
)
```

### Wrapping Errors Across Layers

Always wrap with `fmt.Errorf("context: %w", err)` to preserve the chain. Use `%w` (not `%v`) when the caller needs to inspect the error. Use `%v` when you deliberately want to hide internal details from the public API surface.

```go
// Repository wraps the DB error with context.
func (r *documentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
    doc, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.Document])
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrNotFound    // translate DB-specific to domain sentinel
        }
        return nil, fmt.Errorf("documentRepository.GetByID: %w", err)
    }
    return &doc, nil
}

// Service wraps and re-wraps.
func (s *documentService) GetDocument(ctx context.Context, id, callerID string) (*domain.Document, error) {
    doc, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("documentService.GetDocument: %w", err)
    }
    // ...
}

// Handler checks domain sentinels using errors.Is.
func (h *documentHandler) Get(c echo.Context) error {
    doc, err := h.svc.GetDocument(c.Request().Context(), id, callerID)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrNotFound):
            return echo.NewHTTPError(http.StatusNotFound, "document not found")
        case errors.Is(err, domain.ErrForbidden):
            return echo.NewHTTPError(http.StatusForbidden, "access denied")
        default:
            h.logger.ErrorContext(c.Request().Context(), "unexpected error", "error", err)
            return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
        }
    }
    return c.JSON(http.StatusOK, doc)
}
```

### errors.Is vs errors.As

Use `errors.Is` for sentinel comparisons. Use `errors.As` when you need to extract data from a typed error.

```go
// errors.Is — checks identity/equality through the chain
if errors.Is(err, domain.ErrNotFound) { ... }

// errors.As — extracts a specific error type from the chain
var validErr *domain.ValidationError
if errors.As(err, &validErr) {
    // validErr.Field, validErr.Message are accessible
}
```

### Custom Error Type with Field Data

```go
// internal/domain/errors.go

type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: field %q — %s", e.Field, e.Message)
}

// Usage
return nil, &domain.ValidationError{Field: "email", Message: "must be a valid email address"}
```

### Do Not Panic in Business Logic

Only use `panic` for programmer errors (e.g., nil pointer passed where required). Recover in middleware. Never panic in service or repository code.

---

## 4. Context Propagation and Cancellation

### The Rules

1. Always pass `ctx context.Context` as the **first argument** to any function that does I/O.
2. Never store a context in a struct field — always thread it through function calls.
3. Always `defer cancel()` immediately after creating a cancellable context.
4. Respect cancellation: check `ctx.Err()` in tight loops.

```go
// Good: context flows through every layer
func (h *documentHandler) List(c echo.Context) error {
    ctx := c.Request().Context()   // ← provided by Echo from the HTTP request
    docs, err := h.svc.ListDocuments(ctx, ownerID, opts)
    // ...
}

func (s *documentService) ListDocuments(ctx context.Context, ownerID string, opts domain.ListOptions) ([]*domain.Document, error) {
    return s.repo.List(ctx, ownerID, opts)
}

func (r *documentRepository) List(ctx context.Context, ownerID string, opts domain.ListOptions) ([]*domain.Document, error) {
    rows, err := r.pool.Query(ctx, `SELECT ... FROM documents WHERE owner_id = $1`, ownerID)
    // ...
}
```

### Timeout on External Calls

Wrap external calls with a timeout context to prevent goroutine leaks.

```go
func (s *documentService) FetchExternalMetadata(ctx context.Context, docID string) (*Metadata, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return s.externalClient.Fetch(ctx, docID)
}
```

### Background Workers with Context

Pass a root context from `main` into workers so they shut down cleanly.

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    worker := NewIndexWorker(pool, searchIndex, logger)
    go worker.Run(ctx)

    // ... start HTTP server ...
    <-ctx.Done()
    logger.Info("shutdown signal received")
}

// Worker respects context cancellation.
func (w *IndexWorker) Run(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            w.logger.Info("index worker stopping")
            return
        case <-ticker.C:
            if err := w.processQueue(ctx); err != nil {
                w.logger.Error("index worker error", "error", err)
            }
        }
    }
}
```

### Context Values — Use Sparingly

Only use `context.WithValue` for truly request-scoped, cross-cutting data (trace IDs, authenticated user). Never use it as a back-door parameter passing mechanism.

```go
// Define a private key type to avoid collision.
type contextKey string

const (
    contextKeyUserID   contextKey = "userID"
    contextKeyTraceID  contextKey = "traceID"
)

// Set in auth middleware.
ctx = context.WithValue(ctx, contextKeyUserID, userID)

// Read in handler or service — always assert safely.
userID, ok := ctx.Value(contextKeyUserID).(string)
if !ok {
    return domain.ErrUnauthorized
}
```

---

## 5. slog Structured Logging Conventions

```go
// Initialize once in main, pass as a dependency.
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    // AddSource: true, // add file:line — useful in production
}))
slog.SetDefault(logger) // optional global default

// Use With() to create child loggers with fixed fields.
requestLogger := logger.With(
    "request_id", requestID,
    "user_id", userID,
)

// Always use context-aware variants in request handlers.
logger.InfoContext(ctx, "document created", "document_id", doc.ID)
logger.ErrorContext(ctx, "failed to create document", "error", err, "user_id", userID)

// Prefer slog.Attr for performance-critical paths.
slog.LogAttrs(ctx, slog.LevelInfo, "batch processed",
    slog.Int("count", n),
    slog.Duration("elapsed", elapsed),
)
```

**Levels:**
- `Debug` — fine-grained tracing, disabled in production.
- `Info` — normal operational events (request handled, job started).
- `Warn` — recoverable anomalies (retried, degraded path taken).
- `Error` — failures requiring attention; always include `"error", err`.
