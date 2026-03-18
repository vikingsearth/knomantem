package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
	"github.com/knomantem/knomantem/pkg/markdown"
)

// PageService handles page business logic.
type PageService struct {
	pages  domain.PageRepository
	search domain.SearchRepository
	edges  domain.EdgeRepository
	tags   domain.TagRepository
}

// NewPageService creates a new PageService.
func NewPageService(
	pages domain.PageRepository,
	search domain.SearchRepository,
	edges domain.EdgeRepository,
	tags domain.TagRepository,
) *PageService {
	return &PageService{pages: pages, search: search, edges: edges, tags: tags}
}

// ListBySpace returns all pages in a space ordered for tree rendering.
func (s *PageService) ListBySpace(ctx context.Context, spaceID uuid.UUID, format string, maxDepth int) ([]*domain.Page, error) {
	pages, err := s.pages.ListBySpace(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	if maxDepth >= 0 {
		filtered := pages[:0]
		for _, p := range pages {
			if p.Depth <= maxDepth {
				filtered = append(filtered, p)
			}
		}
		pages = filtered
	}
	return pages, nil
}

// Create creates a new page in a space, indexes it, and returns the persisted entity.
func (s *PageService) Create(ctx context.Context, spaceID uuid.UUID, userID string, req domain.CreatePageRequest) (*domain.Page, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	depth := 0
	if req.ParentID != nil {
		parent, err := s.pages.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("%w: parent page not found", domain.ErrValidation)
		}
		depth = parent.Depth + 1
	}

	p := &domain.Page{
		ID:              uuid.New(),
		SpaceID:         spaceID,
		ParentID:        req.ParentID,
		Title:           req.Title,
		Slug:            slugify(req.Title),
		Content:         req.Content,
		Icon:            req.Icon,
		Position:        req.Position,
		Depth:           depth,
		IsTemplate:      req.IsTemplate,
		FreshnessStatus: domain.FreshnessFresh,
		Version:         1,
		CreatedBy:       uid,
		UpdatedBy:       uid,
	}

	created, err := s.pages.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Index asynchronously — errors are non-fatal.
	_ = s.search.Index(ctx, created)

	return created, nil
}

// GetByID returns a page with its full content.
func (s *PageService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Page, error) {
	return s.pages.GetByID(ctx, id)
}

// Update updates a page, bumps its version, and re-indexes it.
func (s *PageService) Update(ctx context.Context, id uuid.UUID, userID string, req domain.UpdatePageRequest) (*domain.Page, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	p, err := s.pages.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Save old version before updating.
	if len(req.Content) > 0 || req.Title != nil {
		ver := &domain.PageVersion{
			PageID:        p.ID,
			Version:       p.Version,
			Title:         p.Title,
			Content:       p.Content,
			ChangeSummary: req.ChangeSummary,
			CreatedBy:     uid,
		}
		_ = s.pages.CreateVersion(ctx, ver)
		p.Version++
	}

	if req.Title != nil {
		p.Title = *req.Title
	}
	if len(req.Content) > 0 {
		p.Content = req.Content
	}
	if req.Icon != nil {
		p.Icon = req.Icon
	}
	if req.CoverImage != nil {
		p.CoverImage = req.CoverImage
	}
	p.UpdatedBy = uid

	updated, err := s.pages.Update(ctx, p)
	if err != nil {
		return nil, err
	}

	_ = s.search.Index(ctx, updated)
	return updated, nil
}

// Delete removes a page from the store and the search index.
func (s *PageService) Delete(ctx context.Context, id uuid.UUID, userID string) error {
	if _, err := s.pages.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.pages.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.search.Delete(ctx, id)
	return nil
}

// Move relocates a page in the tree.
func (s *PageService) Move(ctx context.Context, id uuid.UUID, userID string, parentID *uuid.UUID, position int) (*domain.Page, error) {
	return s.pages.Move(ctx, id, parentID, position)
}

// ListVersions returns all versions of a page.
func (s *PageService) ListVersions(ctx context.Context, pageID uuid.UUID) ([]*domain.PageVersion, error) {
	return s.pages.ListVersions(ctx, pageID)
}

// GetVersion returns a specific version of a page.
func (s *PageService) GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*domain.PageVersion, error) {
	return s.pages.GetVersion(ctx, pageID, version)
}

