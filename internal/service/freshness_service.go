package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// FreshnessService handles freshness scoring, verification, and the dashboard.
type FreshnessService struct {
	freshness     domain.FreshnessRepository
	pages         domain.PageRepository
	notifications domain.NotificationRepository
}

// NewFreshnessService creates a new FreshnessService.
func NewFreshnessService(
	freshness domain.FreshnessRepository,
	pages domain.PageRepository,
	notifications domain.NotificationRepository,
) *FreshnessService {
	return &FreshnessService{freshness: freshness, pages: pages, notifications: notifications}
}

// GetByPageID returns the freshness record for a page.
func (s *FreshnessService) GetByPageID(ctx context.Context, pageID uuid.UUID) (*domain.Freshness, error) {
	f, err := s.freshness.GetByPageID(ctx, pageID)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Verify marks a page as verified, resets score to 100, and recalculates next review.
func (s *FreshnessService) Verify(ctx context.Context, pageID uuid.UUID, userID string, notes string) (*domain.Freshness, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	f, err := s.freshness.GetByPageID(ctx, pageID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	f.Score = 100.0
	f.Status = domain.FreshnessFresh
	f.LastVerifiedAt = now
	f.LastVerifiedBy = uid
	f.LastReviewedAt = now
	f.NextReviewAt = now.AddDate(0, 0, f.ReviewIntervalDays)

	return s.freshness.Update(ctx, f)
}

// UpdateSettings updates the review interval and decay rate for a page's freshness.
func (s *FreshnessService) UpdateSettings(ctx context.Context, pageID uuid.UUID, userID string, req domain.FreshnessSettingsRequest) (*domain.Freshness, error) {
	f, err := s.freshness.GetByPageID(ctx, pageID)
	if err != nil {
		return nil, err
	}

	if req.ReviewIntervalDays != nil {
		if *req.ReviewIntervalDays < 1 {
			return nil, fmt.Errorf("%w: review_interval_days must be at least 1", domain.ErrValidation)
		}
		f.ReviewIntervalDays = *req.ReviewIntervalDays
		f.NextReviewAt = f.LastReviewedAt.AddDate(0, 0, f.ReviewIntervalDays)
	}
	if req.DecayRate != nil {
		if *req.DecayRate < 0 || *req.DecayRate > 1 {
			return nil, fmt.Errorf("%w: decay_rate must be between 0 and 1", domain.ErrValidation)
		}
		f.DecayRate = *req.DecayRate
	}
	if req.OwnerID != nil {
		ownerID, parseErr := uuid.Parse(*req.OwnerID)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: owner_id must be a valid UUID", domain.ErrValidation)
		}
		f.OwnerID = ownerID
	}

	return s.freshness.Update(ctx, f)
}

// Dashboard returns a freshness overview for the authenticated user.
func (s *FreshnessService) Dashboard(ctx context.Context, userID string, status string, sort string, cursor string, limit int) (*domain.FreshnessDashboard, error) {
	records, total, nextCursor, err := s.freshness.Dashboard(ctx, userID, status, sort, cursor, limit)
	if err != nil {
		return nil, err
	}

	summary := domain.FreshnessSummaryStats{TotalPages: total}
	var pages []domain.FreshnessPageSummary

	for _, f := range records {
		switch f.Status {
		case domain.FreshnessFresh:
			summary.Fresh++
		case domain.FreshnessAging:
			summary.Aging++
		case domain.FreshnessStale:
			summary.Stale++
		}
		summary.AverageScore += f.Score

		pageEntry := domain.FreshnessPageSummary{
			PageID:         f.PageID.String(),
			FreshnessScore: f.Score,
			Status:         f.Status,
			NextReviewAt:   f.NextReviewAt.UTC().Format("2006-01-02T15:04:05Z"),
			Owner: domain.UserSummary{
				ID: f.OwnerID.String(),
			},
		}
		if !f.LastVerifiedAt.IsZero() {
			s := f.LastVerifiedAt.UTC().Format("2006-01-02T15:04:05Z")
			pageEntry.LastVerifiedAt = &s
		}

		// Enrich with page title if available.
		if p, err := s.pages.GetByID(ctx, f.PageID); err == nil {
			pageEntry.Title = p.Title
			pageEntry.Space = domain.SpaceSummary{ID: p.SpaceID.String()}
		}

		pages = append(pages, pageEntry)
	}

	if total > 0 {
		summary.AverageScore /= float64(total)
	}

	return &domain.FreshnessDashboard{
		Summary:    summary,
		Pages:      pages,
		NextCursor: nextCursor,
		Total:      total,
	}, nil
}

// RunDecay applies time-based decay to all freshness records. Called by the worker.
func (s *FreshnessService) RunDecay(ctx context.Context) error {
	// Stale threshold: score below 30
	stale, err := s.freshness.ListStale(ctx, 30.0, 1000)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, f := range stale {
		if f.LastReviewedAt.IsZero() {
			continue
		}
		days := now.Sub(f.LastReviewedAt).Hours() / 24
		newScore := 100.0 * (1 - f.DecayRate*days/float64(f.ReviewIntervalDays))
		if newScore < 0 {
			newScore = 0
		}
		f.Score = newScore
		f.Status = freshnessStatus(newScore)
		_, _ = s.freshness.Update(ctx, f)

		// Notify page owner when score drops below threshold.
		if newScore < 30 {
			n := &domain.Notification{
				UserID:  f.OwnerID,
				Type:    "freshness_alert",
				PageID:  &f.PageID,
				Message: "Page freshness score has dropped below 30. Please review.",
			}
			_, _ = s.notifications.Create(ctx, n)
		}
	}
	return nil
}

func freshnessStatus(score float64) domain.FreshnessStatus {
	switch {
	case score >= 70:
		return domain.FreshnessFresh
	case score >= 30:
		return domain.FreshnessAging
	default:
		return domain.FreshnessStale
	}
}
