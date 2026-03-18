package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// TagService is the interface the TagHandler depends on.
type TagService interface {
	List(ctx context.Context, query string, limit int) ([]*domain.Tag, error)
	Create(ctx context.Context, userID string, req domain.CreateTagRequest) (*domain.Tag, error)
	AddToPage(ctx context.Context, pageID uuid.UUID, userID string, tags []domain.PageTagAssignment) ([]domain.TagWithScore, error)
}

// TagHandler handles /api/v1/tags and /api/v1/pages/:id/tags endpoints.
type TagHandler struct {
	svc TagService
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(svc TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

func toTagResponse(t *domain.Tag) map[string]any {
	return map[string]any{
		"id":               t.ID.String(),
		"name":             t.Name,
		"color":            t.Color,
		"is_ai_generated":  t.IsAIGenerated,
		"page_count":       t.PageCount,
		"created_at":       t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// List handles GET /api/v1/tags.
func (h *TagHandler) List(c echo.Context) error {
	q := c.QueryParam("q")
	limit := 50
	if l := c.QueryParam("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	tags, err := h.svc.List(c.Request().Context(), q, limit)
	if err != nil {
		return mapDomainError(c, err)
	}

	out := make([]map[string]any, len(tags))
	for i, t := range tags {
		out[i] = toTagResponse(t)
	}

	return respondOK(c, out)
}

// createTagRequest is the body for POST /api/v1/tags.
type createTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Create handles POST /api/v1/tags.
func (h *TagHandler) Create(c echo.Context) error {
	userID := userIDFromCtx(c)

	var req createTagRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Name == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
	}

	tag, err := h.svc.Create(c.Request().Context(), userID, domain.CreateTagRequest{
		Name:  req.Name,
		Color: req.Color,
	})
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondCreated(c, toTagResponse(tag))
}

// addTagsRequest is the body for POST /api/v1/pages/:id/tags.
type addTagsRequest struct {
	Tags []struct {
		TagID           string  `json:"tag_id"`
		ConfidenceScore float64 `json:"confidence_score"`
	} `json:"tags"`
}

// AddToPage handles POST /api/v1/pages/:id/tags.
func (h *TagHandler) AddToPage(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req addTagsRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if len(req.Tags) == 0 {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "tags array must not be empty")
	}

	assignments := make([]domain.PageTagAssignment, len(req.Tags))
	for i, t := range req.Tags {
		tid, parseErr := uuid.Parse(t.TagID)
		if parseErr != nil {
			return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "tag_id must be a valid UUID")
		}
		assignments[i] = domain.PageTagAssignment{
			TagID:           tid,
			ConfidenceScore: t.ConfidenceScore,
		}
	}

	tags, err := h.svc.AddToPage(c.Request().Context(), pageID, userID, assignments)
	if err != nil {
		return mapDomainError(c, err)
	}

	tagOut := make([]map[string]any, len(tags))
	for i, t := range tags {
		tagOut[i] = map[string]any{
			"id":               t.ID.String(),
			"name":             t.Name,
			"color":            t.Color,
			"confidence_score": t.ConfidenceScore,
		}
	}

	return respondOK(c, map[string]any{
		"page_id": pageID.String(),
		"tags":    tagOut,
	})
}
