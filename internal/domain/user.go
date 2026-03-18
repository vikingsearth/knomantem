package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Role represents a user's system-level role.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// Settings is a generic JSON settings map stored as JSONB.
type Settings json.RawMessage

// User represents an authenticated Knomantem user.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	PasswordHash string     `json:"-"`
	AvatarURL    *string    `json:"avatar_url"`
	Role         Role       `json:"role"`
	Settings     Settings   `json:"settings,omitempty"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
