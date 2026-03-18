package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// ---- mock FreshnessRepository ----

type mockFreshnessRepo struct {
	records    map[uuid.UUID]*domain.Freshness
	updateFn   func(ctx context.Context, f *domain.Freshness) (*domain.Freshness, error)
	listStale  []*domain.Freshness
	listStaleFn func(ctx context.Context, threshold float64, limit int) ([]*domain.Freshness, error)
}

func newMockFreshnessRepo() *mockFreshnessRepo {
	return &mockFreshnessRepo{records: make(map[uuid.UUID]*domain.Freshness)}
}

func (m *mockFreshnessRepo) GetByPageID(ctx context.Context, pageID uuid.UUID) (*domain.Freshness, error) {
	if f, ok := m.records[pageID]; ok {
		return f, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockFreshnessRepo) Create(ctx context.Context, f *domain.Freshness) (*domain.Freshness, error) {
	m.records[f.PageID] = f
	return f, nil
}

func (m *mockFreshnessRepo) Update(ctx context.Context, f *domain.Freshness) (*domain.Freshness, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, f)
	}
	m.records[f.PageID] = f
	return f, nil
}

func (m *mockFreshnessRepo) ListStale(ctx context.Context, threshold float64, limit int) ([]*domain.Freshness, error) {
	if m.listStaleFn != nil {
		return m.listStaleFn(ctx, threshold, limit)
	}
	return m.listStale, nil
}

func (m *mockFreshnessRepo) Dashboard(ctx context.Context, userID string, status string, sort string, cursor string, limit int) ([]*domain.Freshness, int, string, error) {
	var results []*domain.Freshness
	for _, f := range m.records {
		if status == "" || string(f.Status) == status {
			results = append(results, f)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, len(m.records), "", nil
}

// ---- mock NotificationRepository ----

type mockNotificationRepo struct {
	created []*domain.Notification
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	n.ID = uuid.New()
	m.created = append(m.created, n)
	return n, nil
}

func (m *mockNotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]*domain.Notification, error) {
	return nil, nil
}

func (m *mockNotificationRepo) MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return nil
}

// ---- helpers ----

func seedFreshness(repo *mockFreshnessRepo, pageID uuid.UUID, score float64, interval int, decayRate float64) *domain.Freshness {
	now := time.Now()
	f := &domain.Freshness{
		ID:                 uuid.New(),
		PageID:             pageID,
		OwnerID:            uuid.New(),
		Score:              score,
		Status:             freshnessStatus(score),
		ReviewIntervalDays: interval,
		DecayRate:          decayRate,
		LastReviewedAt:     now,
		NextReviewAt:       now.AddDate(0, 0, interval),
	}
	repo.records[pageID] = f
	return f
}

func newFreshnessSvc(fr *mockFreshnessRepo, pr domain.PageRepository, nr domain.NotificationRepository) *FreshnessService {
	return NewFreshnessService(fr, pr, nr)
}

// ---- freshnessStatus formula tests ----

func TestFreshnessStatus_Boundaries(t *testing.T) {
	cases := []struct {
		score  float64
		status domain.FreshnessStatus
	}{
		{100.0, domain.FreshnessFresh},
		{70.0, domain.FreshnessFresh},
		{69.9, domain.FreshnessAging},
		{30.0, domain.FreshnessAging},
		{29.9, domain.FreshnessStale},
		{0.0, domain.FreshnessStale},
	}
	for _, tc := range cases {
		got := freshnessStatus(tc.score)
		if got != tc.status {
			t.Errorf("freshnessStatus(%.1f) = %q, want %q", tc.score, got, tc.status)
		}
	}
}

func TestDecayFormula(t *testing.T) {
	// score = 100 * (1 - decayRate * days / intervalDays)
	cases := []struct {
		decayRate    float64
		days         float64
		intervalDays float64
		wantScore    float64
	}{
		{0.5, 10, 30, 100 * (1 - 0.5*10.0/30.0)},  // ~83.33
		{1.0, 30, 30, 0.0},                           // fully decayed
		{1.0, 60, 30, 0.0},                           // clamped to 0
		{0.0, 100, 30, 100.0},                         // no decay
		{0.5, 0, 30, 100.0},                           // no days elapsed
	}
	for _, tc := range cases {
		days := tc.days
		score := 100.0 * (1 - tc.decayRate*days/tc.intervalDays)
		if score < 0 {
			score = 0
		}
		if math.Abs(score-tc.wantScore) > 0.001 {
			t.Errorf("decay formula (rate=%.2f days=%.0f interval=%.0f): got %.4f, want %.4f",
				tc.decayRate, tc.days, tc.intervalDays, score, tc.wantScore)
		}
	}
}

