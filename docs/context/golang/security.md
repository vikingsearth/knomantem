# Security Reference

> JWT best practices, password hashing, SQL injection prevention, rate limiting, and Casbin RBAC patterns.
> Stack: golang-jwt/jwt v5, bcrypt/argon2id, pgx v5, Echo v4, Casbin v2.

---

## 1. JWT Best Practices

### Algorithm Selection

| Algorithm | Key Type | Use Case |
|---|---|---|
| HS256 | Shared secret ([]byte) | Single-service, secret stored in env |
| RS256 | RSA private/public key pair | Multi-service; public key distributed to consumers |
| ES256 | ECDSA P-256 key pair | Smaller signatures than RS256; preferred for new systems |
| EdDSA | Ed25519 key pair | Fastest, smallest; use when all consumers support it |

**Recommendation for this codebase:** Use RS256 or ES256 if tokens are verified by multiple services. Use HS256 only when the signing key never leaves the single service that creates and verifies tokens.

**Never use** the `none` algorithm. The jwt/v5 library requires an explicit unsafe constant to accept it — never pass that constant in production code.

### Token Creation

```go
// internal/service/auth_service.go

import (
    "crypto/rand"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func (s *authService) IssueAccessToken(userID, role string) (string, error) {
    now := time.Now()
    claims := Claims{
        UserID: userID,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "yourapp",
            Subject:   userID,
            Audience:  jwt.ClaimStrings{"yourapp-api"},
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)), // short-lived
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.signingKey)
}
```

**Access token lifetime:** 15–30 minutes. Short lifetimes limit the blast radius of a stolen token.

### Token Validation

Always specify `WithValidMethods` to prevent algorithm substitution attacks. If an attacker crafts a token with `alg: none` or switches from RS256 to HS256 (using the public key as the HMAC secret), a naive verifier will accept it.

```go
func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
    claims := &Claims{}
    token, err := jwt.ParseWithClaims(
        tokenString,
        claims,
        func(t *jwt.Token) (any, error) {
            return s.signingKey, nil
        },
        jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
        jwt.WithExpirationRequired(),
        jwt.WithIssuedAt(),
        jwt.WithIssuer("yourapp"),
        jwt.WithAudience("yourapp-api"),
    )
    if err != nil || !token.Valid {
        return nil, fmt.Errorf("invalid token: %w", err)
    }
    return claims, nil
}
```

### Refresh Token Rotation

Refresh tokens are long-lived (7–30 days) but must be rotated on every use to limit reuse. Store refresh tokens in the database — never in a JWT — so they can be revoked.

```go
// Schema
// CREATE TABLE refresh_tokens (
//     id          TEXT PRIMARY KEY,
//     user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
//     token_hash  TEXT NOT NULL UNIQUE,  -- bcrypt hash of the token
//     expires_at  TIMESTAMPTZ NOT NULL,
//     used_at     TIMESTAMPTZ,           -- NULL = not yet used
//     revoked     BOOLEAN NOT NULL DEFAULT FALSE,
//     created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
// );

func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error) {
    // 1. Hash the incoming token and look it up.
    tokenHash := hashToken(refreshToken)
    stored, err := s.tokenRepo.GetByHash(ctx, tokenHash)
    if err != nil {
        return "", "", domain.ErrUnauthorized
    }

    // 2. Check it hasn't been used or revoked (detect token reuse attack).
    if stored.UsedAt != nil || stored.Revoked {
        // Possible token theft — revoke all tokens for this user.
        _ = s.tokenRepo.RevokeAllForUser(ctx, stored.UserID)
        return "", "", domain.ErrUnauthorized
    }

    // 3. Check expiry.
    if time.Now().After(stored.ExpiresAt) {
        return "", "", domain.ErrUnauthorized
    }

    // 4. Mark old token as used (rotation — old token becomes invalid immediately).
    if err := s.tokenRepo.MarkUsed(ctx, stored.ID); err != nil {
        return "", "", fmt.Errorf("mark used: %w", err)
    }

    // 5. Issue new access token + new refresh token.
    accessToken, err = s.IssueAccessToken(stored.UserID, stored.Role)
    if err != nil {
        return "", "", fmt.Errorf("issue access token: %w", err)
    }
    newRefreshToken, err = s.IssueRefreshToken(ctx, stored.UserID, stored.Role)
    if err != nil {
        return "", "", fmt.Errorf("issue refresh token: %w", err)
    }

    return accessToken, newRefreshToken, nil
}

func hashToken(raw string) string {
    h := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(h[:])
}

func generateRefreshToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

### Signing Key Management

```go
// Load from environment — never hardcode.
signingKey := []byte(os.Getenv("JWT_SECRET"))
if len(signingKey) < 32 {
    log.Fatal("JWT_SECRET must be at least 32 bytes")
}

