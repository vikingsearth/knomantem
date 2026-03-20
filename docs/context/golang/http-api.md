# HTTP API Reference — Echo v4

> Best practices for Echo v4 HTTP handlers, middleware, validation, and response patterns.
> Current: Echo v4.15.1 (Feb 2026).

---

## 1. Project Layout for Handlers

```
internal/
  handler/
    document_handler.go      ← one file per domain area
    user_handler.go
    routes.go                ← central route registration
    middleware/
      auth.go
      request_id.go
```

Each handler struct holds only its service dependency and logger — no business logic.

```go
// internal/handler/document_handler.go

package handler

import (
    "errors"
    "log/slog"
    "net/http"

    "github.com/labstack/echo/v4"
    "yourapp/internal/domain"
)

type DocumentHandler struct {
    svc    domain.DocumentService
    logger *slog.Logger
}

func NewDocumentHandler(svc domain.DocumentService, logger *slog.Logger) *DocumentHandler {
    return &DocumentHandler{svc: svc, logger: logger}
}

// RegisterRoutes attaches routes to a group — called from routes.go.
func (h *DocumentHandler) RegisterRoutes(g *echo.Group) {
    g.GET("", h.List)
    g.POST("", h.Create)
    g.GET("/:id", h.Get)
    g.PUT("/:id", h.Update)
    g.DELETE("/:id", h.Delete)
}
```

---

## 2. Middleware Ordering

Order matters. Apply middleware in this sequence:

```go
func setupEcho(logger *slog.Logger) *echo.Echo {
    e := echo.New()
    e.HideBanner = true
    e.HidePort = true

    // 1. Pre-router middleware — modifies the request before routing decisions.
    e.Pre(middleware.RemoveTrailingSlash())

    // 2. Recover — must be first global middleware so panics are caught.
    e.Use(middleware.Recover())

    // 3. Request ID — assign before logging so it appears in all log entries.
    e.Use(middleware.RequestID())

    // 4. Structured request logging.
    e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
        LogURI:       true,
        LogStatus:    true,
        LogMethod:    true,
        LogLatency:   true,
        LogRequestID: true,
        LogError:     true,
        HandleError:  true,
        LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
            logger.InfoContext(c.Request().Context(), "request",
                "method",     v.Method,
                "uri",        v.URI,
                "status",     v.Status,
                "latency_ms", v.Latency.Milliseconds(),
                "request_id", v.RequestID,
                "error",      v.Err,
            )
            return nil
        },
    }))

    // 5. CORS — must come before auth so preflight OPTIONS requests pass through.
    e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     []string{"https://yourapp.com"},
        AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
        AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
        AllowCredentials: true,
        MaxAge:           86400,
    }))

    // 6. Rate limiter — protect against abuse before any auth processing.
    e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
        Skipper: middleware.DefaultSkipper,
        Store: middleware.NewRateLimiterMemoryStoreWithConfig(
            middleware.RateLimiterMemoryStoreConfig{
                Rate:      20,              // requests per second
                Burst:     50,
                ExpiresIn: 3 * time.Minute,
            },
        ),
        IdentifierExtractor: func(ctx echo.Context) (string, error) {
            return ctx.RealIP(), nil
        },
        ErrorHandler: func(context echo.Context, err error) error {
            return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
        },
        DenyHandler: func(context echo.Context, identifier string, err error) error {
            return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
        },
    }))

    return e
}
```

### Route Registration

```go
// internal/handler/routes.go

func RegisterRoutes(e *echo.Echo, docHandler *DocumentHandler, userHandler *UserHandler, authMW echo.MiddlewareFunc) {
    // Public routes — no auth required.
    pub := e.Group("/api/v1")
    userHandler.RegisterPublicRoutes(pub)

    // Authenticated routes — JWT middleware applied to the group.
    api := e.Group("/api/v1", authMW)
    docHandler.RegisterRoutes(api.Group("/documents"))
}
```

---

## 3. Request Binding and Validation

### Custom Validator with go-playground/validator

