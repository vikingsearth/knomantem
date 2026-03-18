package domain

import "time"

// PresenceUser represents a user currently viewing a page.
type PresenceUser struct {
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	JoinedAt    time.Time `json:"joined_at"`
}
