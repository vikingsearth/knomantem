// Package middleware contains Echo middleware used by the Knomantem API server.
package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// CORSConfig holds the allowed origins string from config.
type CORSConfig struct {
	// AllowOrigins is a comma-separated list of allowed origins, or "*".
	AllowOrigins string
}

// CORS returns an Echo middleware that applies CORS headers.
// origins is a comma-separated list of allowed origins (e.g. "https://app.example.com") or "*".
func CORS(origins string) echo.MiddlewareFunc {
	allowed := parseOrigins(origins)
	return echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     allowed,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
}

func parseOrigins(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
