// Package search provides Bleve-backed full-text search for Knomantem.
package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// BleveIndex wraps a bleve.Index and provides search operations.
type BleveIndex struct {
	index   bleve.Index
	idxPath string
}

// NewBleveIndex opens (or creates) the Bleve index at the given path.
// A subdirectory "index.bleve" is used inside path so that TempDir works.
func NewBleveIndex(path string) (*BleveIndex, error) {
	idxPath := filepath.Join(path, "index.bleve")
	index, err := bleve.Open(idxPath)
	if err != nil {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(idxPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("bleve: create index at %s: %w", idxPath, err)
		}
	}
	return &BleveIndex{index: index, idxPath: idxPath}, nil
}

// Close closes the underlying Bleve index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}

// SearchFilters carries optional filter values for direct searches.
type SearchFilters struct {
	SpaceID         string
	Tags            []string
	FreshnessStatus string
	DateFrom        string
	DateTo          string
}

// FacetCount is a single bucket in a facet result.
type FacetCount struct {
	Name  string
	Count int
}

// Index adds or updates a document in the search index by ID.
func (b *BleveIndex) Index(id string, doc map[string]any) error {
	if id == "" {
		return fmt.Errorf("bleve index: id must not be empty")
	}
	return b.index.Index(id, doc)
}

// Delete removes a document from the search index by ID.
func (b *BleveIndex) Delete(id string) error {
	return b.index.Delete(id)
}

// Search executes a full-text query with optional filters and returns IDs, total, facets.
func (b *BleveIndex) Search(queryStr string, filters SearchFilters, sortBy string, from, size int) ([]string, uint64, map[string][]FacetCount, error) {
	var baseQ *bleve.SearchRequest

	if strings.TrimSpace(queryStr) == "" {
		baseQ = bleve.NewSearchRequestOptions(bleve.NewMatchAllQuery(), size, from, false)
	} else {
		// Build conjunction: text query + filters
		mq := bleve.NewMatchQuery(queryStr)
		if filters.SpaceID == "" && filters.FreshnessStatus == "" {
			baseQ = bleve.NewSearchRequestOptions(mq, size, from, false)
		} else {
			cq := bleve.NewConjunctionQuery(mq)
			if filters.SpaceID != "" {
				tq := bleve.NewMatchPhraseQuery(filters.SpaceID)
				tq.SetField("space_id")
				cq.AddQuery(tq)
			}
			if filters.FreshnessStatus != "" {
				tq := bleve.NewMatchPhraseQuery(filters.FreshnessStatus)
				tq.SetField("freshness_status")
				cq.AddQuery(tq)
			}
			baseQ = bleve.NewSearchRequestOptions(cq, size, from, false)
		}
	}

	// Handle filter-only (no text query)
	if strings.TrimSpace(queryStr) == "" && (filters.SpaceID != "" || filters.FreshnessStatus != "") {
		cq := bleve.NewConjunctionQuery(bleve.NewMatchAllQuery())
		if filters.SpaceID != "" {
			tq := bleve.NewMatchPhraseQuery(filters.SpaceID)
			tq.SetField("space_id")
			cq.AddQuery(tq)
		}
		if filters.FreshnessStatus != "" {
			tq := bleve.NewMatchPhraseQuery(filters.FreshnessStatus)
			tq.SetField("freshness_status")
			cq.AddQuery(tq)
		}
		baseQ = bleve.NewSearchRequestOptions(cq, size, from, false)
	}

	req := baseQ
	req.Fields = []string{"*"}
	req.AddFacet("space_id", bleve.NewFacetRequest("space_id", 20))
	req.AddFacet("freshness_status", bleve.NewFacetRequest("freshness_status", 10))
	req.AddFacet("tags", bleve.NewFacetRequest("tags", 50))

	result, err := b.index.Search(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("bleve search: %w", err)
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		ids = append(ids, hit.ID)
	}

	facets := make(map[string][]FacetCount)
	for name, fr := range result.Facets {
		if fr.Terms != nil {
			for _, t := range fr.Terms.Terms() {
				facets[name] = append(facets[name], FacetCount{Name: t.Term, Count: t.Count})
			}
		}
	}

	return ids, result.Total, facets, nil
}

// Rebuild closes the current index, deletes it, creates a fresh one, and indexes the provided documents.
func (b *BleveIndex) Rebuild(docs []map[string]any) error {
	if err := b.index.Close(); err != nil {
		return fmt.Errorf("bleve rebuild: close: %w", err)
	}
	if err := os.RemoveAll(b.idxPath); err != nil {
		return fmt.Errorf("bleve rebuild: remove: %w", err)
	}
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.New(b.idxPath, mapping)
	if err != nil {
		return fmt.Errorf("bleve rebuild: create: %w", err)
	}
	b.index = idx

	batch := idx.NewBatch()
	for _, doc := range docs {
		id, _ := doc["id"].(string)
		if id == "" {
			continue
		}
		batch.Index(id, doc)
	}
	return idx.Batch(batch)
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
