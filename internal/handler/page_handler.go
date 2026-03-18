package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// PageService is the interface the PageHandler depends on.
type PageService interface {
	ListBySpace(ctx context.Context, spaceID uuid.UUID, format string, maxDepth int) ([]*domain.Page, error)
	Create(ctx context.Context, spaceID uuid.UUID, userID string, req domain.CreatePageRequest) (*domain.Page, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Page, error)
	Update(ctx context.Context, id uuid.UUID, userID string, req domain.UpdatePageRequest) (*domain.Page, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
	Move(ctx context.Context, id uuid.UUID, userID string, parentID *uuid.UUID, position int) (*domain.Page, error)
	ListVersions(ctx context.Context, pageID uuid.UUID) ([]*domain.PageVersion, error)
	GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*domain.PageVersion, error)
	ImportMarkdown(ctx context.Context, pageID uuid.UUID, userID string, markdown string) (*domain.Page, error)
	ExportMarkdown(ctx context.Context, pageID uuid.UUID) (string, error)
}

// PageHandler handles /api/v1/spaces/:spaceId/pages and /api/v1/pages/* endpoints.
type PageHandler struct {
	svc PageService
}

// NewPageHandler creates a new PageHandler.
func NewPageHandler(svc PageService) *PageHandler {
	return &PageHandler{svc: svc}
}

// pageListItem is a slim page representation used in tree/list responses.
type pageListItem struct {
	ID              string          `json:"id"`
	SpaceID         string          `json:"space_id"`
	ParentID        *string         `json:"parent_id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Icon            *string         `json:"icon"`
	Position        int             `json:"position"`
	Depth           int             `json:"depth"`
	IsTemplate      bool            `json:"is_template"`
	HasChildren     bool            `json:"has_children"`
	FreshnessStatus string          `json:"freshness_status"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

// pageDetailResponse is the full page representation including content.
type pageDetailResponse struct {
	ID              string          `json:"id"`
	SpaceID         string          `json:"space_id"`
	ParentID        *string         `json:"parent_id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Content         json.RawMessage `json:"content,omitempty"`
	Icon            *string         `json:"icon"`
	CoverImage      *string         `json:"cover_image"`
	Position        int             `json:"position"`
	Depth           int             `json:"depth"`
	IsTemplate      bool            `json:"is_template"`
	CreatedBy       string          `json:"created_by"`
	UpdatedBy       string          `json:"updated_by"`
	Freshness       *freshnessInfo  `json:"freshness,omitempty"`
	Tags            []tagRef        `json:"tags,omitempty"`
	Version         int             `json:"version"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type freshnessInfo struct {
	Score           float64 `json:"score"`
	Status          string  `json:"status"`
	LastVerifiedAt  *string `json:"last_verified_at,omitempty"`
	NextReviewAt    *string `json:"next_review_at,omitempty"`
}

type tagRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func toPageListItem(p *domain.Page) pageListItem {
	item := pageListItem{
		ID:              p.ID.String(),
		SpaceID:         p.SpaceID.String(),
		Title:           p.Title,
		Slug:            p.Slug,
		Icon:            p.Icon,
		Position:        p.Position,
		Depth:           p.Depth,
		IsTemplate:      p.IsTemplate,
		HasChildren:     p.HasChildren,
		FreshnessStatus: string(p.FreshnessStatus),
		CreatedAt:       p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if p.ParentID != nil {
		s := p.ParentID.String()
		item.ParentID = &s
	}
	return item
}

func toPageDetail(p *domain.Page) pageDetailResponse {
	detail := pageDetailResponse{
		ID:         p.ID.String(),
		SpaceID:    p.SpaceID.String(),
		Title:      p.Title,
		Slug:       p.Slug,
		Content:    p.Content,
		Icon:       p.Icon,
		CoverImage: p.CoverImage,
		Position:   p.Position,
		Depth:      p.Depth,
		IsTemplate: p.IsTemplate,
		CreatedBy:  p.CreatedBy.String(),
		UpdatedBy:  p.UpdatedBy.String(),
		Version:    p.Version,
		CreatedAt:  p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if p.ParentID != nil {
		s := p.ParentID.String()
		detail.ParentID = &s
	}
	if p.Freshness != nil {
		fi := &freshnessInfo{
			Score:  p.Freshness.Score,
			Status: string(p.Freshness.Status),
		}
		if !p.Freshness.LastVerifiedAt.IsZero() {
			s := p.Freshness.LastVerifiedAt.UTC().Format("2006-01-02T15:04:05Z")
			fi.LastVerifiedAt = &s
		}
		if !p.Freshness.NextReviewAt.IsZero() {
			s := p.Freshness.NextReviewAt.UTC().Format("2006-01-02T15:04:05Z")
			fi.NextReviewAt = &s
		}
		detail.Freshness = fi
	}
	for _, t := range p.Tags {
		detail.Tags = append(detail.Tags, tagRef{
			ID:    t.ID.String(),
			Name:  t.Name,
			Color: t.Color,
		})
	}
	return detail
}

// ListBySpace handles GET /api/v1/spaces/:spaceId/pages.
func (h *PageHandler) ListBySpace(c echo.Context) error {
	spaceID, err := parseUUID(c, "spaceId")
	if err != nil {
		return err
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "flat"
	}
	maxDepth := -1
	if d := c.QueryParam("depth"); d != "" {
		if n, parseErr := strconv.Atoi(d); parseErr == nil {
			maxDepth = n
		}
	}

	pages, err := h.svc.ListBySpace(c.Request().Context(), spaceID, format, maxDepth)
	if err != nil {
		return mapDomainError(c, err)
	}

	items := make([]pageListItem, len(pages))
	for i, p := range pages {
		items[i] = toPageListItem(p)
	}

	return respondOK(c, items)
}

// Create handles POST /api/v1/spaces/:spaceId/pages.
func (h *PageHandler) Create(c echo.Context) error {
	spaceID, err := parseUUID(c, "spaceId")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req domain.CreatePageRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Title == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "title is required")
	}

	page, err := h.svc.Create(c.Request().Context(), spaceID, userID, req)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondCreated(c, toPageDetail(page))
}

// GetByID handles GET /api/v1/pages/:id.
func (h *PageHandler) GetByID(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	page, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toPageDetail(page))
}

// Update handles PUT /api/v1/pages/:id.
func (h *PageHandler) Update(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req domain.UpdatePageRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}

	page, err := h.svc.Update(c.Request().Context(), id, userID, req)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toPageDetail(page))
}

