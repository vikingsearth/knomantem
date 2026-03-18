package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FreshnessStatus represents the freshness category of a page.
type FreshnessStatus string

const (
	FreshnessFresh  FreshnessStatus = "fresh"
	FreshnessAging  FreshnessStatus = "aging"
	FreshnessStale  FreshnessStatus = "stale"
)

// Page is the central entity — a document within a Space.
type Page struct {
	ID              uuid.UUID       `json:"id"`
	SpaceID         uuid.UUID       `json:"space_id"`
	ParentID        *uuid.UUID      `json:"parent_id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content,omitempty"`
	Icon            *string         `json:"icon"`
	CoverImage      *string         `json:"cover_image"`
	Position        int             `json:"position"`
	Depth           int             `json:"depth"`
	IsTemplate      bool            `json:"is_template"`
	HasChildren     bool            `json:"has_children"`
	FreshnessStatus FreshnessStatus `json:"freshness_status"`
	Freshness       *FreshnessSummary `json:"freshness,omitempty"`
	Tags            []Tag           `json:"tags,omitempty"`
	Version         int             `json:"version"`
	CreatedBy       uuid.UUID       `json:"created_by"`
	UpdatedBy       uuid.UUID       `json:"updated_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// FreshnessSummary is the compact freshness info embedded in a Page response.
type FreshnessSummary struct {
	Score          float64   `json:"score"`
	Status         FreshnessStatus `json:"status"`
	LastVerifiedAt time.Time `json:"last_verified_at,omitempty"`
	NextReviewAt   time.Time `json:"next_review_at,omitempty"`
}

// PageVersion is a historical snapshot of a Page.
type PageVersion struct {
	ID            uuid.UUID       `json:"id"`
	PageID        uuid.UUID       `json:"page_id"`
	Version       int             `json:"version"`
	Title         string          `json:"title"`
	Content       json.RawMessage `json:"content,omitempty"`
	ChangeSummary string          `json:"change_summary"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedByName string          `json:"created_by_name,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// CreatePageRequest carries the fields needed to create a new Page.
type CreatePageRequest struct {
	Title      string          `json:"title"`
	ParentID   *uuid.UUID      `json:"parent_id"`
	Content    json.RawMessage `json:"content"`
	Icon       *string         `json:"icon"`
	Position   int             `json:"position"`
	IsTemplate bool            `json:"is_template"`
}

// UpdatePageRequest carries the fields that can be updated on a Page.
type UpdatePageRequest struct {
	Title         *string         `json:"title"`
	Content       json.RawMessage `json:"content"`
	Icon          *string         `json:"icon"`
	CoverImage    *string         `json:"cover_image"`
	ChangeSummary string          `json:"change_summary"`
}
