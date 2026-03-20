# Testing Reference

> Table-driven tests, mocking with mockery, integration tests with testcontainers, benchmarks, and httptest patterns.
> Go 1.22+, testify/mock, testcontainers-go.

---

## 1. Table-Driven Tests — Standard Pattern

Table-driven tests are idiomatic Go. Define test cases as a slice of structs, iterate, and call `t.Run` for a named sub-test per case.

```go
// internal/service/document_service_test.go

package service_test

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "yourapp/internal/domain"
    "yourapp/internal/mocks"
    "yourapp/internal/service"
)

func TestDocumentService_GetDocument(t *testing.T) {
    t.Parallel()

    existingDoc := &domain.Document{
        ID:      "doc-123",
        Title:   "Hello World",
        OwnerID: "user-456",
    }

    tests := []struct {
        name      string
        docID     string
        callerID  string
        setupMock func(repo *mocks.DocumentRepository)
        wantDoc   *domain.Document
        wantErr   error
    }{
        {
            name:     "returns document for owner",
            docID:    "doc-123",
            callerID: "user-456",
            setupMock: func(r *mocks.DocumentRepository) {
                r.On("GetByID", mock.Anything, "doc-123").Return(existingDoc, nil)
            },
            wantDoc: existingDoc,
            wantErr: nil,
        },
        {
            name:     "returns forbidden for non-owner",
            docID:    "doc-123",
            callerID: "other-user",
            setupMock: func(r *mocks.DocumentRepository) {
                r.On("GetByID", mock.Anything, "doc-123").Return(existingDoc, nil)
            },
            wantDoc: nil,
            wantErr: domain.ErrForbidden,
        },
        {
            name:     "returns not found when document missing",
            docID:    "missing-doc",
            callerID: "user-456",
            setupMock: func(r *mocks.DocumentRepository) {
                r.On("GetByID", mock.Anything, "missing-doc").Return(nil, domain.ErrNotFound)
            },
            wantDoc: nil,
            wantErr: domain.ErrNotFound,
        },
    }

    for _, tc := range tests {
        tc := tc // capture range variable (required before Go 1.22)
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            mockRepo := mocks.NewDocumentRepository(t)
            tc.setupMock(mockRepo)

            svc := service.NewDocumentService(mockRepo, slog.Default())
            got, err := svc.GetDocument(context.Background(), tc.docID, tc.callerID)

            if tc.wantErr != nil {
                require.Error(t, err)
                assert.True(t, errors.Is(err, tc.wantErr),
                    "expected error %v, got %v", tc.wantErr, err)
                assert.Nil(t, got)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tc.wantDoc, got)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### Rules for Good Table-Driven Tests

- Name each case with a short, readable description — it appears in `go test -v` output.
- Use `t.Parallel()` at both the outer and inner test level when tests don't share mutable state.
- Use `require` (fatal on failure) for preconditions; use `assert` (non-fatal) for assertions.
- Call `mockRepo.AssertExpectations(t)` at the end to verify all expected calls were made.
- Avoid deeply nested test helpers — they obscure failure context. Put setup in `setupMock` functions inside the test case struct.

---

## 2. Mock Generation with Mockery

Mockery v2 reads your interfaces and generates testify-compatible mocks. Configure it once, then regenerate whenever interfaces change.

### Configuration (.mockery.yaml)

```yaml
# .mockery.yaml at repo root
with-expecter: true      # enables mock.On(...).Return(...) style
dir: "internal/mocks"
outpkg: "mocks"
mockname: "{{.InterfaceName}}"
filename: "{{.InterfaceName}}.go"
packages:
  yourapp/internal/domain:
    interfaces:
      DocumentRepository:
      DocumentService:
      SearchIndex:
```

Run: `go run github.com/vektra/mockery/v2@latest`

### Generated Mock Usage

```go
import (
    "testing"
    "github.com/stretchr/testify/mock"
    "yourapp/internal/mocks"
)

// Constructor injected with t — mock auto-asserts expectations on t.Cleanup.
mockRepo := mocks.NewDocumentRepository(t)

// Expect a call and define the return values.
mockRepo.On("GetByID", mock.Anything, "doc-123").
    Return(&domain.Document{ID: "doc-123"}, nil)

// Expect a call with specific argument matching.
mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *domain.Document) bool {
    return d.Title != ""
})).Return(nil)

// Return different values on successive calls.
mockRepo.On("GetByID", mock.Anything, "doc-456").
    Return(nil, domain.ErrNotFound).Once().
    Return(&domain.Document{}, nil).Once()

// Verify all expected calls were made.
mockRepo.AssertExpectations(t)

