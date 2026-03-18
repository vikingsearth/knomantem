package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/knomantem/knomantem/internal/domain"
)

// FreshnessService is the interface the FreshnessHandler depends on.
type FreshnessService interface {
	GetByPageID(ctx context.Context, pageID uuid.UUID) (*domain.Freshness, error)
	Verify(ctx context.Context, pageID uuid.UUID, userID string, notes string) (*domain.Freshness, error)
	UpdateSettings(ctx context.Context, pageID uuid.UUID, userID string, req domain.FreshnessSettingsRequest) (*domain.Freshness, error)
	Dashboard(ctx context.Context, userID string, status string, sort string, cursor string, limit int) (*domain.FreshnessDashboard, error)
}

// FreshnessHandler handles freshness-related endpoints.
type FreshnessHandler struct {
	svc FreshnessService
}

// NewFreshnessHandler creates a new FreshnessHandler.
func NewFreshnessHandler(svc FreshnessService) *FreshnessHandler {
	return &FreshnessHandler{svc: svc}
}

func toFreshnessResponse(f *domain.Freshness) map[string]any {
	m := map[string]any{
		"id":                   f.ID.String(),
		"page_id":              f.PageID.String(),
		"owner_id":             f.OwnerID.String(),
		"freshness_score":      f.Score,
		"review_interval_days": f.ReviewIntervalDays,
		"status":               string(f.Status),
		"decay_rate":           f.DecayRate,
		"created_at":           f.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updated_at":           f.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if !f.LastReviewedAt.IsZero() {
		m["last_reviewed_at"] = f.LastReviewedAt.UTC().Format("2006-01-02T15:04:05Z")
	} else {
		m["last_reviewed_at"] = nil
	}
	if !f.NextReviewAt.IsZero() {
		m["next_review_at"] = f.NextReviewAt.UTC().Format("2006-01-02T15:04:05Z")
	} else {
		m["next_review_at"] = nil
	}
	if !f.LastVerifiedAt.IsZero() {
		m["last_verified_at"] = f.LastVerifiedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if f.LastVerifiedBy != (uuid.UUID{}) {
		m["last_verified_by"] = map[string]any{
			"id":           f.LastVerifiedBy.String(),
			"display_name": f.LastVerifiedByName,
		}
	}
	return m
}

// GetByPageID handles GET /api/v1/pages/:id/freshness.
func (h *FreshnessHandler) GetByPageID(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}

	f, err := h.svc.GetByPageID(c.Request().Context(), pageID)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, toFreshnessResponse(f))
}

// verifyRequest is the body for POST /api/v1/pages/:id/freshness/verify.
type verifyRequest struct {
	Notes string `json:"notes"`
}

// Verify handles POST /api/v1/pages/:id/freshness/verify.
func (h *FreshnessHandler) Verify(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req verifyRequest
	_ = c.Bind(&req) // notes is optional

	f, err := h.svc.Verify(c.Request().Context(), pageID, userID, req.Notes)
	if err != nil {
		return mapDomainError(c, err)
	}

	resp := map[string]any{
		"page_id":          f.PageID.String(),
		"freshness_score":  f.Score,
		"status":           string(f.Status),
		"next_review_at":   f.NextReviewAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updated_at":       f.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if !f.LastVerifiedAt.IsZero() {
		resp["last_verified_at"] = f.LastVerifiedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if f.LastVerifiedBy != (uuid.UUID{}) {
		resp["last_verified_by"] = map[string]any{
			"id":           f.LastVerifiedBy.String(),
			"display_name": f.LastVerifiedByName,
		}
	}

	return respondOK(c, resp)
}

// UpdateSettings handles PUT /api/v1/pages/:id/freshness/settings.
func (h *FreshnessHandler) UpdateSettings(c echo.Context) error {
	pageID, err := parseUUID(c, "id")
	if err != nil {
		return err
	}
	userID := userIDFromCtx(c)

	var req domain.FreshnessSettingsRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
	}

	f, err := h.svc.UpdateSettings(c.Request().Context(), pageID, userID, req)
	if err != nil {
		return mapDomainError(c, err)
	}

	return respondOK(c, map[string]any{
		"page_id":              f.PageID.String(),
		"review_interval_days": f.ReviewIntervalDays,
		"decay_rate":           f.DecayRate,
		"owner_id":             f.OwnerID.String(),
		"next_review_at":       f.NextReviewAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updated_at":           f.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// Dashboard handles GET /api/v1/freshness/dashboard.
func (h *FreshnessHandler) Dashboard(c echo.Context) error {
	userID := userIDFromCtx(c)
	status := c.QueryParam("status")
	sort := c.QueryParam("sort")
	if sort == "" {
		sort = "score"
	}
	cursor := c.QueryParam("cursor")
	limit := 20
	if l := c.QueryParam("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	dashboard, err := h.svc.Dashboard(c.Request().Context(), userID, status, sort, cursor, limit)
	if err != nil {
		return mapDomainError(c, err)
	}

	var nc *string
	if dashboard.NextCursor != "" {
		nc = &dashboard.NextCursor
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"summary": dashboard.Summary,
			"pages":   dashboard.Pages,
		},
		"pagination": pagination{
			NextCursor: nc,
			HasMore:    dashboard.NextCursor != "",
			Total:      dashboard.Total,
		},
	})
}
