// Package search provides full-text search over Knomantem pages using Bleve.
//
// Documents are indexed with three weighted fields:
//   - title   (boost 3x)
//   - tags    (boost 2x)
//   - content (boost 1x)
//
// Additional filter fields (not ranked, but facet-able):
//   - space_id, freshness_status, created_at, updated_at
package search

import (
	"fmt"
	"os"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/en"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
)

// IndexDoc is the document shape stored in Bleve.
type IndexDoc struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	Tags             []string `json:"tags"`
	SpaceID          string   `json:"space_id"`
	FreshnessStatus  string   `json:"freshness_status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// SearchFilters carries optional filter values applied as conjunctions.
type SearchFilters struct {
	SpaceID         string
	Tags            []string
	FreshnessStatus string
	// DateFrom / DateTo filter on updated_at (RFC3339 strings)
	DateFrom string
	DateTo   string
}

// SearchResult is the structured return value of Search.
type SearchResult struct {
	IDs    []string
	Total  uint64
	Facets map[string][]FacetCount
}

// FacetCount is a single bucket in a facet result.
type FacetCount struct {
	Name  string
	Count int
}

// BleveSearch manages a Bleve index for page search.
type BleveSearch struct {
	index bleve.Index
	path  string
}

// NewBleveIndex opens (or creates) a Bleve index at the given path.
func NewBleveIndex(path string) (*BleveSearch, error) {
	idx, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		m, mErr := buildMapping()
		if mErr != nil {
			return nil, fmt.Errorf("bleve: build mapping: %w", mErr)
		}
		idx, err = bleve.New(path, m)
		if err != nil {
			return nil, fmt.Errorf("bleve: create index at %s: %w", path, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("bleve: open index at %s: %w", path, err)
	}
	return &BleveSearch{index: idx, path: path}, nil
}

// Index adds or updates a document in the search index.
// doc must be a map[string]any containing at minimum an "id" key.
func (b *BleveSearch) Index(id string, doc map[string]any) error {
	if id == "" {
		return fmt.Errorf("bleve index: id must not be empty")
	}
	return b.index.Index(id, flattenDoc(id, doc))
}

// Delete removes a document from the search index by its ID.
func (b *BleveSearch) Delete(id string) error {
	return b.index.Delete(id)
}

// Search executes a full-text query with optional filters, sorting, and pagination.
//
// Parameters:
//   - queryStr: full-text query string (may be empty for match-all)
//   - filters:  optional structured filters
//   - sortBy:   field name to sort by (prefix with "-" for descending), or "" for relevance
//   - from:     pagination offset
//   - size:     maximum number of results
//
// Returns:
//   - ids:    ordered list of matching document IDs
//   - total:  total number of matches (before pagination)
//   - facets: map of facet name -> []FacetCount
//   - error
func (b *BleveSearch) Search(queryStr string, filters SearchFilters, sortBy string, from, size int) ([]string, uint64, map[string][]FacetCount, error) {
	bq := buildQuery(queryStr, filters)

	req := bleve.NewSearchRequestOptions(bq, size, from, false)
	req.Fields = []string{"id"}

	// Add facets
	req.AddFacet("space_id", bleve.NewFacetRequest("space_id", 20))
	req.AddFacet("freshness_status", bleve.NewFacetRequest("freshness_status", 10))
	req.AddFacet("tags", bleve.NewFacetRequest("tags", 50))

	// Sorting
	if sortBy != "" {
		req.SortBy([]string{sortBy})
	}

	res, err := b.index.Search(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("bleve search: %w", err)
	}

	ids := make([]string, 0, len(res.Hits))
	for _, hit := range res.Hits {
		ids = append(ids, hit.ID)
	}

	facets := extractFacets(res.Facets)
	return ids, res.Total, facets, nil
}

// Rebuild drops and recreates the index, then bulk-indexes the supplied documents.
// Each element of docs must be a map[string]any containing an "id" key.
func (b *BleveSearch) Rebuild(docs []map[string]any) error {
	// Close current index and delete the directory
	if err := b.index.Close(); err != nil {
		return fmt.Errorf("bleve rebuild: close: %w", err)
	}
	if err := os.RemoveAll(b.path); err != nil {
		return fmt.Errorf("bleve rebuild: remove: %w", err)
	}

	// Recreate
	m, err := buildMapping()
	if err != nil {
		return fmt.Errorf("bleve rebuild: mapping: %w", err)
	}
	idx, err := bleve.New(b.path, m)
	if err != nil {
		return fmt.Errorf("bleve rebuild: create: %w", err)
	}
	b.index = idx

	// Bulk index using a batch for performance
	batch := idx.NewBatch()
	for _, doc := range docs {
		id, ok := doc["id"].(string)
		if !ok || id == "" {
			continue
		}
		if err := batch.Index(id, flattenDoc(id, doc)); err != nil {
			return fmt.Errorf("bleve rebuild: batch index %s: %w", id, err)
		}
	}
	return idx.Batch(batch)
}

// Close closes the underlying Bleve index.
func (b *BleveSearch) Close() error {
	return b.index.Close()
}

// buildMapping constructs the Bleve index mapping with boosted fields.
func buildMapping() (mapping.IndexMapping, error) {
	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = en.AnalyzerName

	// --- field mappings ---

	textField := bleve.NewTextFieldMapping()
	textField.Analyzer = en.AnalyzerName
	textField.Store = false

	keywordField := bleve.NewTextFieldMapping()
	keywordField.Analyzer = "keyword"
	keywordField.Store = true

	dateField := bleve.NewDateTimeFieldMapping()
	dateField.Store = false

	// --- document mapping ---
	dm := bleve.NewDocumentMapping()

	dm.AddFieldMappingsAt("title", textField)
	dm.AddFieldMappingsAt("content", textField)
	dm.AddFieldMappingsAt("tags", textField)
	dm.AddFieldMappingsAt("space_id", keywordField)
	dm.AddFieldMappingsAt("freshness_status", keywordField)
	dm.AddFieldMappingsAt("created_at", dateField)
	dm.AddFieldMappingsAt("updated_at", dateField)

	im.DefaultMapping = dm
	return im, nil
}

// buildQuery assembles the Bleve query from a free-text string and structured filters.
func buildQuery(queryStr string, filters SearchFilters) query.Query {
	var must []query.Query

	// Full-text component with field boosting
	if strings.TrimSpace(queryStr) != "" {
		titleQ := bleve.NewMatchQuery(queryStr)
		titleQ.SetField("title")
		titleQ.SetBoost(3)

		tagsQ := bleve.NewMatchQuery(queryStr)
		tagsQ.SetField("tags")
		tagsQ.SetBoost(2)

		contentQ := bleve.NewMatchQuery(queryStr)
		contentQ.SetField("content")
		contentQ.SetBoost(1)

		boolQ := bleve.NewBooleanQuery()
		boolQ.AddShould(titleQ)
		boolQ.AddShould(tagsQ)
		boolQ.AddShould(contentQ)

		must = append(must, boolQ)
	}

	// Space filter
	if filters.SpaceID != "" {
		q := bleve.NewTermQuery(filters.SpaceID)
		q.SetField("space_id")
		must = append(must, q)
	}

	// Freshness status filter
	if filters.FreshnessStatus != "" {
		q := bleve.NewTermQuery(filters.FreshnessStatus)
		q.SetField("freshness_status")
		must = append(must, q)
	}

	// Tags filter (any of the given tags)
	if len(filters.Tags) > 0 {
		tagBool := bleve.NewBooleanQuery()
		for _, tag := range filters.Tags {
			tq := bleve.NewTermQuery(tag)
			tq.SetField("tags")
			tagBool.AddShould(tq)
		}
		must = append(must, tagBool)
	}

	// Date range filter on updated_at
	if filters.DateFrom != "" || filters.DateTo != "" {
		drq := bleve.NewDateRangeStringQuery(filters.DateFrom, filters.DateTo)
		drq.SetField("updated_at")
		must = append(must, drq)
	}

	if len(must) == 0 {
		return bleve.NewMatchAllQuery()
	}
	if len(must) == 1 {
		return must[0]
	}
	bq := bleve.NewBooleanQuery()
	for _, q := range must {
		bq.AddMust(q)
	}
	return bq
}

// flattenDoc normalises a raw map[string]any into an IndexDoc.
func flattenDoc(id string, raw map[string]any) IndexDoc {
	d := IndexDoc{ID: id}
	if v, ok := raw["title"].(string); ok {
		d.Title = v
	}
	if v, ok := raw["content"].(string); ok {
		d.Content = v
	}
	if v, ok := raw["space_id"].(string); ok {
		d.SpaceID = v
	}
	if v, ok := raw["freshness_status"].(string); ok {
		d.FreshnessStatus = v
	}
	if v, ok := raw["created_at"].(string); ok {
		d.CreatedAt = v
	}
	if v, ok := raw["updated_at"].(string); ok {
		d.UpdatedAt = v
	}
	// Tags can be []string or []any
	switch tv := raw["tags"].(type) {
	case []string:
		d.Tags = tv
	case []any:
		for _, t := range tv {
			if s, ok := t.(string); ok {
				d.Tags = append(d.Tags, s)
			}
		}
	}
	return d
}

// extractFacets converts Bleve's FacetResults into our FacetCount type.
func extractFacets(fr bleveSearch.FacetResults) map[string][]FacetCount {
	result := make(map[string][]FacetCount, len(fr))
	for name, f := range fr {
		if f == nil {
			continue
		}
		var counts []FacetCount
		if f.Terms != nil {
			for _, t := range f.Terms.Terms() {
				counts = append(counts, FacetCount{Name: t.Term, Count: t.Count})
			}
		}
		result[name] = counts
	}
	return result
}
