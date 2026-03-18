package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// PresenceService is the interface the PresenceHandler depends on.
type PresenceService interface {
	// Join registers a WebSocket connection for a user on a page.
	// It blocks until the connection closes.
	Join(pageID uuid.UUID, userID string, conn *websocket.Conn)
}

// PresenceHandler handles the WebSocket upgrade for presence notifications.
type PresenceHandler struct {
	svc      PresenceService
	upgrader websocket.Upgrader
}

// NewPresenceHandler creates a new PresenceHandler.
func NewPresenceHandler(svc PresenceService, allowedOrigins []string) *PresenceHandler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if len(allowedOrigins) == 0 {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
	}
	return &PresenceHandler{svc: svc, upgrader: upgrader}
}

// Connect handles GET /api/v1/pages/:id/presence (WebSocket upgrade).
// The client must already be authenticated; the JWT middleware runs before this.
func (h *PresenceHandler) Connect(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	userID := userIDFromCtx(c)
	if userID == "" {
		return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
	}

	conn, upgradeErr := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if upgradeErr != nil {
		// upgrader has already written the HTTP error response
		return nil
	}

	// Block until the connection closes; the service manages the hub.
	h.svc.Join(pageID, userID, conn)
	return nil
}

// GetPresence handles GET /api/v1/pages/:id/presence/viewers (HTTP, non-WS).
// Returns the list of users currently viewing the page.
type PresenceViewersService interface {
	Viewers(pageID uuid.UUID) []domain.PresenceUser
}

// PresenceHTTPHandler handles the REST presence endpoint (who is viewing).
type PresenceHTTPHandler struct {
	svc PresenceViewersService
}

// NewPresenceHTTPHandler creates a handler for the REST presence viewers endpoint.
func NewPresenceHTTPHandler(svc PresenceViewersService) *PresenceHTTPHandler {
	return &PresenceHTTPHandler{svc: svc}
}

// GetViewers handles GET /api/v1/pages/:id/presence/viewers.
func (h *PresenceHTTPHandler) GetViewers(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	viewers := h.svc.Viewers(pageID)
	out := make([]map[string]any, len(viewers))
	for i, v := range viewers {
		out[i] = map[string]any{
			"user_id":      v.UserID,
			"display_name": v.DisplayName,
			"avatar_url":   v.AvatarURL,
			"joined_at":    v.JoinedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	return respondOK(c, out)
}
