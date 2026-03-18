package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// SearchService is the interface the SearchHandler depends on.
type SearchService interface {
	Search(ctx context.Context, userID string, params domain.SearchParams) (*domain.SearchResult, error)
}

// SearchHandler handles /api/v1/search.
type SearchHandler struct {
	svc SearchService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(svc SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search handles GET /api/v1/search.
func (h *SearchHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "query parameter 'q' is required")
	}

	params := domain.SearchParams{
		Query:     q,
		Tags:      c.QueryParam("tags"),
		Freshness: c.QueryParam("freshness"),
		From:      c.QueryParam("from"),
		To:        c.QueryParam("to"),
		Sort:      c.QueryParam("sort"),
		Cursor:    c.QueryParam("cursor"),
		Limit:     20,
	}

	if s := c.QueryParam("space"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "space must be a valid UUID")
		}
		params.SpaceID = &id
	}

	if l := c.QueryParam("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			params.Limit = n
		}
	}
	if params.Sort == "" {
		params.Sort = "relevance"
	}

	userID := userIDFromCtx(c)
	result, err := h.svc.Search(c.Request().Context(), userID, params)
	if err != nil {
		return mapDomainError(c, err)
	}

	var nc *string
	if result.NextCursor != "" {
		nc = &result.NextCursor
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"results":       result.Items,
			"facets":        result.Facets,
			"query_time_ms": result.QueryTimeMs,
			"total":         result.Total,
		},
		"pagination": pagination{
			NextCursor: nc,
			HasMore:    result.NextCursor != "",
			Total:      result.Total,
		},
	})
}
