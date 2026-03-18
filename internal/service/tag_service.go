package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// TagService handles tag management.
type TagService struct {
	tags domain.TagRepository
}

// NewTagService creates a new TagService.
func NewTagService(tags domain.TagRepository) *TagService {
	return &TagService{tags: tags}
}

// List returns all tags optionally filtered by a name prefix.
func (s *TagService) List(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	return s.tags.List(ctx, query, limit)
}

// Create creates a new tag.
func (s *TagService) Create(ctx context.Context, userID string, req domain.CreateTagRequest) (*domain.Tag, error) {
	t := &domain.Tag{
		ID:    uuid.New(),
		Name:  req.Name,
		Color: req.Color,
	}
	if t.Color == "" {
		t.Color = "#6366F1"
	}
	return s.tags.Create(ctx, t)
}

// AddToPage associates tags with a page.
func (s *TagService) AddToPage(ctx context.Context, pageID uuid.UUID, userID string, assignments []domain.PageTagAssignment) ([]domain.TagWithScore, error) {
	return s.tags.AddToPage(ctx, pageID, assignments)
}
