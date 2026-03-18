// Package search provides Bleve-backed full-text search for Knomantem.
package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// BleveIndex wraps a bleve.Index and provides search operations.
type BleveIndex struct {
	index bleve.Index
}

// NewBleveIndex opens (or creates) the Bleve index at the given path.
// If the path does not exist a new index is created.
func NewBleveIndex(path string) (*BleveIndex, error) {
	index, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(path, mapping)
	}
	if err != nil {
		return nil, fmt.Errorf("bleve: open index at %s: %w", path, err)
	}
	return &BleveIndex{index: index}, nil
}

// Close closes the underlying Bleve index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}

// pageDoc is the document structure stored in the Bleve index.
type pageDoc struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	SpaceID   string    `json:"space_id"`
	Tags      string    `json:"tags"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SearchRepoAdapter adapts BleveIndex to domain.SearchRepository.
type SearchRepoAdapter struct {
	bleve *BleveIndex
}

// NewSearchRepoAdapter creates a SearchRepoAdapter.
func NewSearchRepoAdapter(b *BleveIndex) *SearchRepoAdapter {
	return &SearchRepoAdapter{bleve: b}
}

// Index adds or updates a page in the search index.
func (a *SearchRepoAdapter) Index(ctx context.Context, p *domain.Page) error {
	doc := pageDoc{
		ID:        p.ID.String(),
		Title:     p.Title,
		SpaceID:   p.SpaceID.String(),
		Status:    string(p.FreshnessStatus),
		UpdatedAt: p.UpdatedAt,
	}
	// Build a plain-text body from the JSON content for indexing.
	if len(p.Content) > 0 {
		doc.Body = extractText(string(p.Content))
	}
	for _, t := range p.Tags {
		if doc.Tags != "" {
			doc.Tags += " "
		}
		doc.Tags += t.Name
	}
	return a.bleve.index.Index(p.ID.String(), doc)
}

// Delete removes a page from the search index.
func (a *SearchRepoAdapter) Delete(_ context.Context, pageID uuid.UUID) error {
	return a.bleve.index.Delete(pageID.String())
}

// Search executes a full-text query and returns matching page IDs with scores.
func (a *SearchRepoAdapter) Search(_ context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	query := bleve.NewMatchQuery(params.Query)
	search := bleve.NewSearchRequest(query)
	search.Size = params.Limit
	if search.Size == 0 {
		search.Size = 20
	}
	search.Fields = []string{"id", "title", "space_id", "tags", "status", "updated_at"}
	search.IncludeLocations = false

	start := time.Now()
	result, err := a.bleve.index.Search(search)
	if err != nil {
		return nil, fmt.Errorf("bleve: search: %w", err)
	}
	elapsed := time.Since(start).Milliseconds()

	out := &domain.SearchResult{
		QueryTimeMs: elapsed,
		Total:       int(result.Total),
	}

	for _, hit := range result.Hits {
		item := domain.SearchItem{
			PageID: hit.ID,
			Score:  hit.Score,
		}
		if v, ok := hit.Fields["title"]; ok {
			item.Title, _ = v.(string)
		}
		if v, ok := hit.Fields["space_id"]; ok {
			item.Space.ID, _ = v.(string)
		}
		out.Items = append(out.Items, item)
	}

	return out, nil
}

// extractText does a naive extraction of text content from a JSON AST string.
// It is used to build the plain-text body field for Bleve indexing.
func extractText(jsonStr string) string {
	// Strip JSON structure markers; keep only string content between quotes.
	// This is intentionally simple — a full AST walker would be more accurate
	// but this is sufficient for keyword matching in the POC.
	var sb strings.Builder
	inString := false
	escape := false
	for _, ch := range jsonStr {
		if escape {
			if inString {
				sb.WriteRune(ch)
			}
			escape = false
			continue
		}
		switch ch {
		case '\\':
			escape = true
		case '"':
			inString = !inString
		default:
			if inString {
				sb.WriteRune(ch)
			}
		}
	}
	return sb.String()
}