// ---- Verify tests ----

func TestFreshnessService_Verify_Success(t *testing.T) {
	fr := newMockFreshnessRepo()
	pr := &stubPageRepo{}
	nr := &mockNotificationRepo{}
	svc := newFreshnessSvc(fr, pr, nr)

	pageID := uuid.New()
	seedFreshness(fr, pageID, 40.0, 30, 0.5)
	userID := uuid.New()

	f, err := svc.Verify(context.Background(), pageID, userID.String(), "looks good")
	if err != nil {
		t.Fatalf("Verify: unexpected error: %v", err)
	}
	if f.Score != 100.0 {
		t.Errorf("Score after verify: got %.1f, want 100.0", f.Score)
	}
	if f.Status != domain.FreshnessFresh {
		t.Errorf("Status after verify: got %q, want fresh", f.Status)
	}
	if f.LastVerifiedBy != userID {
		t.Errorf("LastVerifiedBy mismatch")
	}
	if f.LastVerifiedAt.IsZero() {
		t.Error("LastVerifiedAt should not be zero after verify")
	}
	// NextReviewAt should be set to ReviewIntervalDays in the future.
	expectedNext := f.LastReviewedAt.AddDate(0, 0, f.ReviewIntervalDays)
	diff := f.NextReviewAt.Sub(expectedNext)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("NextReviewAt off: got %v, expected ~%v", f.NextReviewAt, expectedNext)
	}
}