```go
// internal/handler/validator.go

package handler

import (
    "net/http"

    "github.com/go-playground/validator/v10"
    "github.com/labstack/echo/v4"
)

type RequestValidator struct {
    validator *validator.Validate
}

func NewRequestValidator() *RequestValidator {
    v := validator.New(validator.WithRequiredStructEnabled())
    // Register custom tags here.
    _ = v.RegisterValidation("slug", validateSlug)
    return &RequestValidator{validator: v}
}

func (rv *RequestValidator) Validate(i interface{}) error {
    if err := rv.validator.Struct(i); err != nil {
        return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
    }
    return nil
}

// Register on the Echo instance.
// e.Validator = handler.NewRequestValidator()
```

### Binding and Validating in a Handler

```go
type CreateDocumentRequest struct {
    Title string `json:"title" validate:"required,min=1,max=255"`
    Body  string `json:"body"  validate:"required"`
}

func (h *DocumentHandler) Create(c echo.Context) error {
    var req CreateDocumentRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }
    if err := c.Validate(&req); err != nil {
        return err // already wrapped as HTTPError by the validator
    }

    doc, err := h.svc.CreateDocument(c.Request().Context(), domain.CreateDocumentRequest{
        Title:   req.Title,
        Body:    req.Body,
        OwnerID: mustGetUserID(c),
    })
    if err != nil {
        return h.mapError(c, err)
    }
    return c.JSON(http.StatusCreated, NewDocumentResponse(doc))
}
```

**Do not** pass request structs directly to the service layer. Define separate request types per layer to avoid coupling the HTTP interface to the domain model.

### Path Parameters with Type Safety (Echo v4.15+)

```go
// Typed path params — returns an error if not convertible.
id, err := echo.PathParam[string](c, "id")
if err != nil {
    return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
}

// Query params with defaults.
page, _ := echo.QueryParamOr[int](c, "page", 1)
limit, _ := echo.QueryParamOr[int](c, "limit", 20)
if limit > 100 {
    limit = 100
}
```

---

## 4. Response Envelope Pattern

Use a consistent envelope for all API responses. This makes it easy for clients to handle errors and success uniformly.

```go
// internal/handler/response.go

package handler

import "net/http"

// Envelope wraps all successful responses.
type Envelope[T any] struct {
    Data T      `json:"data"`
    Meta *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    Total  int `json:"total,omitempty"`
    Page   int `json:"page,omitempty"`
    Limit  int `json:"limit,omitempty"`
}

// ErrorEnvelope is returned on all errors.
type ErrorEnvelope struct {
    Error   string            `json:"error"`
    Code    string            `json:"code,omitempty"`
    Details map[string]string `json:"details,omitempty"`
}

func JSON[T any](c echo.Context, status int, data T) error {
    return c.JSON(status, Envelope[T]{Data: data})
}

func JSONList[T any](c echo.Context, data []T, total, page, limit int) error {
    return c.JSON(http.StatusOK, Envelope[[]T]{
        Data: data,
        Meta: &Meta{Total: total, Page: page, Limit: limit},
    })
}
```

**Example response shapes:**

```json
// Success
{
  "data": { "id": "abc123", "title": "My Document" }
}

// List
{
  "data": [ ... ],
  "meta": { "total": 42, "page": 1, "limit": 20 }
}
```

---

## 5. Centralised Error Mapping

Define a helper on each handler that maps domain errors to HTTP errors. This keeps the mapping logic in one place.

```go
// internal/handler/errors.go

package handler

import (
    "errors"
    "log/slog"
    "net/http"

    "github.com/labstack/echo/v4"
    "yourapp/internal/domain"
)

// Global custom error handler — register as e.HTTPErrorHandler.
func CustomHTTPErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
    return func(err error, c echo.Context) {
        if c.Response().Committed {
            return
        }

        var code int
        var message string

        var he *echo.HTTPError
        if errors.As(err, &he) {
            code = he.Code
            if msg, ok := he.Message.(string); ok {
                message = msg
            } else {
                message = http.StatusText(code)
            }
        } else {
            // Unmapped error — log it, return 500.
            logger.ErrorContext(c.Request().Context(), "unhandled error", "error", err,
                "method", c.Request().Method,
                "path",   c.Path(),
            )
            code = http.StatusInternalServerError
            message = "internal server error"
        }

        _ = c.JSON(code, ErrorEnvelope{Error: message})
    }
}

// mapError translates domain errors to Echo HTTP errors.
// Call from handlers instead of duplicating switch statements.
func (h *DocumentHandler) mapError(c echo.Context, err error) error {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return echo.NewHTTPError(http.StatusNotFound, "not found")
    case errors.Is(err, domain.ErrForbidden):
        return echo.NewHTTPError(http.StatusForbidden, "forbidden")
    case errors.Is(err, domain.ErrConflict):
        return echo.NewHTTPError(http.StatusConflict, "conflict")
    case errors.Is(err, domain.ErrInvalidArgument):
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    default:
        h.logger.ErrorContext(c.Request().Context(), "service error", "error", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
    }
}
```

