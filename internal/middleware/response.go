package middleware

import (
	"github.com/labstack/echo/v4"
)

// errorResponse writes a standard {"error":{"code":"...","message":"..."}} JSON body.
func errorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
