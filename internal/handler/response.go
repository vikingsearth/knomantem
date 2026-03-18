// Package handler contains all Echo HTTP handler implementations for the
// Knomantem REST API.
package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// respondOK writes {"data": data} with HTTP 200.
func respondOK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, map[string]any{"data": data})
}

// respondCreated writes {"data": data} with HTTP 201.
func respondCreated(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, map[string]any{"data": data})
}

// respondNoContent writes HTTP 204 with no body.
func respondNoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// respondError writes {"error":{"code":"...","message":"..."}} with the given status.
func respondError(c echo.Context, status int, code, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// mapDomainError translates a domain sentinel error into an HTTP response.
// It returns the echo error so the caller can "return mapDomainError(c, err)".
func mapDomainError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return respondError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConflict):
		return respondError(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domain.ErrValidation):
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return respondError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		return respondError(c, http.StatusInternalServerError, "INTERNAL", "an internal error occurred")
	}
}

// userIDFromCtx reads the authenticated user ID stored by the JWT middleware.
func userIDFromCtx(c echo.Context) string {
	if v := c.Get("user_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// userRoleFromCtx reads the authenticated user role stored by the JWT middleware.
func userRoleFromCtx(c echo.Context) string {
	if v := c.Get("user_role"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// pagination is the standard pagination envelope added to list responses.
type pagination struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Total      int     `json:"total"`
}