// For RS256 — load PEM from file or secret manager.
keyBytes, _ := os.ReadFile(os.Getenv("JWT_PRIVATE_KEY_PATH"))
privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
```

---

## 2. Password Hashing

### bcrypt vs argon2id — Which to Use

| | bcrypt | argon2id |
|---|---|---|
| Go stdlib support | `golang.org/x/crypto/bcrypt` | `golang.org/x/crypto/argon2` |
| Side-channel resistance | Partial | Yes (Argon2id combines i and d) |
| Memory hardness | No (CPU-only) | Yes (configurable) |
| OWASP recommended 2024 | Yes (cost ≥ 10) | Yes (preferred for new systems) |
| Max password length | 72 bytes | Unlimited |

**Recommendation:** Use **argon2id** for new systems. Use **bcrypt** if you need compatibility with existing hashes or if memory constraints are a concern.

### bcrypt Implementation

```go
import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12 // OWASP recommends ≥ 10; tune so hashing takes ~100–300ms

func HashPassword(password string) (string, error) {
    if len(password) > 72 {
        return "", errors.New("password exceeds maximum length of 72 characters")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    if err != nil {
        return "", fmt.Errorf("HashPassword: %w", err)
    }
    return string(hash), nil
}

func CheckPassword(hash, password string) error {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
        return domain.ErrUnauthorized
    }
    return err
}

// Upgrade cost factor if hash was created with an old cost.
func NeedsRehash(hash string) bool {
    cost, err := bcrypt.Cost([]byte(hash))
    if err != nil {
        return true
    }
    return cost < bcryptCost
}
```

### argon2id Implementation

```go
import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "fmt"
    "strings"

    "golang.org/x/crypto/argon2"
)

type Argon2idParams struct {
    Memory      uint32 // KiB
    Iterations  uint32
    Parallelism uint8
    SaltLength  uint32
    KeyLength   uint32
}

// RFC 9106 recommended parameters (tune so hashing takes ~100–300ms on your hardware).
var DefaultArgon2idParams = Argon2idParams{
    Memory:      64 * 1024, // 64 MiB
    Iterations:  1,
    Parallelism: 4,
    SaltLength:  16,
    KeyLength:   32,
}

func HashPasswordArgon2id(password string, p Argon2idParams) (string, error) {
    salt := make([]byte, p.SaltLength)
    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("generate salt: %w", err)
    }

    hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

    // Encode as $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
    encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version,
        p.Memory, p.Iterations, p.Parallelism,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    )
    return encoded, nil
}

