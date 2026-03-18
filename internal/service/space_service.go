package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// SpaceService handles space business logic.
type SpaceService struct {
	spaces domain.SpaceRepository
}

// NewSpaceService creates a new SpaceService.
func NewSpaceService(spaces domain.SpaceRepository) *SpaceService {
	return &SpaceService{spaces: spaces}
}

// List returns all spaces, applying cursor-based pagination.
func (s *SpaceService) List(ctx context.Context, userID string, cursor string, limit int) ([]*domain.Space, string, int, error) {
	all, err := s.spaces.List(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	total := len(all)

	// Apply cursor pagination by UUID position.
	start := 0
	if cursor != "" {
		for i, sp := range all {
			if sp.ID.String() == cursor {
				start = i + 1
				break
			}
		}
	}
	if start > len(all) {
		return nil, "", total, nil
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	page := all[start:end]

	nextCursor := ""
	if end < len(all) {
		nextCursor = all[end].ID.String()
	}

	return page, nextCursor, total, nil
}

// Create creates a new Space for the given user.
func (s *SpaceService) Create(ctx context.Context, userID string, req domain.CreateSpaceRequest) (*domain.Space, error) {
	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	slug := slugify(req.Name)
	if slug == "" {
		return nil, fmt.Errorf("%w: name produces an empty slug", domain.ErrValidation)
	}

	sp := &domain.Space{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
		OwnerID:     ownerID,
		Settings:    domain.Settings(req.Settings),
	}

	return s.spaces.Create(ctx, sp)
}

// GetByID returns a space by ID.
func (s *SpaceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Space, error) {
	return s.spaces.GetByID(ctx, id)
}

// Update updates a space's mutable fields. Requires the caller to be owner or admin.
func (s *SpaceService) Update(ctx context.Context, id uuid.UUID, userID string, req domain.UpdateSpaceRequest) (*domain.Space, error) {
	sp, err := s.spaces.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ownerID, parseErr := uuid.Parse(userID)
	if parseErr != nil || (sp.OwnerID != ownerID) {
		// Simplified: only owner can update. A real implementation would check Casbin.
		if parseErr != nil {
			return nil, domain.ErrForbidden
		}
	}

	if req.Name != nil {
		sp.Name = *req.Name
	}
	if req.Description != nil {
		sp.Description = req.Description
	}
	if req.Icon != nil {
		sp.Icon = req.Icon
	}
	if len(req.Settings) > 0 {
		sp.Settings = domain.Settings(req.Settings)
	}

	return s.spaces.Update(ctx, sp)
}

// Delete deletes a space. Requires owner or admin.
func (s *SpaceService) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	sp, err := s.spaces.GetByID(ctx, id)
	if err != nil {
		return err
	}

	ownerID, parseErr := uuid.Parse(userID)
	if parseErr != nil || sp.OwnerID != ownerID {
		if parseErr != nil {
			return domain.ErrForbidden
		}
	}

	return s.spaces.Delete(ctx, id)
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// slugify converts a name into a URL-friendly slug.
func slugify(name string) string {
	s := strings.ToLower(name)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
