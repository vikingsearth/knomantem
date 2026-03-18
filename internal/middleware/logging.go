package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// Logger returns an Echo middleware that logs every request using slog.
// It records method, path, status, latency, user ID, and request ID.
func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			res := c.Response()

			err := next(c)

			latency := time.Since(start)

			// Extract user ID from context if JWT middleware has already run.
			userID := ""
			if v := c.Get("user_id"); v != nil {
				if s, ok := v.(string); ok {
					userID = s
				}
			}

			requestID := req.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = res.Header().Get("X-Request-ID")
			}

			attrs := []any{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Duration("latency", latency),
				slog.String("request_id", requestID),
			}
			if userID != "" {
				attrs = append(attrs, slog.String("user_id", userID))
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				logger.Error("request", attrs...)
			} else {
				logger.Info("request", attrs...)
			}

			return err
		}
	}
}