func CheckPasswordArgon2id(encoded, password string) error {
    p, salt, hash, err := decodeArgon2idHash(encoded)
    if err != nil {
        return fmt.Errorf("decode hash: %w", err)
    }

    otherHash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

    // Use constant-time comparison to prevent timing attacks.
    if subtle.ConstantTimeCompare(hash, otherHash) != 1 {
        return domain.ErrUnauthorized
    }
    return nil
}
```

---

## 3. SQL Injection Prevention

pgx v5 **never** interpolates parameters into SQL strings. All query arguments are sent as separate protocol-level parameters to PostgreSQL. This makes SQL injection structurally impossible when using the pgx API correctly.

**Do this — always use parameterised queries:**
```go
rows, err := pool.Query(ctx,
    `SELECT * FROM documents WHERE owner_id = $1 AND status = $2`,
    ownerID, status,
)
```

**Never do this — string interpolation:**
```go
// CRITICAL SECURITY VULNERABILITY — never do this.
query := fmt.Sprintf(`SELECT * FROM documents WHERE owner_id = '%s'`, ownerID)
pool.Query(ctx, query)
```

**Named args are equally safe:**
```go
pool.Exec(ctx,
    `UPDATE documents SET title = @title WHERE id = @id`,
    pgx.NamedArgs{"title": newTitle, "id": docID},
)
```

**Dynamic ORDER BY requires whitelisting — never interpolate user input:**
```go
// Safe: validate against an allowed set before interpolating.
validColumns := map[string]bool{"title": true, "created_at": true, "updated_at": true}
if !validColumns[sortColumn] {
    return domain.ErrInvalidArgument
}
// sortColumn is now safe to interpolate because it came from a controlled set.
query := fmt.Sprintf(`SELECT * FROM documents ORDER BY %s %s LIMIT $1`, sortColumn, sortDir)
```

---

## 4. Rate Limiting Patterns

### In-Process Rate Limiter (Echo Built-in)

Suitable for single-instance deployments or when per-process limiting is acceptable.

```go
e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
    Store: middleware.NewRateLimiterMemoryStoreWithConfig(
        middleware.RateLimiterMemoryStoreConfig{
            Rate:      10,             // 10 requests per second per client
            Burst:     30,             // allow bursts up to 30
            ExpiresIn: 3 * time.Minute,
        },
    ),
    IdentifierExtractor: func(c echo.Context) (string, error) {
        // Rate limit by authenticated user ID if available; else by IP.
        if userID, ok := c.Get("userID").(string); ok && userID != "" {
            return userID, nil
        }
        return c.RealIP(), nil
    },
    DenyHandler: func(c echo.Context, identifier string, err error) error {
        return c.JSON(http.StatusTooManyRequests, ErrorEnvelope{
            Error: "rate limit exceeded",
            Code:  "RATE_LIMITED",
        })
    },
}))
```

### Distributed Rate Limiting (Redis — for multi-instance deployments)

For multiple API instances behind a load balancer, use Redis-backed rate limiting:

```go
import "github.com/redis/go-redis/v9"

// Use a sliding window algorithm via Redis sorted sets.
// Third-party package: github.com/go-redis/redis_rate/v10

import "github.com/go-redis/redis_rate/v10"

limiter := redis_rate.NewLimiter(redisClient)

func RateLimitMiddleware(limiter *redis_rate.Limiter) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            key := c.RealIP()
            res, err := limiter.Allow(c.Request().Context(), key, redis_rate.PerMinute(60))
            if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "rate limiter error")
            }
            c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(res.Limit.Rate))
            c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
            if res.Allowed == 0 {
                return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
            }
            return next(c)
        }
    }
}
```

### Per-Endpoint Rate Limits

Apply stricter limits on sensitive endpoints (login, token refresh, password reset):

```go
authGroup := e.Group("/api/v1/auth")
authGroup.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
    Store: middleware.NewRateLimiterMemoryStoreWithConfig(
        middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 10},
    ),
    IdentifierExtractor: func(c echo.Context) (string, error) {
        return c.RealIP(), nil // always use IP for auth endpoints
    },
}))
authGroup.POST("/login", authHandler.Login)
authGroup.POST("/refresh", authHandler.Refresh)
```

---

## 5. Casbin RBAC — Policy Design

### Model Configuration

```ini
# config/rbac_model.conf

