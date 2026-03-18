package service

import (
	"context"

	"github.com/knomantem/knomantem/internal/domain"
)

// SearchService orchestrates full-text search.
type SearchService struct {
	search domain.SearchRepository
	pages  domain.PageRepository
}

// NewSearchService creates a new SearchService.
func NewSearchService(search domain.SearchRepository, pages domain.PageRepository) *SearchService {
	return &SearchService{search: search, pages: pages}
}

// Search executes a full-text search and enriches results with page metadata.
func (s *SearchService) Search(ctx context.Context, userID string, params domain.SearchParams) (*domain.SearchResult, error) {
	result, err := s.search.Search(ctx, params)
	if err != nil {
		return nil, err
	}
	return result, nil
}