// Verify a specific method was not called.
mockRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
```

### With Expecter (Typed, Refactor-Safe)

When `with-expecter: true`, mockery generates a typed `EXPECT()` API:

```go
mockRepo.EXPECT().GetByID(mock.Anything, "doc-123").Return(existingDoc, nil)
mockRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Times(1)
```

This is preferred over string-based `.On("GetByID", ...)` because it fails at compile time if the interface signature changes.

---

## 3. Integration Tests with testcontainers-go

Use the `testcontainers-go/modules/postgres` module for tests that need a real PostgreSQL database. The container starts once per `TestMain` and is shared across tests using snapshots for isolation.

### TestMain Setup

```go
// internal/repository/repository_test.go (or a shared testhelper package)

package repository_test

import (
    "context"
    "log"
    "os"
    "path/filepath"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
    ctx := context.Background()

    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        postgres.WithInitScripts(filepath.Join("..", "..", "migrations")),
        postgres.BasicWaitStrategies(),
    )
    if err != nil {
        log.Fatalf("failed to start postgres container: %v", err)
    }
    defer testcontainers.TerminateContainer(pgContainer)

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        log.Fatalf("failed to get connection string: %v", err)
    }

    testPool, err = pgxpool.New(ctx, connStr)
    if err != nil {
        log.Fatalf("failed to create pool: %v", err)
    }
    defer testPool.Close()

    // Create a baseline snapshot for fast reset between tests.
    if err := pgContainer.Snapshot(ctx, postgres.WithSnapshotName("test_base")); err != nil {
        log.Fatalf("failed to snapshot: %v", err)
    }

    os.Exit(m.Run())
}
```

### Per-Test Isolation via Snapshot Restore

```go
func TestDocumentRepository_Create(t *testing.T) {
    ctx := context.Background()

    // Restore database to clean state before each test.
    if err := pgContainer.Restore(ctx, postgres.WithSnapshotName("test_base")); err != nil {
        t.Fatalf("restore snapshot: %v", err)
    }

    repo := repository.NewDocumentRepository(testPool)

    doc := &domain.Document{
        ID:      "test-doc-1",
        Title:   "Test Document",
        Body:    "Hello, World",
        OwnerID: "user-1",
    }
    err := repo.Create(ctx, doc)
    require.NoError(t, err)

    got, err := repo.GetByID(ctx, "test-doc-1")
    require.NoError(t, err)
    assert.Equal(t, doc.Title, got.Title)
}
```

Snapshot + Restore is significantly faster than truncating tables manually or restarting the container, making it suitable for dozens of integration tests.

### Build Tag Isolation

Guard integration tests with a build tag so they don't run on every `go test ./...`:

```go
//go:build integration

package repository_test
```

Run them explicitly:
```bash
go test -tags=integration ./internal/repository/...
```

---

## 4. HTTP Handler Testing with httptest

Test handlers in isolation using `httptest.NewRecorder` and `echo.NewContext`. No need to start a real HTTP server.

```go
// internal/handler/document_handler_test.go

package handler_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "yourapp/internal/domain"
    "yourapp/internal/handler"
    "yourapp/internal/mocks"
)

func newTestEcho() *echo.Echo {
    e := echo.New()
    e.Validator = handler.NewRequestValidator()
    return e
}

func TestDocumentHandler_Get(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name       string
        docID      string
        userID     string
        setupMock  func(*mocks.DocumentService)
        wantStatus int
    }{
        {
            name:   "returns 200 for valid request",
            docID:  "doc-123",
            userID: "user-456",
            setupMock: func(svc *mocks.DocumentService) {
                svc.EXPECT().GetDocument(mock.Anything, "doc-123", "user-456").
                    Return(&domain.Document{ID: "doc-123", Title: "Hello"}, nil)
            },
            wantStatus: http.StatusOK,
        },
        {
            name:   "returns 404 for missing document",
            docID:  "missing",
            userID: "user-456",
            setupMock: func(svc *mocks.DocumentService) {
                svc.EXPECT().GetDocument(mock.Anything, "missing", "user-456").
                    Return(nil, domain.ErrNotFound)
            },
            wantStatus: http.StatusNotFound,
        },
        {
            name:   "returns 403 for unauthorized access",
            docID:  "doc-123",
            userID: "other-user",
            setupMock: func(svc *mocks.DocumentService) {
                svc.EXPECT().GetDocument(mock.Anything, "doc-123", "other-user").
                    Return(nil, domain.ErrForbidden)
            },
            wantStatus: http.StatusForbidden,
        },
    }

    for _, tc := range tests {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            e := newTestEcho()
            mockSvc := mocks.NewDocumentService(t)
            tc.setupMock(mockSvc)
            h := handler.NewDocumentHandler(mockSvc, slog.Default())

            req := httptest.NewRequest(http.MethodGet, "/", nil)
            rec := httptest.NewRecorder()
            c := e.NewContext(req, rec)
            c.SetParamNames("id")
            c.SetParamValues(tc.docID)
            c.Set("userID", tc.userID)

            err := h.Get(c)
            if err != nil {
                // Echo's HTTPErrorHandler processes the error — simulate it.
                he, ok := err.(*echo.HTTPError)
                require.True(t, ok, "expected HTTPError, got: %v", err)
                assert.Equal(t, tc.wantStatus, he.Code)
            } else {
                assert.Equal(t, tc.wantStatus, rec.Code)
            }
        })
    }
}

