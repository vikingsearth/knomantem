package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// AuthService is the interface the AuthHandler depends on.
type AuthService interface {
	Register(ctx context.Context, email, displayName, password string) (*domain.User, string, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, string, error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	Me(ctx context.Context, userID string) (*domain.User, error)
}

// AuthHandler handles /api/v1/auth/* endpoints.
type AuthHandler struct {
	svc AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// registerRequest is the body for POST /auth/register.
type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// loginRequest is the body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// refreshRequest is the body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// userResponse is the public representation of a User.
type userResponse struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	DisplayName string          `json:"display_name"`
	AvatarURL   *string         `json:"avatar_url"`
	Role        string          `json:"role"`
	Settings    json.RawMessage `json:"settings,omitempty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Role:        string(u.Role),
		Settings:    json.RawMessage(u.Settings),
		CreatedAt:   u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Email == "" || req.DisplayName == "" || req.Password == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"email, display_name, and password are required")
	}

	user, accessToken, refreshToken, err := h.svc.Register(
		c.Request().Context(), req.Email, req.DisplayName, req.Password,
	)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondCreated(c, map[string]any{
		"user":          toUserResponse(user),
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"email and password are required")
	}

	user, accessToken, refreshToken, err := h.svc.Login(
		c.Request().Context(), req.Email, req.Password,
	)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, map[string]any{
		"user":          toUserResponse(user),
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.RefreshToken == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"refresh_token is required")
	}

	accessToken, newRefreshToken, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, map[string]any{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(c echo.Context) error {
	userID := userIDFromCtx(c)
	if userID == "" {
		return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
	}

	user, err := h.svc.Me(c.Request().Context(), userID)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toUserResponse(user))
}
