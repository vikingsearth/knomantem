package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Space is a top-level container for pages (equivalent to a "workspace" or
// "knowledge base" in competing products).
type Space struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	OwnerID     uuid.UUID       `json:"owner_id"`
	Settings    Settings        `json:"settings,omitempty"`
	PageCount   int             `json:"page_count,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateSpaceRequest carries the fields needed to create a new Space.
type CreateSpaceRequest struct {
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	Settings    json.RawMessage `json:"settings"`
}

// UpdateSpaceRequest carries the fields that can be updated on a Space.
type UpdateSpaceRequest struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	Settings    json.RawMessage `json:"settings"`
}
