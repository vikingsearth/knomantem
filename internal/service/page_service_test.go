package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// ---- stub / mock PageRepository ----

// stubPageRepo is a minimal in-memory PageRepository used by tests in this package.
type stubPageRepo struct {
	pages      map[uuid.UUID]*domain.Page
	versions   []*domain.PageVersion
	createFn   func(ctx context.Context, p *domain.Page) (*domain.Page, error)
	updateFn   func(ctx context.Context, p *domain.Page) (*domain.Page, error)
	deleteFn   func(ctx context.Context, id uuid.UUID) error
}

func newStubPageRepo() *stubPageRepo {
	return &stubPageRepo{pages: make(map[uuid.UUID]*domain.Page)}
}

func (m *stubPageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Page, error) {
	if p, ok := m.pages[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m *stubPageRepo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]*domain.Page, error) {
	var out []*domain.Page
	for _, p := range m.pages {
		if p.SpaceID == spaceID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *stubPageRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Page, error) {
	var out []*domain.Page
	for _, id := range ids {
		if p, ok := m.pages[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *stubPageRepo) Create(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	if m.pages == nil {
		m.pages = make(map[uuid.UUID]*domain.Page)
	}
	m.pages[p.ID] = p
	return p, nil
}

func (m *stubPageRepo) Update(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	m.pages[p.ID] = p
	return p, nil
}

func (m *stubPageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	if _, ok := m.pages[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.pages, id)
	return nil
}

func (m *stubPageRepo) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, position int) (*domain.Page, error) {
	p, ok := m.pages[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	p.ParentID = parentID
	p.Position = position
	return p, nil
}

func (m *stubPageRepo) ListVersions(ctx context.Context, pageID uuid.UUID) ([]*domain.PageVersion, error) {
	var out []*domain.PageVersion
	for _, v := range m.versions {
		if v.PageID == pageID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *stubPageRepo) GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*domain.PageVersion, error) {
	for _, v := range m.versions {
		if v.PageID == pageID && v.Version == version {
			return v, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *stubPageRepo) CreateVersion(ctx context.Context, v *domain.PageVersion) error {
	v.ID = uuid.New()
	m.versions = append(m.versions, v)
	return nil
}

// ---- stub SearchRepository ----

type stubSearchRepo struct {
	indexed []*domain.Page
	deleted []uuid.UUID
}

func (m *stubSearchRepo) Index(ctx context.Context, p *domain.Page) error {
	m.indexed = append(m.indexed, p)
	return nil
}

func (m *stubSearchRepo) Delete(ctx context.Context, pageID uuid.UUID) error {
	m.deleted = append(m.deleted, pageID)
	return nil
}

func (m *stubSearchRepo) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	return &domain.SearchResult{}, nil
}

// ---- stub EdgeRepository ----

type stubEdgeRepo struct{}

func (m *stubEdgeRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Edge, error) {
	return nil, domain.ErrNotFound
}
func (m *stubEdgeRepo) ListByPage(ctx context.Context, pageID uuid.UUID, direction string, edgeType string) ([]*domain.Edge, error) {
	return nil, nil
}
func (m *stubEdgeRepo) Create(ctx context.Context, e *domain.Edge) (*domain.Edge, error) {
	return e, nil
}
func (m *stubEdgeRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *stubEdgeRepo) Explore(ctx context.Context, rootID uuid.UUID, depth int, edgeType string, limit int) (*domain.GraphExploreResult, error) {
	return &domain.GraphExploreResult{}, nil
}

// ---- stub TagRepository ----

type stubTagRepo struct{}

func (m *stubTagRepo) List(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	return nil, nil
}
func (m *stubTagRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	return nil, domain.ErrNotFound
}
func (m *stubTagRepo) Create(ctx context.Context, t *domain.Tag) (*domain.Tag, error) {
	return t, nil
}
func (m *stubTagRepo) AddToPage(ctx context.Context, pageID uuid.UUID, assignments []domain.PageTagAssignment) ([]domain.TagWithScore, error) {
	return nil, nil
}
func (m *stubTagRepo) ListByPage(ctx context.Context, pageID uuid.UUID) ([]domain.Tag, error) {
	return nil, nil
}

// ---- helper constructors ----

func newPageSvc(pr *stubPageRepo, sr *stubSearchRepo) *PageService {
	return NewPageService(pr, sr, &stubEdgeRepo{}, &stubTagRepo{})
}

func makeContentJSON(text string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	})
	return raw
}

// ---- Create tests ----

func TestPageService_Create_Success(t *testing.T) {
	pr := newStubPageRepo()
	sr := &stubSearchRepo{}
	svc := newPageSvc(pr, sr)

	spaceID := uuid.New()
	userID := uuid.New()
	req := domain.CreatePageRequest{
		Title:   "Hello World",
		Content: makeContentJSON("some text"),
	}

	page, err := svc.Create(context.Background(), spaceID, userID.String(), req)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if page == nil {
		t.Fatal("expected non-nil page")
	}
	if page.Title != "Hello World" {
		t.Errorf("Title: got %q, want %q", page.Title, "Hello World")
	}
	if page.Slug != "hello-world" {
		t.Errorf("Slug: got %q, want %q", page.Slug, "hello-world")
	}
	if page.SpaceID != spaceID {
		t.Errorf("SpaceID mismatch")
	}
	if page.CreatedBy != userID {
		t.Errorf("CreatedBy mismatch")
	}
	if page.Version != 1 {
		t.Errorf("Version: got %d, want 1", page.Version)
	}
	if page.FreshnessStatus != domain.FreshnessFresh {
		t.Errorf("FreshnessStatus: got %q, want fresh", page.FreshnessStatus)
	}
	if page.Depth != 0 {
		t.Errorf("Depth: got %d, want 0 for root page", page.Depth)
	}

	// Verify search was indexed.
	if len(sr.indexed) == 0 {
		t.Error("expected page to be indexed in search")
	}
}

func TestPageService_Create_WithParent(t *testing.T) {
	pr := newStubPageRepo()
	sr := &stubSearchRepo{}
	svc := newPageSvc(pr, sr)

	spaceID := uuid.New()
	userID := uuid.New()

	// Create parent page first.
	parent := &domain.Page{
		ID:      uuid.New(),
		SpaceID: spaceID,
		Title:   "Parent",
		Slug:    "parent",
		Depth:   0,
		Version: 1,
	}
	pr.pages[parent.ID] = parent

	childReq := domain.CreatePageRequest{
		Title:    "Child",
		ParentID: &parent.ID,
	}
	child, err := svc.Create(context.Background(), spaceID, userID.String(), childReq)
	if err != nil {
		t.Fatalf("Create child: unexpected error: %v", err)
	}
	if child.Depth != 1 {
		t.Errorf("child Depth: got %d, want 1", child.Depth)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("child ParentID mismatch")
	}
}

func TestPageService_Create_ParentNotFound(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	spaceID := uuid.New()
	userID := uuid.New()
	nonExistentParent := uuid.New()
	req := domain.CreatePageRequest{
		Title:    "Orphan",
		ParentID: &nonExistentParent,
	}

	_, err := svc.Create(context.Background(), spaceID, userID.String(), req)
	if err == nil {
		t.Fatal("expected error for missing parent, got nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestPageService_Create_InvalidUserID(t *testing.T) {
	svc := newPageSvc(newStubPageRepo(), &stubSearchRepo{})
	_, err := svc.Create(context.Background(), uuid.New(), "not-a-uuid", domain.CreatePageRequest{Title: "X"})
	if err == nil {
		t.Fatal("expected error for invalid userID, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPageService_Create_RepoError(t *testing.T) {
	pr := newStubPageRepo()
	pr.createFn = func(ctx context.Context, p *domain.Page) (*domain.Page, error) {
		return nil, errors.New("db error")
	}
	svc := newPageSvc(pr, &stubSearchRepo{})

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New().String(), domain.CreatePageRequest{Title: "Fail"})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ---- Update tests ----

func TestPageService_Update_Success(t *testing.T) {
	pr := newStubPageRepo()
	sr := &stubSearchRepo{}
	svc := newPageSvc(pr, sr)

	spaceID := uuid.New()
	userID := uuid.New()
	page := &domain.Page{
		ID:      uuid.New(),
		SpaceID: spaceID,
		Title:   "Original Title",
		Slug:    "original-title",
		Content: makeContentJSON("original content"),
		Version: 1,
	}
	pr.pages[page.ID] = page

	newTitle := "Updated Title"
	newContent := makeContentJSON("updated content")
	updated, err := svc.Update(context.Background(), page.ID, userID.String(), domain.UpdatePageRequest{
		Title:         &newTitle,
		Content:       newContent,
		ChangeSummary: "Updated content",
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("Title: got %q, want %q", updated.Title, newTitle)
	}
	if updated.Version != 2 {
		t.Errorf("Version: got %d, want 2 (should be bumped)", updated.Version)
	}
	if updated.UpdatedBy != userID {
		t.Errorf("UpdatedBy mismatch")
	}

	// A version record should have been saved.
	if len(pr.versions) == 0 {
		t.Error("expected a PageVersion to be created on update")
	}
	ver := pr.versions[0]
	if ver.Version != 1 {
		t.Errorf("saved version: got %d, want 1 (the old version)", ver.Version)
	}
	if ver.Title != "Original Title" {
		t.Errorf("saved version title: got %q, want %q", ver.Title, "Original Title")
	}

	// Re-index should have been called.
	if len(sr.indexed) == 0 {
		t.Error("expected search to be re-indexed on update")
	}
}

func TestPageService_Update_TitleOnly(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	page := &domain.Page{
		ID:      uuid.New(),
		Title:   "Old",
		Content: makeContentJSON("keep this"),
		Version: 3,
	}
	pr.pages[page.ID] = page

	newTitle := "New Title"
	updated, err := svc.Update(context.Background(), page.ID, uuid.New().String(), domain.UpdatePageRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update (title only): %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("Title: got %q, want %q", updated.Title, newTitle)
	}
	if updated.Version != 4 {
		t.Errorf("Version should be bumped when title changes: got %d", updated.Version)
	}
}

func TestPageService_Update_ContentOnly(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	page := &domain.Page{
		ID:      uuid.New(),
		Title:   "Keep",
		Content: makeContentJSON("old"),
		Version: 1,
	}
	pr.pages[page.ID] = page

	updated, err := svc.Update(context.Background(), page.ID, uuid.New().String(), domain.UpdatePageRequest{
		Content: makeContentJSON("new content"),
	})
	if err != nil {
		t.Fatalf("Update (content only): %v", err)
	}
	if updated.Version != 2 {
		t.Errorf("Version should bump on content change: got %d", updated.Version)
	}
	if updated.Title != "Keep" {
		t.Errorf("Title should be unchanged: got %q", updated.Title)
	}
}

func TestPageService_Update_MetadataOnly_NoVersionBump(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	page := &domain.Page{
		ID:      uuid.New(),
		Title:   "Page",
		Content: makeContentJSON("body"),
		Version: 2,
	}
	pr.pages[page.ID] = page

	icon := "📄"
	cover := "https://example.com/cover.jpg"
	updated, err := svc.Update(context.Background(), page.ID, uuid.New().String(), domain.UpdatePageRequest{
		Icon:       &icon,
		CoverImage: &cover,
	})
	if err != nil {
		t.Fatalf("Update (metadata only): %v", err)
	}
	// Version should NOT be bumped when only icon/cover changes.
	if updated.Version != 2 {
		t.Errorf("Version should not change for metadata-only update: got %d", updated.Version)
	}
	if len(pr.versions) != 0 {
		t.Errorf("expected no version records for metadata-only update, got %d", len(pr.versions))
	}
}

func TestPageService_Update_InvalidUserID(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	page := &domain.Page{ID: uuid.New(), Title: "Page", Version: 1}
	pr.pages[page.ID] = page

	_, err := svc.Update(context.Background(), page.ID, "bad-uuid", domain.UpdatePageRequest{})
	if err == nil {
		t.Fatal("expected error for invalid userID, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestPageService_Update_NotFound(t *testing.T) {
	svc := newPageSvc(newStubPageRepo(), &stubSearchRepo{})
	title := "X"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New().String(), domain.UpdatePageRequest{Title: &title})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPageService_Update_RepoError(t *testing.T) {
	pr := newStubPageRepo()
	page := &domain.Page{ID: uuid.New(), Title: "Page", Version: 1}
	pr.pages[page.ID] = page
	pr.updateFn = func(ctx context.Context, p *domain.Page) (*domain.Page, error) {
		return nil, errors.New("db error")
	}
	svc := newPageSvc(pr, &stubSearchRepo{})

	title := "New"
	_, err := svc.Update(context.Background(), page.ID, uuid.New().String(), domain.UpdatePageRequest{Title: &title})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ---- Delete tests ----

func TestPageService_Delete_Success(t *testing.T) {
	pr := newStubPageRepo()
	sr := &stubSearchRepo{}
	svc := newPageSvc(pr, sr)

	page := &domain.Page{ID: uuid.New(), Title: "To Delete", Version: 1}
	pr.pages[page.ID] = page

	err := svc.Delete(context.Background(), page.ID, uuid.New().String())
	if err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Page should be removed from store.
	if _, ok := pr.pages[page.ID]; ok {
		t.Error("expected page to be removed from store")
	}

	// Search index entry should be removed.
	if len(sr.deleted) == 0 {
		t.Error("expected search entry to be deleted")
	}
	if sr.deleted[0] != page.ID {
		t.Errorf("search deleted ID: got %v, want %v", sr.deleted[0], page.ID)
	}
}

func TestPageService_Delete_NotFound(t *testing.T) {
	svc := newPageSvc(newStubPageRepo(), &stubSearchRepo{})
	err := svc.Delete(context.Background(), uuid.New(), uuid.New().String())
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPageService_Delete_RepoError(t *testing.T) {
	pr := newStubPageRepo()
	page := &domain.Page{ID: uuid.New(), Title: "Page", Version: 1}
	pr.pages[page.ID] = page
	pr.deleteFn = func(ctx context.Context, id uuid.UUID) error {
		return errors.New("db error")
	}
	svc := newPageSvc(pr, &stubSearchRepo{})

	err := svc.Delete(context.Background(), page.ID, uuid.New().String())
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ---- GetByID tests ----

func TestPageService_GetByID_Found(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	page := &domain.Page{ID: uuid.New(), Title: "My Page", Version: 1}
	pr.pages[page.ID] = page

	got, err := svc.GetByID(context.Background(), page.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}
	if got.ID != page.ID {
		t.Errorf("ID mismatch")
	}
}

func TestPageService_GetByID_NotFound(t *testing.T) {
	svc := newPageSvc(newStubPageRepo(), &stubSearchRepo{})
	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- ListBySpace tests ----

func TestPageService_ListBySpace(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	spaceID := uuid.New()
	otherSpaceID := uuid.New()

	for i := 0; i < 3; i++ {
		p := &domain.Page{ID: uuid.New(), SpaceID: spaceID, Depth: 0}
		pr.pages[p.ID] = p
	}
	// Page in a different space.
	other := &domain.Page{ID: uuid.New(), SpaceID: otherSpaceID, Depth: 0}
	pr.pages[other.ID] = other

	pages, err := svc.ListBySpace(context.Background(), spaceID, "", -1)
	if err != nil {
		t.Fatalf("ListBySpace: %v", err)
	}
	if len(pages) != 3 {
		t.Errorf("expected 3 pages for space, got %d", len(pages))
	}
}

func TestPageService_ListBySpace_MaxDepthFilter(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	spaceID := uuid.New()
	for depth := 0; depth <= 3; depth++ {
		p := &domain.Page{ID: uuid.New(), SpaceID: spaceID, Depth: depth}
		pr.pages[p.ID] = p
	}

	pages, err := svc.ListBySpace(context.Background(), spaceID, "", 1)
	if err != nil {
		t.Fatalf("ListBySpace (maxDepth=1): %v", err)
	}
	for _, p := range pages {
		if p.Depth > 1 {
			t.Errorf("page with depth %d should have been filtered out", p.Depth)
		}
	}
	if len(pages) != 2 {
		t.Errorf("expected 2 pages (depth 0 and 1), got %d", len(pages))
	}
}

// ---- ListVersions / GetVersion tests ----

func TestPageService_ListVersions(t *testing.T) {
	pr := newStubPageRepo()
	svc := newPageSvc(pr, &stubSearchRepo{})

	pageID := uuid.New()
	page := &domain.Page{
		ID:      pageID,
		Title:   "Versioned",
		Content: makeContentJSON("v1 content"),
		Version: 1,
	}
	pr.pages[pageID] = page

	// Create a few versions through Update.
	for i := 0; i < 3; i++ {
		content := makeContentJSON("v content")
		if _, err := svc.Update(context.Background(), pageID, uuid.New().String(), domain.UpdatePageRequest{Content: content}); err != nil {
			break
		}
	}

	versions, err := svc.ListVersions(context.Background(), pageID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) == 0 {
		t.Error("expected at least one version record")
	}
}

func TestPageService_GetVersion_NotFound(t *testing.T) {
	svc := newPageSvc(newStubPageRepo(), &stubSearchRepo{})
	_, err := svc.GetVersion(context.Background(), uuid.New(), 99)
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
