package service

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/knomantem/knomantem/internal/domain"
)

// presenceEntry tracks a single WebSocket connection for a user on a page.
type presenceEntry struct {
	conn        *websocket.Conn
	userID      string
	displayName string
	joinedAt    time.Time
}

// PresenceService manages in-memory WebSocket presence connections.
// It implements both handler.PresenceService and handler.PresenceViewersService.
type PresenceService struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID][]*presenceEntry
}

// NewPresenceService creates a new PresenceService.
func NewPresenceService() *PresenceService {
	return &PresenceService{
		rooms: make(map[uuid.UUID][]*presenceEntry),
	}
}

// presenceMessage is the JSON structure sent to WebSocket clients on join/leave events.
type presenceMessage struct {
	Event   string                    `json:"event"`
	PageID  string                    `json:"page_id"`
	Viewers []domain.PresenceUser     `json:"viewers"`
}

// Join registers a WebSocket connection for a user on a page.
// It blocks until the connection is closed.
func (s *PresenceService) Join(pageID uuid.UUID, userID string, conn *websocket.Conn) {
	entry := &presenceEntry{
		conn:     conn,
		userID:   userID,
		joinedAt: time.Now(),
	}

	s.mu.Lock()
	s.rooms[pageID] = append(s.rooms[pageID], entry)
	s.mu.Unlock()

	s.broadcast(pageID, "join")

	// Read loop — block until connection closes or errors.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	// Cleanup on disconnect.
	s.mu.Lock()
	entries := s.rooms[pageID]
	for i, e := range entries {
		if e == entry {
			s.rooms[pageID] = append(entries[:i], entries[i+1:]...)
			break
		}
	}
	if len(s.rooms[pageID]) == 0 {
		delete(s.rooms, pageID)
	}
	s.mu.Unlock()

	s.broadcast(pageID, "leave")
	_ = conn.Close()
}

// Viewers returns the list of users currently viewing a page.
func (s *PresenceService) Viewers(pageID uuid.UUID) []domain.PresenceUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.rooms[pageID]
	out := make([]domain.PresenceUser, len(entries))
	for i, e := range entries {
		out[i] = domain.PresenceUser{
			UserID:   e.userID,
			JoinedAt: e.joinedAt,
		}
	}
	return out
}

// broadcast sends the current viewer list to all connections on a page.
func (s *PresenceService) broadcast(pageID uuid.UUID, event string) {
	s.mu.RLock()
	entries := s.rooms[pageID]
	viewers := make([]domain.PresenceUser, len(entries))
	for i, e := range entries {
		viewers[i] = domain.PresenceUser{
			UserID:   e.userID,
			JoinedAt: e.joinedAt,
		}
	}
	s.mu.RUnlock()

	msg := presenceMessage{
		Event:   event,
		PageID:  pageID.String(),
		Viewers: viewers,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("presence: marshal broadcast", slog.String("error", err.Error()))
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.rooms[pageID] {
		if wErr := e.conn.WriteMessage(websocket.TextMessage, data); wErr != nil {
			slog.Debug("presence: write to client failed",
				slog.String("user_id", e.userID),
				slog.String("error", wErr.Error()),
			)
		}
	}
}
