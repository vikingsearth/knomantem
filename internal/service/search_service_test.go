package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// ---- mock SearchRepository ----
//
// mockSearchRepo is a hand-written in-memory mock that records every call and
// lets individual tests inject custom responses via function hooks.

type mockSearchRepo struct {
	// recorded calls
	indexedPages  []*domain.Page
	deletedIDs    []uuid.UUID
	searchCalls   []domain.SearchParams

	// injectable behaviour
	searchFn func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error)
	indexFn  func(ctx context.Context, p *domain.Page) error
	deleteFn func(ctx context.Context, pageID uuid.UUID) error
}

func newMockSearchRepo() *mockSearchRepo { return &mockSearchRepo{} }

func (m *mockSearchRepo) Index(ctx context.Context, p *domain.Page) error {
	if m.indexFn != nil {
		return m.indexFn(ctx, p)
	}
	m.indexedPages = append(m.indexedPages, p)
	return nil
}

func (m *mockSearchRepo) Delete(ctx context.Context, pageID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, pageID)
	}
	m.deletedIDs = append(m.deletedIDs, pageID)
	return nil
}

func (m *mockSearchRepo) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	m.searchCalls = append(m.searchCalls, params)
	if m.searchFn != nil {
		return m.searchFn(ctx, params)
	}
	return &domain.SearchResult{}, nil
}

// ---- mock PageRepository for SearchService ----
//
// searchTestPageRepo is a minimal stub used to satisfy the PageRepository
// dependency of SearchService.  It only needs GetByID for the current
// implementation; everything else can no-op.

type searchTestPageRepo struct {
	pages map[uuid.UUID]*domain.Page
}

func newSearchTestPageRepo() *searchTestPageRepo {
	return &searchTestPageRepo{pages: make(map[uuid.UUID]*domain.Page)}
}

func (r *searchTestPageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Page, error) {
	if p, ok := r.pages[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (r *searchTestPageRepo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]*domain.Page, error) {
	return nil, nil
}
func (r *searchTestPageRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Page, error) {
	return nil, nil
}
func (r *searchTestPageRepo) Create(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	return p, nil
}
func (r *searchTestPageRepo) Update(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	return p, nil
}
func (r *searchTestPageRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *searchTestPageRepo) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, position int) (*domain.Page, error) {
	return nil, domain.ErrNotFound
}
func (r *searchTestPageRepo) ListVersions(ctx context.Context, pageID uuid.UUID) ([]*domain.PageVersion, error) {
	return nil, nil
}
func (r *searchTestPageRepo) GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*domain.PageVersion, error) {
	return nil, domain.ErrNotFound
}
func (r *searchTestPageRepo) CreateVersion(ctx context.Context, v *domain.PageVersion) error {
	return nil
}

// ---- constructor helper ----

func newSearchSvc(sr *mockSearchRepo, pr domain.PageRepository) *SearchService {
	return NewSearchService(sr, pr)
}

// ---- Search tests ----

// TestSearchService_Search_PassesParamsToRepository verifies that the service
// forwards every SearchParams field to the underlying SearchRepository without
// modification.
func TestSearchService_Search_PassesParamsToRepository(t *testing.T) {
	sr := newMockSearchRepo()
	svc := newSearchSvc(sr, newSearchTestPageRepo())

	spaceID := uuid.New()
	params := domain.SearchParams{
		Query:     "knowledge graph",
		SpaceID:   &spaceID,
		Tags:      "architecture,design",
		Freshness: "fresh",
		From:      "2026-01-01",
		To:        "2026-03-20",
		Sort:      "relevance",
		Cursor:    "abc123",
		Limit:     15,
	}

	_, err := svc.Search(context.Background(), uuid.New().String(), params)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}

	if len(sr.searchCalls) != 1 {
		t.Fatalf("expected exactly 1 search call, got %d", len(sr.searchCalls))
	}

	got := sr.searchCalls[0]
	if got.Query != params.Query {
		t.Errorf("Query: got %q, want %q", got.Query, params.Query)
	}
	if got.SpaceID == nil || *got.SpaceID != spaceID {
		t.Errorf("SpaceID mismatch")
	}
	if got.Tags != params.Tags {
		t.Errorf("Tags: got %q, want %q", got.Tags, params.Tags)
	}
	if got.Freshness != params.Freshness {
		t.Errorf("Freshness: got %q, want %q", got.Freshness, params.Freshness)
	}
	if got.Sort != params.Sort {
		t.Errorf("Sort: got %q, want %q", got.Sort, params.Sort)
	}
	if got.Cursor != params.Cursor {
		t.Errorf("Cursor: got %q, want %q", got.Cursor, params.Cursor)
	}
	if got.Limit != params.Limit {
		t.Errorf("Limit: got %d, want %d", got.Limit, params.Limit)
	}
}

