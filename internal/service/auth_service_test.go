package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
	mw "github.com/knomantem/knomantem/internal/middleware"
)

// ---- mock UserRepository ----

type mockUserRepo struct {
	byEmail  map[string]*domain.User
	byID     map[uuid.UUID]*domain.User
	createFn func(ctx context.Context, u *domain.User) (*domain.User, error)
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) (*domain.User, error) {
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}

func (m *mockUserRepo) UpdateLastActive(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if u, ok := m.byID[id]; ok {
		delete(m.byEmail, u.Email)
		delete(m.byID, id)
		return nil
	}
	return domain.ErrNotFound
}

// ---- helpers ----

const testSecret = "test-secret-key"

func newAuthSvc(repo *mockUserRepo) *AuthService {
	return NewAuthService(repo, testSecret, 15*time.Minute, 7*24*time.Hour)
}

func parseClaims(t *testing.T, tokenStr string) *mw.Claims {
	t.Helper()
	claims := &mw.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
		return []byte(testSecret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("parseClaims: invalid token: %v", err)
	}
	return claims
}

// ---- Register tests ----

func TestAuthService_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	user, access, refresh, err := svc.Register(context.Background(), "alice@example.com", "Alice", "secret123")
	if err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("Register: expected non-nil user")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "alice@example.com")
	}
	if user.DisplayName != "Alice" {
		t.Errorf("DisplayName: got %q, want %q", user.DisplayName, "Alice")
	}
	if user.Role != domain.RoleMember {
		t.Errorf("Role: got %q, want member", user.Role)
	}
	if user.PasswordHash == "" {
		t.Error("PasswordHash should not be empty")
	}
	if access == "" {
		t.Error("access token should not be empty")
	}
	if refresh == "" {
		t.Error("refresh token should not be empty")
	}

	// Verify access token claims
	claims := parseClaims(t, access)
	if claims.UserID != user.ID.String() {
		t.Errorf("access token UserID: got %q, want %q", claims.UserID, user.ID.String())
	}
	if claims.Role != string(domain.RoleMember) {
		t.Errorf("access token Role: got %q, want member", claims.Role)
	}
}

func TestAuthService_Register_MissingFields(t *testing.T) {
	cases := []struct {
		name        string
		email       string
		displayName string
		password    string
	}{
		{"empty email", "", "Alice", "pass"},
		{"empty display name", "a@b.com", "", "pass"},
		{"empty password", "a@b.com", "Alice", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newAuthSvc(newMockUserRepo())
			_, _, _, err := svc.Register(context.Background(), tc.email, tc.displayName, tc.password)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	_, _, _, err := svc.Register(context.Background(), "dup@example.com", "User", "pass1")
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	_, _, _, err = svc.Register(context.Background(), "dup@example.com", "User2", "pass2")
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestAuthService_Register_RepoCreateError(t *testing.T) {
	repo := newMockUserRepo()
	repo.createFn = func(ctx context.Context, u *domain.User) (*domain.User, error) {
		return nil, errors.New("db error")
	}
	svc := newAuthSvc(repo)

	_, _, _, err := svc.Register(context.Background(), "x@example.com", "X", "pass")
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Errorf("expected db error in message, got %v", err)
	}
}

// ---- Login tests ----

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	// Register first so a real bcrypt hash is stored.
	registered, _, _, err := svc.Register(context.Background(), "bob@example.com", "Bob", "mypassword")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, access, refresh, err := svc.Login(context.Background(), "bob@example.com", "mypassword")
	if err != nil {
		t.Fatalf("Login: unexpected error: %v", err)
	}
	if user.ID != registered.ID {
		t.Errorf("Login: user ID mismatch")
	}
	if access == "" || refresh == "" {
		t.Error("expected non-empty tokens")
	}

	claims := parseClaims(t, access)
	if claims.UserID != user.ID.String() {
		t.Errorf("token UserID mismatch")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	_, _, _, err := svc.Register(context.Background(), "carol@example.com", "Carol", "correct")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, _, err = svc.Login(context.Background(), "carol@example.com", "wrong")
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	svc := newAuthSvc(newMockUserRepo())
	_, _, _, err := svc.Login(context.Background(), "nobody@example.com", "pass")
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---- Refresh tests ----

func TestAuthService_Refresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	_, _, refreshToken, err := svc.Register(context.Background(), "dave@example.com", "Dave", "pass")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	newAccess, newRefresh, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("Refresh: unexpected error: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty tokens from Refresh")
	}

	// The new access token should have the same UserID.
	origClaims := parseClaims(t, refreshToken)
	newClaims := parseClaims(t, newAccess)
	if origClaims.UserID != newClaims.UserID {
		t.Errorf("UserID mismatch after refresh: orig=%q new=%q", origClaims.UserID, newClaims.UserID)
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc := newAuthSvc(newMockUserRepo())
	_, _, err := svc.Refresh(context.Background(), "not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Refresh_WrongSecret(t *testing.T) {
	// Issue a token with a different secret, then try to refresh using the service's secret.
	other := NewAuthService(newMockUserRepo(), "different-secret", 15*time.Minute, 7*24*time.Hour)
	_, _, refreshToken, _ := other.Register(context.Background(), "eve@example.com", "Eve", "pass")

	svc := newAuthSvc(newMockUserRepo())
	_, _, err := svc.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Fatal("expected error for token signed with wrong secret, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---- ValidateToken (via Refresh round-trip) ----

func TestAuthService_Token_Expiry(t *testing.T) {
	// Create a service with a very short access expiry and confirm the token
	// is parseable before expiry.
	repo := newMockUserRepo()
	svc := NewAuthService(repo, testSecret, 1*time.Hour, 24*time.Hour)

	_, access, _, err := svc.Register(context.Background(), "frank@example.com", "Frank", "pass")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	claims := parseClaims(t, access)
	expiry := claims.ExpiresAt.Time
	issuedAt := claims.IssuedAt.Time

	if expiry.Before(issuedAt) {
		t.Error("token expiry should be after issued-at")
	}
	expectedExpiry := issuedAt.Add(1 * time.Hour)
	diff := expiry.Sub(expectedExpiry)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("expected expiry ~1h from issued-at, diff=%v", diff)
	}
}

// ---- Me tests ----

func TestAuthService_Me_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newAuthSvc(repo)

	registered, _, _, err := svc.Register(context.Background(), "grace@example.com", "Grace", "pass")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := svc.Me(context.Background(), registered.ID.String())
	if err != nil {
		t.Fatalf("Me: unexpected error: %v", err)
	}
	if got.ID != registered.ID {
		t.Errorf("Me: ID mismatch")
	}
}

func TestAuthService_Me_InvalidUUID(t *testing.T) {
	svc := newAuthSvc(newMockUserRepo())
	_, err := svc.Me(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Me_NotFound(t *testing.T) {
	svc := newAuthSvc(newMockUserRepo())
	_, err := svc.Me(context.Background(), uuid.New().String())
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
