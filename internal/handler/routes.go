package handler

import (
	"github.com/labstack/echo/v4"

	mw "github.com/knomantem/knomantem/internal/middleware"
)

// Handlers bundles all handler instances required by RegisterRoutes.
type Handlers struct {
	Auth            *AuthHandler
	Space           *SpaceHandler
	Page            *PageHandler
	Search          *SearchHandler
	Freshness       *FreshnessHandler
	Graph           *GraphHandler
	Tag             *TagHandler
	Presence        *PresenceHandler
	PresenceViewers *PresenceHTTPHandler
}

// RegisterRoutes wires all 30+ REST and WebSocket endpoints onto the given Echo
// instance. The jwtSecret is used to build the JWT authentication middleware
// applied to every authenticated group.
//
// Endpoint inventory (section 4 of POC.md):
//
//	Auth      (4 endpoints)  – POST register, POST login, POST refresh, GET me
//	Spaces    (5 endpoints)  – GET list, POST create, GET by-id, PUT update, DELETE
//	Pages     (10 endpoints) – GET tree, POST create, GET by-id, PUT update, DELETE,
//	                           PUT move, GET versions, GET version, POST import-md,
//	                           GET export-md
//	Search    (1 endpoint)   – GET search
//	Freshness (4 endpoints)  – GET page-freshness, POST verify, PUT settings, GET dashboard
//	Graph     (3 endpoints)  – GET neighbors, POST edge, GET explore
//	Tags      (3 endpoints)  – GET list, POST create, POST page-tags
//	Presence  (2 endpoints)  – WS connect, GET viewers
func RegisterRoutes(e *echo.Echo, h *Handlers, jwtSecret string) {
	api := e.Group("/api/v1")

	// ── Public (no JWT required) ───────────────────────────────────────────
	auth := api.Group("/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)
	auth.POST("/refresh", h.Auth.Refresh)

	// ── Authenticated ──────────────────────────────────────────────────────
	jwt := mw.JWTAuth(jwtSecret)
	secured := api.Group("", jwt)

	// Auth (self)
	secured.GET("/auth/me", h.Auth.Me)

	// Spaces
	secured.GET("/spaces", h.Space.List)
	secured.POST("/spaces", h.Space.Create)
	secured.GET("/spaces/:id", h.Space.GetByID)
	secured.PUT("/spaces/:id", h.Space.Update)
	secured.DELETE("/spaces/:id", h.Space.Delete)

	// Pages (space-scoped creation & tree listing)
	secured.GET("/spaces/:spaceId/pages", h.Page.ListBySpace)
	secured.POST("/spaces/:spaceId/pages", h.Page.Create)

	// Pages (individual operations)
	secured.GET("/pages/:id", h.Page.GetByID)
	secured.PUT("/pages/:id", h.Page.Update)
	secured.DELETE("/pages/:id", h.Page.Delete)
	secured.PUT("/pages/:id/move", h.Page.Move)

	// Page versions
	secured.GET("/pages/:id/versions", h.Page.ListVersions)
	secured.GET("/pages/:id/versions/:version", h.Page.GetVersion)

	// Page import / export
	secured.POST("/pages/:id/import/markdown", h.Page.ImportMarkdown)
	secured.GET("/pages/:id/export/markdown", h.Page.ExportMarkdown)

	// Search
	secured.GET("/search", h.Search.Search)

	// Freshness (page-specific)
	secured.GET("/pages/:id/freshness", h.Freshness.GetByPageID)
	secured.POST("/pages/:id/freshness/verify", h.Freshness.Verify)
	secured.PUT("/pages/:id/freshness/settings", h.Freshness.UpdateSettings)

	// Freshness dashboard
	secured.GET("/freshness/dashboard", h.Freshness.Dashboard)

	// Graph
	secured.GET("/pages/:id/graph", h.Graph.GetNeighbors)
	secured.POST("/pages/:id/graph/edges", h.Graph.CreateEdge)
	secured.GET("/graph/explore", h.Graph.Explore)

	// Tags
	secured.GET("/tags", h.Tag.List)
	secured.POST("/tags", h.Tag.Create)
	secured.POST("/pages/:id/tags", h.Tag.AddToPage)

	// Presence
	// WebSocket upgrade — JWT is validated before upgrade
	secured.GET("/pages/:id/presence", h.Presence.Connect)
	// REST: who is currently viewing
	secured.GET("/pages/:id/presence/viewers", h.PresenceViewers.GetViewers)
}