// TestSearchService_Search_ReturnsResultsFromRepository verifies that the
// results produced by the SearchRepository are returned unchanged to the
// caller.
func TestSearchService_Search_ReturnsResultsFromRepository(t *testing.T) {
	sr := newMockSearchRepo()
	spaceID := uuid.New()
	sr.searchFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
		return &domain.SearchResult{
			Items: []domain.SearchItem{
				{
					PageID:  uuid.New().String(),
					Title:   "Getting Started",
					Excerpt: "A guide to ...",
					Space:   domain.SpaceSummary{ID: spaceID.String(), Name: "Docs"},
					Freshness: domain.FreshnessBrief{
						Score:  92.0,
						Status: "fresh",
					},
					Score:     1.23,
					UpdatedAt: "2026-03-20T10:00:00Z",
				},
			},
			Total:       1,
			QueryTimeMs: 5,
		}, nil
	}

	svc := newSearchSvc(sr, newSearchTestPageRepo())

	result, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
		Query: "getting started",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Search: expected non-nil result")
	}
	if result.Total != 1 {
		t.Errorf("Total: got %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("Items length: got %d, want 1", len(result.Items))
	}
	if result.Items[0].Title != "Getting Started" {
		t.Errorf("Item title: got %q, want %q", result.Items[0].Title, "Getting Started")
	}
}

// TestSearchService_Search_EmptyResults verifies that an empty result set is
// returned cleanly without error.
func TestSearchService_Search_EmptyResults(t *testing.T) {
	sr := newMockSearchRepo()
	// Default searchFn returns &domain.SearchResult{} with zero Items.
	svc := newSearchSvc(sr, newSearchTestPageRepo())

	result, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
		Query: "no match",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Search: unexpected error for empty results: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even for empty set")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

// TestSearchService_Search_PropagatesRepositoryError verifies that errors from
// the SearchRepository bubble up to the caller unchanged.
func TestSearchService_Search_PropagatesRepositoryError(t *testing.T) {
	sr := newMockSearchRepo()
	repoErr := errors.New("index unavailable")
	sr.searchFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
		return nil, repoErr
	}
	svc := newSearchSvc(sr, newSearchTestPageRepo())

	_, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
		Query: "anything",
		Limit: 20,
	})
	if err == nil {
		t.Fatal("expected error from repository, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error %v, got %v", repoErr, err)
	}
}

// TestSearchService_Search_NoSpaceFilter verifies that when SpaceID is nil
// the param is forwarded with a nil SpaceID (no space filter applied).
func TestSearchService_Search_NoSpaceFilter(t *testing.T) {
	sr := newMockSearchRepo()
	svc := newSearchSvc(sr, newSearchTestPageRepo())

	_, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
		Query: "everything",
		Limit: 20,
		// SpaceID intentionally nil
	})
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}

	if len(sr.searchCalls) != 1 {
		t.Fatalf("expected 1 search call, got %d", len(sr.searchCalls))
	}
	if sr.searchCalls[0].SpaceID != nil {
		t.Errorf("SpaceID should be nil when no space filter, got %v", sr.searchCalls[0].SpaceID)
	}
}

// TestSearchService_Search_WithPagination verifies that cursor and limit are
// forwarded correctly for paginated requests.
func TestSearchService_Search_WithPagination(t *testing.T) {
	cases := []struct {
		name   string
		cursor string
		limit  int
	}{
		{"first page", "", 10},
		{"subsequent page", "cursor-abc", 10},
		{"max limit", "", 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sr := newMockSearchRepo()
			svc := newSearchSvc(sr, newSearchTestPageRepo())

			_, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
				Query:  "paging test",
				Cursor: tc.cursor,
				Limit:  tc.limit,
			})
			if err != nil {
				t.Fatalf("Search: unexpected error: %v", err)
			}
			if len(sr.searchCalls) == 0 {
				t.Fatal("expected search to be called")
			}
			call := sr.searchCalls[0]
			if call.Cursor != tc.cursor {
				t.Errorf("Cursor: got %q, want %q", call.Cursor, tc.cursor)
			}
			if call.Limit != tc.limit {
				t.Errorf("Limit: got %d, want %d", call.Limit, tc.limit)
			}
		})
	}
}

// TestSearchService_Search_WithCursor verifies that a non-empty NextCursor in
// the repository result is passed back to the caller.
func TestSearchService_Search_WithCursor(t *testing.T) {
	sr := newMockSearchRepo()
	sr.searchFn = func(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
		return &domain.SearchResult{
			Items:      []domain.SearchItem{},
			Total:      50,
			NextCursor: "next-page-token",
		}, nil
	}
	svc := newSearchSvc(sr, newSearchTestPageRepo())

	result, err := svc.Search(context.Background(), uuid.New().String(), domain.SearchParams{
		Query: "paged",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if result.NextCursor != "next-page-token" {
		t.Errorf("NextCursor: got %q, want %q", result.NextCursor, "next-page-token")
	}
	if result.Total != 50 {
		t.Errorf("Total: got %d, want 50", result.Total)
	}
}
