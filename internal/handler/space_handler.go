package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// SpaceService is the interface the SpaceHandler depends on.
type SpaceService interface {
	List(ctx context.Context, userID string, cursor string, limit int) ([]*domain.Space, string, int, error)
	Create(ctx context.Context, userID string, req domain.CreateSpaceRequest) (*domain.Space, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Space, error)
	Update(ctx context.Context, id uuid.UUID, userID string, req domain.UpdateSpaceRequest) (*domain.Space, error)
	Delete(ctx context.Context, id uuid.UUID, userID string) error
}

// SpaceHandler handles /api/v1/spaces/* endpoints.
type SpaceHandler struct {
	svc SpaceService
}

// NewSpaceHandler creates a new SpaceHandler.
func NewSpaceHandler(svc SpaceService) *SpaceHandler {
	return &SpaceHandler{svc: svc}
}

// spaceResponse is the public representation of a Space.
type spaceResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	OwnerID     string          `json:"owner_id"`
	Settings    json.RawMessage `json:"settings,omitempty"`
	PageCount   int             `json:"page_count,omitempty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

func toSpaceResponse(s *domain.Space) spaceResponse {
	return spaceResponse{
		ID:          s.ID.String(),
		Name:        s.Name,
		Slug:        s.Slug,
		Description: s.Description,
		Icon:        s.Icon,
		OwnerID:     s.OwnerID.String(),
		Settings:    json.RawMessage(s.Settings),
		CreatedAt:   s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// List handles GET /api/v1/spaces.
func (h *SpaceHandler) List(c echo.Context) error {
	userID := userIDFromCtx(c)
	cursor := c.QueryParam("cursor")
	limit := 20
	if l := c.QueryParam("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	spaces, nextCursor, total, err := h.svc.List(c.Request().Context(), userID, cursor, limit)
	if err != nil {
		return mapDomainError(c, err)
	}

	out := make([]spaceResponse, len(spaces))
	for i, s := range spaces {
		out[i] = toSpaceResponse(s)
	}

	var nc *string
	if nextCursor != "" {
		nc = &nextCursor
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": out,
		"pagination": pagination{
			NextCursor: nc,
			HasMore:    nextCursor != "",
			Total:      total,
		},
	})
}

// Create handles POST /api/v1/spaces.
func (h *SpaceHandler) Create(c echo.Context) error {
	userID := userIDFromCtx(c)

	var req domain.CreateSpaceRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}
	if req.Name == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
	}

	space, err := h.svc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondCreated(c, toSpaceResponse(space))
}

// GetByID handles GET /api/v1/spaces/:id.
func (h *SpaceHandler) GetByID(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	space, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toSpaceResponse(space))
}

// Update handles PUT /api/v1/spaces/:id.
func (h *SpaceHandler) Update(c echo.Context) error {
	id, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req domain.UpdateSpaceRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}

	space, err := h.svc.Update(c.Request().Context(), id, userID, req)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toSpaceResponse(space))
}

// Delete handles DELETE /api/v1/spaces/:id.
func (h *SpaceHandler) Delete(c echo.Context) error {
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

// parseUUID extracts a UUID path parameter, returning a 400 response on failure.
func parseUUID(c echo.Context, param string) (uuid.UUID, error) {
	raw := c.Param(param)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, respondError(c, http.StatusBadRequest, "BAD_REQUEST",
			param+" must be a valid UUID")
	}
	return id, nil
}
