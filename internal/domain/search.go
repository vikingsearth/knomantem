package domain

import "github.com/google/uuid"

// SearchParams holds the full set of query parameters for a search request.
type SearchParams struct {
	Query     string
	SpaceID   *uuid.UUID
	Tags      string
	Freshness string
	From      string
	To        string
	Sort      string
	Cursor    string
	Limit     int
}

// SearchResult is the top-level search response.
type SearchResult struct {
	Items       []SearchItem   `json:"results"`
	Facets      *SearchFacets  `json:"facets,omitempty"`
	QueryTimeMs int64          `json:"query_time_ms"`
	Total       int            `json:"total"`
	NextCursor  string         `json:"next_cursor,omitempty"`
}

// SearchItem is a single search result.
type SearchItem struct {
	PageID    string         `json:"page_id"`
	Title     string         `json:"title"`
	Excerpt   string         `json:"excerpt"`
	Space     SpaceSummary   `json:"space"`
	Freshness FreshnessBrief `json:"freshness"`
	Tags      []TagBrief     `json:"tags,omitempty"`
	Score     float64        `json:"score"`
	UpdatedAt string         `json:"updated_at"`
}

// FreshnessBrief is a minimal freshness reference used in search results.
type FreshnessBrief struct {
	Score  float64 `json:"score"`
	Status string  `json:"status"`
}

// TagBrief is a minimal tag reference used in search results.
type TagBrief struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// SearchFacets contains facet counts for the search result set.
type SearchFacets struct {
	Spaces    []FacetSpace     `json:"spaces,omitempty"`
	Tags      []FacetTag       `json:"tags,omitempty"`
	Freshness []FacetFreshness `json:"freshness,omitempty"`
}

// FacetSpace is a space facet entry.
type FacetSpace struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FacetTag is a tag facet entry.
type FacetTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FacetFreshness is a freshness-status facet entry.
type FacetFreshness struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}
