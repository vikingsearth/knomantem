package domain

import (
	"time"

	"github.com/google/uuid"
)

// Tag is a label that can be applied to pages.
type Tag struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Color         string    `json:"color"`
	IsAIGenerated bool      `json:"is_ai_generated"`
	PageCount     int       `json:"page_count,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateTagRequest carries the fields needed to create a new Tag.
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PageTagAssignment represents associating a tag with a page at a given confidence level.
type PageTagAssignment struct {
	TagID           uuid.UUID `json:"tag_id"`
	ConfidenceScore float64   `json:"confidence_score"`
}

// TagWithScore is a tag augmented with the confidence score for a page assignment.
type TagWithScore struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Color           string    `json:"color"`
	ConfidenceScore float64   `json:"confidence_score"`
}
