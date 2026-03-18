package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// GraphService is the interface the GraphHandler depends on.
type GraphService interface {
	GetNeighbors(ctx context.Context, pageID uuid.UUID, edgeType string, direction string) (*domain.GraphNeighbors, error)
	CreateEdge(ctx context.Context, sourcePageID uuid.UUID, userID string, req domain.CreateEdgeRequest) (*domain.Edge, error)
	Explore(ctx context.Context, rootID uuid.UUID, depth int, edgeType string, limit int) (*domain.GraphExploreResult, error)
}

// GraphHandler handles graph-related endpoints.
type GraphHandler struct {
	svc GraphService
}

// NewGraphHandler creates a new GraphHandler.
func NewGraphHandler(svc GraphService) *GraphHandler {
	return &GraphHandler{svc: svc}
}

// GetNeighbors handles GET /api/v1/pages/:id/graph.
func (h *GraphHandler) GetNeighbors(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	edgeType := c.QueryParam("edge_type")
	direction := c.QueryParam("direction")
	if direction == "" {
		direction = "both"
	}

	neighbors, err := h.svc.GetNeighbors(c.Request().Context(), pageID, edgeType, direction)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, neighbors)
}

// createEdgeRequest is the body for POST /api/v1/pages/:id/graph/edges.
type createEdgeRequest struct {
	TargetPageID string         `json:"target_page_id"`
	EdgeType     string         `json:"edge_type"`
	Metadata     map[string]any `json:"metadata"`
}

// CreateEdge handles POST /api/v1/pages/:id/graph/edges.
func (h *GraphHandler) CreateEdge(c echo.Context) error {
	sourceID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req createEdgeRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.TargetPageID == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "target_page_id is required")
	}
	if req.EdgeType == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "edge_type is required")
	}

	targetID, parseErr := uuid.Parse(req.TargetPageID)
	if parseErr != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "target_page_id must be a valid UUID")
	}

	edge, err := h.svc.CreateEdge(c.Request().Context(), sourceID, userID, domain.CreateEdgeRequest{
		TargetPageID: targetID,
		EdgeType:     req.EdgeType,
		Metadata:     req.Metadata,
	})
	if err != nil {
		return mapDomainError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": map[string]any{
			"id":             edge.ID.String(),
			"source_page_id": edge.SourcePageID.String(),
			"target_page_id": edge.TargetPageID.String(),
			"edge_type":      edge.EdgeType,
			"metadata":       edge.Metadata,
			"created_by":     edge.CreatedBy.String(),
			"created_at":     edge.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		},
	})
}

// Explore handles GET /api/v1/graph/explore.
func (h *GraphHandler) Explore(c echo.Context) error {
	rootStr := c.QueryParam("root")
	if rootStr == "" {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "root parameter is required")
	}
	rootID, err := uuid.Parse(rootStr)
	if err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "root must be a valid UUID")
	}

	depth := 2
	if d := c.QueryParam("depth"); d != "" {
		if n, parseErr := strconv.Atoi(d); parseErr == nil && n >= 0 && n <= 5 {
			depth = n
		}
	}

	edgeType := c.QueryParam("edge_type")

	limit := 100
	if l := c.QueryParam("limit"); l != "" {
		if n, parseErr := strconv.Atoi(l); parseErr == nil && n > 0 {
			limit = n
		}
	}

	result, err := h.svc.Explore(c.Request().Context(), rootID, depth, edgeType, limit)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, result)
}