func TestDocumentHandler_Create(t *testing.T) {
    t.Parallel()

    e := newTestEcho()
    mockSvc := mocks.NewDocumentService(t)
    mockSvc.EXPECT().CreateDocument(mock.Anything, mock.Anything).
        Return(&domain.Document{ID: "new-doc", Title: "My Doc"}, nil)

    h := handler.NewDocumentHandler(mockSvc, slog.Default())

    body := `{"title":"My Doc","body":"Some content"}`
    req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.Set("userID", "user-123")

    require.NoError(t, h.Create(c))
    assert.Equal(t, http.StatusCreated, rec.Code)

    var resp map[string]any
    require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
    data := resp["data"].(map[string]any)
    assert.Equal(t, "new-doc", data["id"])
}
```

---

## 5. Benchmarks and Profiling

### Writing Benchmarks

```go
// internal/service/document_service_bench_test.go

package service_test

import (
    "context"
    "testing"
)

func BenchmarkDocumentService_GetDocument(b *testing.B) {
    // Setup outside the loop (not counted in timing).
    mockRepo := mocks.NewDocumentRepository(nil) // nil *testing.T OK for benchmarks
    mockRepo.On("GetByID", mock.Anything, mock.Anything).
        Return(&domain.Document{ID: "doc-1"}, nil)
    svc := service.NewDocumentService(mockRepo, slog.Default())
    ctx := context.Background()

    b.ResetTimer()             // Start timing here.
    b.ReportAllocs()           // Show allocations per op.

    for range b.N {
        _, _ = svc.GetDocument(ctx, "doc-1", "user-1")
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem -benchtime=5s ./internal/service/...
```

### Profiling

```bash
# CPU profile
go test -bench=. -cpuprofile=cpu.out ./...
go tool pprof cpu.out

# Memory profile
go test -bench=. -memprofile=mem.out ./...
go tool pprof mem.out

# Interactive web UI
go tool pprof -http=:8080 cpu.out
```

### Profiling a Running HTTP Server

Add the pprof endpoint in development builds only:

```go
//go:build !production

package handler

import (
    "net/http"
    _ "net/http/pprof" // registers /debug/pprof/* handlers
)

func RegisterDebugRoutes(e *echo.Echo) {
    e.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))
}
```

```bash
# Capture 30 seconds of CPU profile from a live server.
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

---

## 6. Test Helpers and Utilities

### Asserting Error Types

```go
// require.ErrorIs for sentinel errors
require.ErrorIs(t, err, domain.ErrNotFound)

// require.ErrorAs for typed errors
var valErr *domain.ValidationError
require.ErrorAs(t, err, &valErr)
assert.Equal(t, "email", valErr.Field)
```

### Test Fixtures

```go
// internal/testhelper/fixtures.go

package testhelper

import "yourapp/internal/domain"

func NewTestDocument(overrides ...func(*domain.Document)) *domain.Document {
    doc := &domain.Document{
        ID:      "test-doc-1",
        Title:   "Test Document",
        Body:    "Test body content",
        OwnerID: "test-user-1",
    }
    for _, fn := range overrides {
        fn(doc)
    }
    return doc
}
```

```go
doc := testhelper.NewTestDocument(func(d *domain.Document) {
    d.OwnerID = "custom-owner"
})
```

### Parallel Test Safety

- Do not share mutable global state between parallel tests.
- If using a real database, each test must restore a clean state (use snapshot/restore or isolated schema).
- Mock objects created with `mocks.NewXxx(t)` are safe to use in parallel tests — each gets its own instance.

---

## 7. Recommended Testing Stack

| Purpose | Package |
|---|---|
| Assertions | `github.com/stretchr/testify/assert` + `require` |
| Mocking | `github.com/vektra/mockery/v2` + `testify/mock` |
| Integration (DB) | `github.com/testcontainers/testcontainers-go/modules/postgres` |
| Goroutine leak detection | `go.uber.org/goleak` |
| HTTP handler testing | `net/http/httptest` (stdlib) |
| Benchmarks | stdlib `testing.B` |
| Fuzzing | stdlib `testing.F` (Go 1.18+) |
