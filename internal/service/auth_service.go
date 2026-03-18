// Package service contains all business-logic services for Knomantem.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/knomantem/knomantem/internal/domain"
	mw "github.com/knomantem/knomantem/internal/middleware"
)

// AuthService handles user registration, authentication, and JWT issuance.
type AuthService struct {
	users         domain.UserRepository
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	users domain.UserRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
	refreshExpiry time.Duration,
) *AuthService {
	return &AuthService{
		users:         users,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// Register creates a new user account, hashes the password, and returns tokens.
func (s *AuthService) Register(ctx context.Context, email, displayName, password string) (*domain.User, string, string, error) {
	if email == "" || displayName == "" || password == "" {
		return nil, "", "", fmt.Errorf("%w: email, display_name, and password are required", domain.ErrValidation)
	}

	// Check for existing user.
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, "", "", fmt.Errorf("%w: email already registered", domain.ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth: hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: string(hash),
		Role:         domain.RoleMember,
	}

	created, err := s.users.Create(ctx, user)
	if err != nil {
		return nil, "", "", err
	}

	accessToken, refreshToken, err := s.issueTokens(created)
	if err != nil {
		return nil, "", "", err
	}

	return created, accessToken, refreshToken, nil
}

// Login authenticates a user with email/password and returns tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", "", domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", domain.ErrUnauthorized
	}

	_ = s.users.UpdateLastActive(ctx, user.ID, time.Now())

	accessToken, refreshToken, err := s.issueTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

// Refresh validates a refresh token and issues new token pair.
func (s *AuthService) Refresh(_ context.Context, refreshToken string) (string, string, error) {
	claims := &mw.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", domain.ErrUnauthorized
	}

	user := &domain.User{
		ID:   uuid.MustParse(claims.UserID),
		Role: domain.Role(claims.Role),
	}

	accessToken, newRefresh, err := s.issueTokens(user)
	if err != nil {
		return "", "", err
	}
	return accessToken, newRefresh, nil
}

// Me returns the authenticated user's profile.
func (s *AuthService) Me(ctx context.Context, userID string) (*domain.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	return s.users.GetByID(ctx, id)
}

// issueTokens generates an access token and refresh token for the given user.
func (s *AuthService) issueTokens(u *domain.User) (accessToken, refreshToken string, err error) {
	now := time.Now()

	accessExpiry := s.jwtExpiry
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry := s.refreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}

	accessClaims := &mw.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessExpiry)),
		},
		UserID: u.ID.String(),
		Role:   string(u.Role),
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("auth: sign access token: %w", err)
	}

	refreshClaims := &mw.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshExpiry)),
		},
		UserID: u.ID.String(),
		Role:   string(u.Role),
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", fmt.Errorf("auth: sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}