Register the global handler in main:
```go
e.HTTPErrorHandler = handler.CustomHTTPErrorHandler(logger)
```

---

## 6. Authentication Middleware (JWT)

```go
// internal/middleware/auth.go

package middleware

import (
    "errors"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/labstack/echo/v4"
)

type Claims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func JWTAuth(secret []byte) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
            if !strings.HasPrefix(authHeader, "Bearer ") {
                return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
            }
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")

            claims := &Claims{}
            token, err := jwt.ParseWithClaims(tokenString, claims,
                func(t *jwt.Token) (interface{}, error) {
                    return secret, nil
                },
                jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
                jwt.WithExpirationRequired(),
                jwt.WithIssuedAt(),
            )
            if err != nil || !token.Valid {
                return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
            }

            // Store claims in context for downstream handlers.
            c.Set("claims", claims)
            c.Set("userID", claims.UserID)
            return next(c)
        }
    }
}

// Helper used in handlers — panics only if middleware is missing (programmer error).
func mustGetUserID(c echo.Context) string {
    id, ok := c.Get("userID").(string)
    if !ok || id == "" {
        panic("userID not set in context — is JWTAuth middleware applied?")
    }
    return id
}
```

---

## 7. Graceful Shutdown

```go
// cmd/server/main.go

import (
    "context"
    "os/signal"
    "syscall"
    "time"
)

func run(e *echo.Echo, logger *slog.Logger) error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    errCh := make(chan error, 1)
    go func() {
        if err := e.Start(":8080"); err != nil {
            errCh <- err
        }
    }()

    select {
    case <-ctx.Done():
        logger.Info("shutdown signal received")
    case err := <-errCh:
        return err
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    return e.Shutdown(shutdownCtx)
}
```

---

## 8. OpenAPI Documentation

For Echo, two pragmatic approaches exist:

### Option A: swaggo/swag (annotation-based)

```go
// @Summary     Get a document
// @Tags        documents
// @Produce     json
// @Param       id  path  string  true  "Document ID"
// @Success     200  {object}  handler.Envelope[domain.Document]
// @Failure     404  {object}  handler.ErrorEnvelope
// @Router      /documents/{id} [get]
// @Security    BearerAuth
func (h *DocumentHandler) Get(c echo.Context) error { ... }
```

Run `swag init` to generate `docs/swagger.json`. Then serve with `swaggo/echo-swagger`.

### Option B: libopenapi + manual spec

Write a `docs/openapi.yaml` manually (or code-first with a spec builder) and validate it with `pb33f/libopenapi`. This gives full control over the spec without relying on comment parsing.

**Recommendation:** Use swaggo/swag for teams that prefer staying close to the handler code. Use a hand-crafted spec for complex APIs or when the spec is treated as the source of truth.

---

## 9. Testing Handlers

See [testing.md](./testing.md) for full httptest patterns. Quick reference:

```go
func TestDocumentHandler_Get(t *testing.T) {
    e := echo.New()
    e.Validator = handler.NewRequestValidator()

    mockSvc := mocks.NewDocumentService(t)
    mockSvc.On("GetDocument", mock.Anything, "doc-123", "user-456").
        Return(&domain.Document{ID: "doc-123", Title: "Hello"}, nil)

    h := handler.NewDocumentHandler(mockSvc, slog.Default())

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("doc-123")
    c.Set("userID", "user-456")

    require.NoError(t, h.Get(c))
    assert.Equal(t, http.StatusOK, rec.Code)
}
```
