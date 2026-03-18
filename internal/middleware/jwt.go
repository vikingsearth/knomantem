package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// Claims represents the payload of a Knomantem JWT access token.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// JWTAuth returns an Echo middleware that validates a Bearer JWT token.
// On success it stores "user_id" and "user_role" in the Echo context.
// On failure it returns 401 Unauthorized.
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, err := extractBearer(c.Request())
			if err != nil {
				return errorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return errorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			}

			c.Set("user_id", claims.UserID)
			c.Set("user_role", claims.Role)
			return next(c)
		}
	}
}

// extractBearer reads the Authorization header and returns the token string.
func extractBearer(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", domain.ErrUnauthorized
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", domain.ErrUnauthorized
	}
	if parts[1] == "" {
		return "", domain.ErrUnauthorized
	}
	return parts[1], nil
}

// UserIDFromContext retrieves the authenticated user ID from an Echo context.
// Returns empty string if not set.
func UserIDFromContext(c echo.Context) string {
	if v := c.Get("user_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UserRoleFromContext retrieves the authenticated user role from an Echo context.
func UserRoleFromContext(c echo.Context) string {
	if v := c.Get("user_role"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
