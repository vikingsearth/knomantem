package middleware

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimiter returns an Echo middleware that enforces per-user (or per-IP
// when unauthenticated) rate limits using a token-bucket algorithm.
//
// r specifies the steady-state request rate (requests per second).
// burst specifies the maximum burst size.
func RateLimiter(r rate.Limit, burst int) echo.MiddlewareFunc {
	limiters := &sync.Map{}

	getLimiter := func(key string) *rate.Limiter {
		v, _ := limiters.LoadOrStore(key, rate.NewLimiter(r, burst))
		return v.(*rate.Limiter)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Prefer authenticated user ID as the rate-limit key; fall back to IP.
			key := UserIDFromContext(c)
			if key == "" {
				key = c.RealIP()
			}

			limiter := getLimiter(key)
			if !limiter.Allow() {
				return errorResponse(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			}

			return next(c)
		}
	}
}