[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
```

`keyMatch2` supports path patterns like `/documents/:id` — appropriate for REST APIs.

### Policy CSV (or PostgreSQL adapter)

```csv
# Roles
g, alice, admin
g, bob, editor
g, carol, viewer

# Admin can do anything on documents.
p, admin, /documents, GET
p, admin, /documents, POST
p, admin, /documents/:id, GET
p, admin, /documents/:id, PUT
p, admin, /documents/:id, DELETE

# Editor can read and write but not delete.
p, editor, /documents, GET
p, editor, /documents, POST
p, editor, /documents/:id, GET
p, editor, /documents/:id, PUT

# Viewer can only read.
p, viewer, /documents, GET
p, viewer, /documents/:id, GET
```

### PostgreSQL Adapter

Store policies in PostgreSQL so they can be edited at runtime without redeployment.

```bash
go get github.com/casbin/casbin-pg-adapter
```

```go
import (
    "github.com/casbin/casbin/v2"
    pgadapter "github.com/casbin/casbin-pg-adapter"
)

func NewEnforcer(dsn, modelPath string) (*casbin.Enforcer, error) {
    adapter, err := pgadapter.NewAdapter(dsn)
    if err != nil {
        return nil, fmt.Errorf("casbin pg adapter: %w", err)
    }

    enforcer, err := casbin.NewEnforcer(modelPath, adapter)
    if err != nil {
        return nil, fmt.Errorf("casbin enforcer: %w", err)
    }

    // Load policies from DB.
    if err := enforcer.LoadPolicy(); err != nil {
        return nil, fmt.Errorf("load policy: %w", err)
    }

    return enforcer, nil
}
```

### Casbin Middleware for Echo

```go
// internal/middleware/casbin.go

package middleware

import (
    "net/http"

    "github.com/casbin/casbin/v2"
    "github.com/labstack/echo/v4"
)

func CasbinRBAC(enforcer *casbin.Enforcer) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userID, ok := c.Get("userID").(string)
            if !ok || userID == "" {
                return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
            }

            obj := c.Path()  // registered path, e.g. /documents/:id
            act := c.Request().Method

            allowed, err := enforcer.Enforce(userID, obj, act)
            if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "authorization error")
            }
            if !allowed {
                return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
            }

            return next(c)
        }
    }
}
```

Apply after JWT auth so `userID` is available:
```go
api := e.Group("/api/v1", jwtMiddleware, casbinMiddleware)
```

### Programmatic Policy Management

```go
type PolicyService struct {
    enforcer *casbin.Enforcer
}

// AssignRole adds a user to a role.
func (s *PolicyService) AssignRole(ctx context.Context, userID, role string) error {
    ok, err := s.enforcer.AddRoleForUser(userID, role)
    if err != nil {
        return fmt.Errorf("AssignRole: %w", err)
    }
    if !ok {
        return nil // already assigned
    }
    return s.enforcer.SavePolicy()
}

// RevokeRole removes a user from a role.
func (s *PolicyService) RevokeRole(ctx context.Context, userID, role string) error {
    ok, err := s.enforcer.DeleteRoleForUser(userID, role)
    if err != nil {
        return fmt.Errorf("RevokeRole: %w", err)
    }
    if !ok {
        return domain.ErrNotFound
    }
    return s.enforcer.SavePolicy()
}

// GetRoles returns all roles for a user (including inherited).
func (s *PolicyService) GetRoles(ctx context.Context, userID string) ([]string, error) {
    return s.enforcer.GetImplicitRolesForUser(userID)
}
```

### SyncedEnforcer for Multi-Instance Deployments

If multiple API instances run concurrently, use `SyncedEnforcer` to reload policies automatically:

```go
enforcer, err := casbin.NewSyncedEnforcer(modelPath, adapter)
enforcer.StartAutoLoadPolicy(1 * time.Minute) // reload every minute
```

Or use the casbin watcher (e.g., Redis watcher) for immediate propagation when policies change:

```go
import rediswatcher "github.com/casbin/redis-watcher/v2"

watcher, err := rediswatcher.NewWatcher("redis:6379", rediswatcher.WatcherOptions{})
enforcer.SetWatcher(watcher)
watcher.SetUpdateCallback(func(msg string) {
    enforcer.LoadPolicy()
})
```

---

## 6. Security Headers

Add security headers via Echo middleware. Apply globally before any route-specific logic.

```go
e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
    XSSProtection:         "1; mode=block",
    ContentTypeNosniff:    "nosniff",
    XFrameOptions:         "DENY",
    HSTSMaxAge:            31536000, // 1 year
    HSTSExcludeSubdomains: false,
    ContentSecurityPolicy: "default-src 'self'",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
}))
```

---

## 7. Secrets Management

- Load all secrets (JWT key, DB password, API keys) from environment variables or a secrets manager (Vault, AWS SSM, GCP Secret Manager) — never from source code or config files committed to git.
- Validate that required secrets are present at startup and fail fast if they are missing.
- For JWT signing keys: rotate periodically. Support two valid keys during the rotation window (old key for validation, new key for signing).

```go
func mustGetEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("required environment variable %q is not set", key)
    }
    return v
}
```
