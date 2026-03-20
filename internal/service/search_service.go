package service

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// freshnessMultiplier returns the score boost factor for a given freshness status.
// Stale content is penalised to push it lower in results; fresh content is unaffected.
// Pages with no freshness record receive a mild penalty (treated as mildly aging).
func freshnessMultiplier(status domain.FreshnessStatus, hasRecord bool) float64 {
	if !hasRecord {
		return 0.90
	}
	switch status {
	case domain.FreshnessFresh:
		return 1.00
	case domain.FreshnessAging:
		return 0.85
	default: // FreshnessStale or any unrecognised value
		return 0.60
	}
}

// SearchService orchestrates full-text search with freshness-weighted re-ranking.
type SearchService struct {
	search    domain.SearchRepository
	pages     domain.PageRepository
	freshness domain.FreshnessRepository
}

// NewSearchService creates a new SearchService.
func NewSearchService(search domain.SearchRepository, pages domain.PageRepository, freshness domain.FreshnessRepository) *SearchService {
	return &SearchService{search: search, pages: pages, freshness: freshness}
}

// Search executes a full-text search, enriches each result with its freshness record,
// applies a freshness multiplier to the Bleve BM25 score, and re-sorts results.
func (s *SearchService) Search(ctx context.Context, userID string, params domain.SearchParams) (*domain.SearchResult, error) {
	result, err := s.search.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return result, nil
	}

	// Batch-fetch freshness records for all result pages in a single query.
	pageIDs := make([]uuid.UUID, 0, len(result.Items))
	for _, item := range result.Items {
		id, parseErr := uuid.Parse(item.PageID)
		if parseErr == nil {
			pageIDs = append(pageIDs, id)
		}
	}

	freshnessMap := make(map[string]*domain.Freshness, len(pageIDs))
	if len(pageIDs) > 0 {
		records, fetchErr := s.freshness.GetByPageIDs(ctx, pageIDs)
		if fetchErr == nil {
			for _, f := range records {
				freshnessMap[f.PageID.String()] = f
			}
		}
		// If the batch fetch fails we proceed without freshness data rather than
		// returning an error — search results are still useful even without boosting.
	}

	// Apply freshness multiplier and populate the Freshness field on each item.
	for i := range result.Items {
		item := &result.Items[i]
		f, hasRecord := freshnessMap[item.PageID]

		var status domain.FreshnessStatus
		if hasRecord {
			status = f.Status
			item.Freshness = domain.FreshnessBrief{
				Score:  f.Score,
				Status: string(f.Status),
			}
		}

		item.Score = item.Score * freshnessMultiplier(status, hasRecord)
	}

	// Re-sort by adjusted score descending.
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].Score > result.Items[j].Score
	})

	return result, nil
}