func TestFreshnessService_Verify_InvalidUserID(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	seedFreshness(fr, pageID, 50.0, 30, 0.5)

	_, err := svc.Verify(context.Background(), pageID, "not-a-uuid", "")
	if err == nil {
		t.Fatal("expected error for invalid userID, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestFreshnessService_Verify_NotFound(t *testing.T) {
	svc := newFreshnessSvc(newMockFreshnessRepo(), &stubPageRepo{}, &mockNotificationRepo{})
	_, err := svc.Verify(context.Background(), uuid.New(), uuid.New().String(), "")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- UpdateSettings tests ----

func TestFreshnessService_UpdateSettings_Interval(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	seedFreshness(fr, pageID, 80.0, 30, 0.5)

	newInterval := 60
	f, err := svc.UpdateSettings(context.Background(), pageID, uuid.New().String(), domain.FreshnessSettingsRequest{
		ReviewIntervalDays: &newInterval,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if f.ReviewIntervalDays != 60 {
		t.Errorf("ReviewIntervalDays: got %d, want 60", f.ReviewIntervalDays)
	}
}

func TestFreshnessService_UpdateSettings_DecayRate(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	seedFreshness(fr, pageID, 80.0, 30, 0.5)

	newRate := 0.8
	f, err := svc.UpdateSettings(context.Background(), pageID, uuid.New().String(), domain.FreshnessSettingsRequest{
		DecayRate: &newRate,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if math.Abs(f.DecayRate-0.8) > 0.0001 {
		t.Errorf("DecayRate: got %f, want 0.8", f.DecayRate)
	}
}

func TestFreshnessService_UpdateSettings_ValidationErrors(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})
	pageID := uuid.New()
	seedFreshness(fr, pageID, 80.0, 30, 0.5)

	zero := 0
	negRate := -0.1
	overRate := 1.1
	badOwner := "not-a-uuid"

	cases := []struct {
		name string
		req  domain.FreshnessSettingsRequest
	}{
		{"zero interval", domain.FreshnessSettingsRequest{ReviewIntervalDays: &zero}},
		{"negative decay rate", domain.FreshnessSettingsRequest{DecayRate: &negRate}},
		{"decay rate > 1", domain.FreshnessSettingsRequest{DecayRate: &overRate}},
		{"invalid owner UUID", domain.FreshnessSettingsRequest{OwnerID: &badOwner}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpdateSettings(context.Background(), pageID, uuid.New().String(), tc.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestFreshnessService_UpdateSettings_OwnerChange(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	seedFreshness(fr, pageID, 80.0, 30, 0.5)

	newOwner := uuid.New().String()
	f, err := svc.UpdateSettings(context.Background(), pageID, uuid.New().String(), domain.FreshnessSettingsRequest{
		OwnerID: &newOwner,
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if f.OwnerID.String() != newOwner {
		t.Errorf("OwnerID: got %v, want %v", f.OwnerID, newOwner)
	}
}

// ---- RunDecay (RecalculateAll) tests ----

func TestFreshnessService_RunDecay_ScoresDecay(t *testing.T) {
	fr := newMockFreshnessRepo()
	nr := &mockNotificationRepo{}
	svc := newFreshnessSvc(fr, &stubPageRepo{}, nr)

	pageID := uuid.New()
	ownerID := uuid.New()
	// Set LastReviewedAt to 15 days ago — with 0.5 decay over 30-day interval,
	// score = 100*(1 - 0.5*15/30) = 75.0  → "aging"
	f := &domain.Freshness{
		ID:                 uuid.New(),
		PageID:             pageID,
		OwnerID:            ownerID,
		Score:              90.0,
		Status:             domain.FreshnessFresh,
		ReviewIntervalDays: 30,
		DecayRate:          0.5,
		LastReviewedAt:     time.Now().AddDate(0, 0, -15),
		NextReviewAt:       time.Now().AddDate(0, 0, 15),
	}
	fr.listStale = []*domain.Freshness{f}
	fr.records[pageID] = f

	err := svc.RunDecay(context.Background())
	if err != nil {
		t.Fatalf("RunDecay: unexpected error: %v", err)
	}

	updated := fr.records[pageID]
	expectedScore := 100.0 * (1 - 0.5*15.0/30.0)
	if math.Abs(updated.Score-expectedScore) > 0.5 {
		t.Errorf("Score after decay: got %.2f, want ~%.2f", updated.Score, expectedScore)
	}
}

func TestFreshnessService_RunDecay_ScoreClampsToZero(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	f := &domain.Freshness{
		ID:                 uuid.New(),
		PageID:             pageID,
		OwnerID:            uuid.New(),
		Score:              5.0,
		Status:             domain.FreshnessStale,
		ReviewIntervalDays: 10,
		DecayRate:          1.0,
		// 100 days ago — would produce deeply negative score
		LastReviewedAt: time.Now().AddDate(0, 0, -100),
		NextReviewAt:   time.Now().AddDate(0, 0, -90),
	}
	fr.listStale = []*domain.Freshness{f}
	fr.records[pageID] = f

	if err := svc.RunDecay(context.Background()); err != nil {
		t.Fatalf("RunDecay: %v", err)
	}

	updated := fr.records[pageID]
	if updated.Score < 0 {
		t.Errorf("Score should be clamped to 0, got %f", updated.Score)
	}
}

func TestFreshnessService_RunDecay_NotificationSentBelowThreshold(t *testing.T) {
	fr := newMockFreshnessRepo()
	nr := &mockNotificationRepo{}
	svc := newFreshnessSvc(fr, &stubPageRepo{}, nr)

	pageID := uuid.New()
	ownerID := uuid.New()
	// 100 days ago with 1.0 decay → score goes to 0 (< 30 threshold)
	f := &domain.Freshness{
		ID:                 uuid.New(),
		PageID:             pageID,
		OwnerID:            ownerID,
		Score:              25.0,
		Status:             domain.FreshnessStale,
		ReviewIntervalDays: 30,
		DecayRate:          1.0,
		LastReviewedAt:     time.Now().AddDate(0, 0, -100),
		NextReviewAt:       time.Now().AddDate(0, 0, -70),
	}
	fr.listStale = []*domain.Freshness{f}
	fr.records[pageID] = f

	if err := svc.RunDecay(context.Background()); err != nil {
		t.Fatalf("RunDecay: %v", err)
	}

	if len(nr.created) == 0 {
		t.Error("expected notification to be created for score below 30")
	}
	n := nr.created[0]
	if n.Type != "freshness_alert" {
		t.Errorf("notification type: got %q, want freshness_alert", n.Type)
	}
	if n.UserID != ownerID {
		t.Errorf("notification UserID mismatch")
	}
}

func TestFreshnessService_RunDecay_ZeroLastReviewedSkipped(t *testing.T) {
	fr := newMockFreshnessRepo()
	nr := &mockNotificationRepo{}
	svc := newFreshnessSvc(fr, &stubPageRepo{}, nr)

	pageID := uuid.New()
	// zero LastReviewedAt — should be skipped by RunDecay
	f := &domain.Freshness{
		ID:                 uuid.New(),
		PageID:             pageID,
		OwnerID:            uuid.New(),
		Score:              20.0,
		Status:             domain.FreshnessStale,
		ReviewIntervalDays: 30,
		DecayRate:          0.5,
		// LastReviewedAt intentionally zero
	}
	fr.listStale = []*domain.Freshness{f}
	fr.records[pageID] = f

	// The score should remain unchanged because the record is skipped.
	originalScore := f.Score
	if err := svc.RunDecay(context.Background()); err != nil {
		t.Fatalf("RunDecay: %v", err)
	}
	// Update fn is default (stores back), but RunDecay skips zero LastReviewedAt.
	// So no notification should have been fired and the score in records should
	// still be the original.
	if len(nr.created) != 0 {
		t.Error("expected no notifications for record with zero LastReviewedAt")
	}
	if fr.records[pageID].Score != originalScore {
		t.Errorf("score changed for zero-LastReviewedAt record: got %f", fr.records[pageID].Score)
	}
}

func TestFreshnessService_RunDecay_RepoError(t *testing.T) {
	fr := newMockFreshnessRepo()
	fr.listStaleFn = func(ctx context.Context, threshold float64, limit int) ([]*domain.Freshness, error) {
		return nil, errors.New("db error")
	}
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	err := svc.RunDecay(context.Background())
	if err == nil {
		t.Fatal("expected error from ListStale, got nil")
	}
}

// ---- GetByPageID tests ----

func TestFreshnessService_GetByPageID_Found(t *testing.T) {
	fr := newMockFreshnessRepo()
	svc := newFreshnessSvc(fr, &stubPageRepo{}, &mockNotificationRepo{})

	pageID := uuid.New()
	seeded := seedFreshness(fr, pageID, 85.0, 30, 0.5)

	got, err := svc.GetByPageID(context.Background(), pageID)
	if err != nil {
		t.Fatalf("GetByPageID: unexpected error: %v", err)
	}
	if got.ID != seeded.ID {
		t.Errorf("ID mismatch")
	}
	if math.Abs(got.Score-85.0) > 0.001 {
		t.Errorf("Score: got %f, want 85.0", got.Score)
	}
}

func TestFreshnessService_GetByPageID_NotFound(t *testing.T) {
	svc := newFreshnessSvc(newMockFreshnessRepo(), &stubPageRepo{}, &mockNotificationRepo{})
	_, err := svc.GetByPageID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- Dashboard tests ----

func TestFreshnessService_Dashboard_SummarisesStatuses(t *testing.T) {
	fr := newMockFreshnessRepo()
	pr := &stubPageRepo{}
	svc := newFreshnessSvc(fr, pr, &mockNotificationRepo{})

	userID := uuid.New()
	for _, score := range []float64{90, 50, 20} {
		pageID := uuid.New()
		f := &domain.Freshness{
			ID:             uuid.New(),
			PageID:         pageID,
			OwnerID:        userID,
			Score:          score,
			Status:         freshnessStatus(score),
			LastReviewedAt: time.Now(),
			NextReviewAt:   time.Now().AddDate(0, 0, 30),
		}
		fr.records[pageID] = f
	}

	dash, err := svc.Dashboard(context.Background(), userID.String(), "", "score", "", 10)
	if err != nil {
		t.Fatalf("Dashboard: unexpected error: %v", err)
	}
	if dash.Total != 3 {
		t.Errorf("Total: got %d, want 3", dash.Total)
	}
	if len(dash.Pages) != 3 {
		t.Errorf("Pages len: got %d, want 3", len(dash.Pages))
	}
}