// Delete handles DELETE /api/v1/pages/:id.
func (h *PageHandler) Delete(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	if err := h.svc.Delete(c.Request().Context(), id, userID); err != nil {
		return mapDomainError(c, err)
	}

	return respondNoContent(c)
}

// moveRequest is the body for PUT /api/v1/pages/:id/move.
type moveRequest struct {
	ParentID *string `json:"parent_id"`
	Position int     `json:"position"`
}

// Move handles PUT /api/v1/pages/:id/move.
func (h *PageHandler) Move(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req moveRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		pid, parseErr := uuid.Parse(*req.ParentID)
		if parseErr != nil {
			return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "parent_id must be a valid UUID")
		}
		parentID = &pid
	}

	page, err := h.svc.Move(c.Request().Context(), id, userID, parentID, req.Position)
	if err != nil {
		return mapDomainError(c, err)
	}

	parentIDStr := (*string)(nil)
	if page.ParentID != nil {
		s := page.ParentID.String()
		parentIDStr = &s
	}

	return respondOK(c, map[string]any{
		"id":         page.ID.String(),
		"parent_id":  parentIDStr,
		"position":   page.Position,
		"depth":      page.Depth,
		"updated_at": page.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// ListVersions handles GET /api/v1/pages/:id/versions.
func (h *PageHandler) ListVersions(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	versions, err := h.svc.ListVersions(c.Request().Context(), id)
	if err != nil {
		return mapDomainError(c, err)
	}

	out := make([]map[string]any, len(versions))
	for i, v := range versions {
		out[i] = versionToMap(v)
	}

	return respondOK(c, out)
}

// GetVersion handles GET /api/v1/pages/:id/versions/:version.
func (h *PageHandler) GetVersion(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	versionNum, parseErr := strconv.Atoi(c.Param("version"))
	if parseErr != nil || versionNum < 1 {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "version must be a positive integer")
	}

	v, err := h.svc.GetVersion(c.Request().Context(), id, versionNum)
	if err != nil {
		return mapDomainError(c, err)
	}

	m := versionToMap(v)
	m["content"] = v.Content
	return respondOK(c, m)
}

func versionToMap(v *domain.PageVersion) map[string]any {
	m := map[string]any{
		"id":             v.ID.String(),
		"page_id":        v.PageID.String(),
		"version":        v.Version,
		"title":          v.Title,
		"change_summary": v.ChangeSummary,
		"created_at":     v.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if v.CreatedBy != (uuid.UUID{}) {
		m["created_by"] = map[string]any{
			"id":           v.CreatedBy.String(),
			"display_name": v.CreatedByName,
		}
	}
	return m
}

// importMarkdownRequest is the body for POST /api/v1/pages/:id/import/markdown.
type importMarkdownRequest struct {
	Markdown string `json:"markdown"`
}

// ImportMarkdown handles POST /api/v1/pages/:id/import/markdown.
func (h *PageHandler) ImportMarkdown(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req importMarkdownRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Markdown == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "markdown is required")
	}

	page, err := h.svc.ImportMarkdown(c.Request().Context(), id, userID, req.Markdown)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, map[string]any{
		"id":         page.ID.String(),
		"title":      page.Title,
		"content":    page.Content,
		"version":    page.Version,
		"updated_at": page.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// ExportMarkdown handles GET /api/v1/pages/:id/export/markdown.
func (h *PageHandler) ExportMarkdown(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	// We need the page title for the response.
	page, svcErr := h.svc.GetByID(c.Request().Context(), id)
	if svcErr != nil {
		return mapDomainError(c, svcErr)
	}

	md, err := h.svc.ExportMarkdown(c.Request().Context(), id)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, map[string]any{
		"page_id":     id.String(),
		"title":       page.Title,
		"markdown":    md,
		"exported_at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	})
}
