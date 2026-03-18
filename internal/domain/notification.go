package domain

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents an in-app notification for a user.
type Notification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"`
	PageID    *uuid.UUID `json:"page_id,omitempty"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}
