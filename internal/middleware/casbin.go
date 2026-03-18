package middleware

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/labstack/echo/v4"
)

// CasbinRBAC returns an Echo middleware that enforces Casbin RBAC rules.
// It reads the user's role from the Echo context (set by JWTAuth) and checks
// the (role, resource, action) tuple against the enforcer.
//
// resource is the Casbin object string (e.g. "/api/v1/pages").
// action is mapped from the HTTP method (GET→read, POST/PUT/DELETE→write).
//
// If the enforcer is nil the middleware is a no-op (useful in tests / early
// stages where policy has not been configured yet).
func CasbinRBAC(enforcer *casbin.Enforcer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if enforcer == nil {
				return next(c)
			}

			role := UserRoleFromContext(c)
			if role == "" {
				role = "anonymous"
			}

			resource := c.Request().URL.Path
			action := methodToAction(c.Request().Method)

			ok, err := enforcer.Enforce(role, resource, action)
			if err != nil {
				return errorResponse(c, http.StatusInternalServerError, "INTERNAL", "policy enforcement error")
			}
			if !ok {
				return errorResponse(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			}

			return next(c)
		}
	}
}

// methodToAction maps an HTTP method to a Casbin action string.
func methodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	default:
		return "write"
	}
}
