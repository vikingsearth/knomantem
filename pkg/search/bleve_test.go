package search_test

import (
	"testing"

	"github.com/knomantem/knomantem/pkg/search"
)

// newIndex creates a temporary BleveSearch for a test.
func newIndex(t *testing.T) *search.BleveSearch {
	t.Helper()
	dir := t.TempDir()
	idx, err := search.NewBleveIndex(dir)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	return idx
}

func TestNewBleveIndex_CreatesIndex(t *testing.T) {
	idx := newIndex(t)
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
}

func TestNewBleveIndex_OpensExisting(t *testing.T) {
	dir := t.TempDir()
	idx, err := search.NewBleveIndex(dir)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = idx.Close()

	// Re-open the same path
	idx2, err := search.NewBleveIndex(dir)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	_ = idx2.Close()
}

func TestBleveSearch_IndexAndSearch(t *testing.T) {
	idx := newIndex(t)

	doc := map[string]any{
		"id":               "page-1",
		"title":            "Getting Started with Go",
		"content":          "This guide covers the basics of the Go programming language.",
		"tags":             []string{"go", "tutorial"},
		"space_id":         "space-a",
		"freshness_status": "fresh",
	}
	if err := idx.Index("page-1", doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	ids, total, _, err := idx.Search("Go programming", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total == 0 {
		t.Error("expected at least one result")
	}
	if len(ids) == 0 {
		t.Error("expected ids to be non-empty")
	}
	if ids[0] != "page-1" {
		t.Errorf("expected page-1, got %q", ids[0])
	}
}

func TestBleveSearch_EmptyQueryReturnsAll(t *testing.T) {
	idx := newIndex(t)

	for _, id := range []string{"a", "b", "c"} {
		doc := map[string]any{
			"id":      id,
			"title":   "Page " + id,
			"content": "content for " + id,
		}
		if err := idx.Index(id, doc); err != nil {
			t.Fatalf("Index %s: %v", id, err)
		}
	}

	ids, total, _, err := idx.Search("", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d", len(ids))
	}
}

func TestBleveSearch_Delete(t *testing.T) {
	idx := newIndex(t)

	doc := map[string]any{
		"id":      "del-1",
		"title":   "To be deleted",
		"content": "Some content here",
	}
	if err := idx.Index("del-1", doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	if err := idx.Delete("del-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	ids, total, _, err := idx.Search("deleted", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	for _, id := range ids {
		if id == "del-1" {
			t.Error("deleted document should not appear in results")
		}
	}
	_ = total
}

func TestBleveSearch_FilterBySpaceID(t *testing.T) {
	idx := newIndex(t)

	docs := []map[string]any{
		{"id": "s1p1", "title": "Page in space A", "content": "alpha content", "space_id": "space-a"},
		{"id": "s2p1", "title": "Page in space B", "content": "beta content", "space_id": "space-b"},
		{"id": "s1p2", "title": "Another in A", "content": "more alpha", "space_id": "space-a"},
	}
	for _, d := range docs {
		if err := idx.Index(d["id"].(string), d); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	ids, total, _, err := idx.Search("", search.SearchFilters{SpaceID: "space-a"}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 results for space-a, got %d", total)
	}
	for _, id := range ids {
		if id == "s2p1" {
			t.Error("space-b document should not appear in space-a filter")
		}
	}
}

func TestBleveSearch_FilterByFreshnessStatus(t *testing.T) {
	idx := newIndex(t)

	docs := []map[string]any{
		{"id": "f1", "title": "Fresh page", "content": "current", "freshness_status": "fresh"},
		{"id": "f2", "title": "Stale page", "content": "outdated", "freshness_status": "stale"},
		{"id": "f3", "title": "Aging page", "content": "aging", "freshness_status": "aging"},
	}
	for _, d := range docs {
		if err := idx.Index(d["id"].(string), d); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	ids, total, _, err := idx.Search("", search.SearchFilters{FreshnessStatus: "stale"}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 stale result, got %d", total)
	}
	if len(ids) == 0 || ids[0] != "f2" {
		t.Errorf("expected stale page f2, got %v", ids)
	}
}

func TestBleveSearch_Pagination(t *testing.T) {
	idx := newIndex(t)

	for i := 0; i < 5; i++ {
		id := "pg-" + string(rune('a'+i))
		doc := map[string]any{
			"id":      id,
			"title":   "Pagination test " + string(rune('A'+i)),
			"content": "content for pagination",
		}
		if err := idx.Index(id, doc); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	// First page
	ids1, total1, _, err := idx.Search("pagination", search.SearchFilters{}, "", 0, 2)
	if err != nil {
		t.Fatalf("Search page 1: %v", err)
	}
	if total1 != 5 {
		t.Errorf("expected total=5, got %d", total1)
	}
	if len(ids1) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(ids1))
	}

	// Second page
	ids2, _, _, err := idx.Search("pagination", search.SearchFilters{}, "", 2, 2)
	if err != nil {
		t.Fatalf("Search page 2: %v", err)
	}
	if len(ids2) != 2 {
		t.Errorf("expected 2 results on page 2, got %d", len(ids2))
	}

	// No overlap between pages
	set1 := make(map[string]bool)
	for _, id := range ids1 {
		set1[id] = true
	}
	for _, id := range ids2 {
		if set1[id] {
			t.Errorf("document %q appeared on both pages", id)
		}
	}
}

func TestBleveSearch_FacetsReturned(t *testing.T) {
	idx := newIndex(t)

	docs := []map[string]any{
		{"id": "fac1", "title": "Doc One", "content": "text", "space_id": "space-x", "freshness_status": "fresh", "tags": []string{"go", "api"}},
		{"id": "fac2", "title": "Doc Two", "content": "text", "space_id": "space-x", "freshness_status": "stale", "tags": []string{"go"}},
		{"id": "fac3", "title": "Doc Three", "content": "text", "space_id": "space-y", "freshness_status": "fresh", "tags": []string{"api"}},
	}
	for _, d := range docs {
		if err := idx.Index(d["id"].(string), d); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	_, _, facets, err := idx.Search("", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if facets == nil {
		t.Fatal("expected non-nil facets")
	}
	if _, ok := facets["space_id"]; !ok {
		t.Error("expected space_id facet")
	}
	if _, ok := facets["freshness_status"]; !ok {
		t.Error("expected freshness_status facet")
	}
}

func TestBleveSearch_Rebuild(t *testing.T) {
	dir := t.TempDir()
	idx, err := search.NewBleveIndex(dir)
	if err != nil {
		t.Fatalf("NewBleveIndex: %v", err)
	}

	// Index some documents
	for _, id := range []string{"old-1", "old-2"} {
		doc := map[string]any{"id": id, "title": "Old doc", "content": "stale"}
		if err := idx.Index(id, doc); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	// Rebuild with different documents
	newDocs := []map[string]any{
		{"id": "new-1", "title": "Rebuilt document alpha", "content": "fresh content"},
		{"id": "new-2", "title": "Rebuilt document beta", "content": "fresh content"},
	}
	if err := idx.Rebuild(newDocs); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	t.Cleanup(func() { _ = idx.Close() })

	// Old docs should not be found
	ids, total, _, err := idx.Search("stale", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search after rebuild: %v", err)
	}
	_ = total
	for _, id := range ids {
		if id == "old-1" || id == "old-2" {
			t.Errorf("old document %q should not exist after rebuild", id)
		}
	}

	// New docs should be found
	ids2, total2, _, err := idx.Search("Rebuilt", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search new docs: %v", err)
	}
	if total2 != 2 {
		t.Errorf("expected 2 rebuilt docs, got %d", total2)
	}
	_ = ids2
}

func TestBleveSearch_IndexEmptyIDError(t *testing.T) {
	idx := newIndex(t)
	err := idx.Index("", map[string]any{"title": "no id"})
	if err == nil {
		t.Error("expected error when indexing with empty id")
	}
}

func TestBleveSearch_TitleBoosted(t *testing.T) {
	idx := newIndex(t)

	// "golang" only in title of doc-1, only in content of doc-2
	docs := []map[string]any{
		{"id": "title-match", "title": "golang tutorial", "content": "programming guide"},
		{"id": "content-match", "title": "programming guide", "content": "golang basics"},
	}
	for _, d := range docs {
		if err := idx.Index(d["id"].(string), d); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	ids, total, _, err := idx.Search("golang", search.SearchFilters{}, "", 0, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 results, got %d", total)
	}
	if len(ids) == 0 {
		t.Fatal("expected results")
	}
	// title match should rank first due to boost
	if ids[0] != "title-match" {
		t.Errorf("expected title-match to rank first, got %q", ids[0])
	}
}
