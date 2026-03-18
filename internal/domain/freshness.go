package domain

import (
	"time"

	"github.com/google/uuid"
)

// Freshness tracks the knowledge freshness state of a Page.
type Freshness struct {
	ID                 uuid.UUID       `json:"id"`
	PageID             uuid.UUID       `json:"page_id"`
	OwnerID            uuid.UUID       `json:"owner_id"`
	Score              float64         `json:"freshness_score"`
	Status             FreshnessStatus `json:"status"`
	ReviewIntervalDays int             `json:"review_interval_days"`
	DecayRate          float64         `json:"decay_rate"`
	LastReviewedAt     time.Time       `json:"last_reviewed_at,omitempty"`
	NextReviewAt       time.Time       `json:"next_review_at,omitempty"`
	LastVerifiedAt     time.Time       `json:"last_verified_at,omitempty"`
	LastVerifiedBy     uuid.UUID       `json:"last_verified_by,omitempty"`
	LastVerifiedByName string          `json:"last_verified_by_name,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// FreshnessSettingsRequest carries the fields the user can update on a Freshness record.
type FreshnessSettingsRequest struct {
	ReviewIntervalDays *int     `json:"review_interval_days"`
	DecayRate          *float64 `json:"decay_rate"`
	OwnerID            *string  `json:"owner_id"`
}

// FreshnessDashboard is the response for the dashboard endpoint.
type FreshnessDashboard struct {
	Summary    FreshnessSummaryStats    `json:"summary"`
	Pages      []FreshnessPageSummary   `json:"pages"`
	NextCursor string                   `json:"next_cursor,omitempty"`
	Total      int                      `json:"total"`
}

// FreshnessSummaryStats is the aggregate freshness statistics block.
type FreshnessSummaryStats struct {
	TotalPages   int     `json:"total_pages"`
	Fresh        int     `json:"fresh"`
	Aging        int     `json:"aging"`
	Stale        int     `json:"stale"`
	AverageScore float64 `json:"average_score"`
}

// FreshnessPageSummary is a slim page record used in the dashboard.
type FreshnessPageSummary struct {
	PageID         string          `json:"page_id"`
	Title          string          `json:"title"`
	Space          SpaceSummary    `json:"space"`
	FreshnessScore float64         `json:"freshness_score"`
	Status         FreshnessStatus `json:"status"`
	LastVerifiedAt *string         `json:"last_verified_at"`
	NextReviewAt   string          `json:"next_review_at"`
	Owner          UserSummary     `json:"owner"`
}

// SpaceSummary is a minimal space reference.
type SpaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserSummary is a minimal user reference.
type UserSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}