// ImportMarkdown parses Markdown into the JSON AST and updates the page content.
func (s *PageService) ImportMarkdown(ctx context.Context, pageID uuid.UUID, userID string, md string) (*domain.Page, error) {
	ast, err := markdown.ImportMarkdown(md)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrValidation, err.Error())
	}

	content, err := json.Marshal(ast)
	if err != nil {
		return nil, fmt.Errorf("page: marshal content: %w", err)
	}

	// Extract title from the first H1 node if present.
	title := extractTitle(ast)

	req := domain.UpdatePageRequest{
		Content:       json.RawMessage(content),
		ChangeSummary: "Imported from Markdown",
	}
	if title != "" {
		req.Title = &title
	}

	return s.Update(ctx, pageID, userID, req)
}

// ExportMarkdown converts a page's JSON AST content to Markdown.
func (s *PageService) ExportMarkdown(ctx context.Context, pageID uuid.UUID) (string, error) {
	p, err := s.pages.GetByID(ctx, pageID)
	if err != nil {
		return "", err
	}
	if len(p.Content) == 0 {
		return "", nil
	}
	return jsonASTToMarkdown(p.Content), nil
}

// extractTitle attempts to pull the text of the first H1 from the AST.
func extractTitle(ast map[string]any) string {
	content, _ := ast["content"].([]any)
	for _, node := range content {
		m, ok := node.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == "heading" {
			if attrs, ok := m["attrs"].(map[string]any); ok {
				if level, ok := attrs["level"].(int); ok && level == 1 {
					return extractInlineText(m)
				}
				// goldmark returns float64 for JSON numbers
				if level, ok := attrs["level"].(float64); ok && int(level) == 1 {
					return extractInlineText(m)
				}
			}
		}
	}
	return ""
}

func extractInlineText(node map[string]any) string {
	children, _ := node["content"].([]any)
	var sb strings.Builder
	for _, c := range children {
		if m, ok := c.(map[string]any); ok {
			if t, ok := m["text"].(string); ok {
				sb.WriteString(t)
			}
		}
	}
	return sb.String()
}

// jsonASTToMarkdown is a simplified converter from the JSON AST to Markdown.
// A full implementation lives in pkg/markdown; this is a minimal fallback.
func jsonASTToMarkdown(raw json.RawMessage) string {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	var sb strings.Builder
	content, _ := doc["content"].([]any)
	for _, node := range content {
		if m, ok := node.(map[string]any); ok {
			renderBlock(&sb, m, 0)
		}
	}
	return sb.String()
}

func renderBlock(sb *strings.Builder, node map[string]any, indent int) {
	typ, _ := node["type"].(string)
	children, _ := node["content"].([]any)

	switch typ {
	case "heading":
		level := 1
		if attrs, ok := node["attrs"].(map[string]any); ok {
			if l, ok := attrs["level"].(float64); ok {
				level = int(l)
			}
		}
		sb.WriteString(strings.Repeat("#", level) + " ")
		renderInlineChildren(sb, children)
		sb.WriteString("\n\n")
	case "paragraph":
		renderInlineChildren(sb, children)
		sb.WriteString("\n\n")
	case "code_block":
		lang := ""
		if attrs, ok := node["attrs"].(map[string]any); ok {
			if l, ok := attrs["language"].(string); ok {
				lang = l
			}
		}
		sb.WriteString("```" + lang + "\n")
		renderInlineChildren(sb, children)
		sb.WriteString("\n```\n\n")
	case "bullet_list":
		for _, item := range children {
			if m, ok := item.(map[string]any); ok {
				sb.WriteString("- ")
				itemContent, _ := m["content"].([]any)
				for _, ic := range itemContent {
					if im, ok := ic.(map[string]any); ok {
						renderBlock(sb, im, indent+2)
					}
				}
			}
		}
		sb.WriteString("\n")
	case "blockquote":
		for _, child := range children {
			if m, ok := child.(map[string]any); ok {
				sb.WriteString("> ")
				renderBlock(sb, m, 0)
			}
		}
	}
}

func renderInlineChildren(sb *strings.Builder, nodes []any) {
	for _, n := range nodes {
		if m, ok := n.(map[string]any); ok {
			text, _ := m["text"].(string)
			marks, _ := m["marks"].([]any)

			prefix, suffix := "", ""
			for _, mark := range marks {
				if mm, ok := mark.(map[string]any); ok {
					switch mm["type"] {
					case "bold":
						prefix += "**"
						suffix = "**" + suffix
					case "italic":
						prefix += "_"
						suffix = "_" + suffix
					case "code":
						prefix += "`"
						suffix = "`" + suffix
					case "strike":
						prefix += "~~"
						suffix = "~~" + suffix
					}
				}
			}
			sb.WriteString(prefix + text + suffix)
		}
	}
}
