package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// FreshnessRepo implements domain.FreshnessRepository against PostgreSQL.
type FreshnessRepo struct {
	db *DB
}

// NewFreshnessRepo creates a new FreshnessRepo.
func NewFreshnessRepo(db *DB) *FreshnessRepo {
	return &FreshnessRepo{db: db}
}

func (r *FreshnessRepo) GetByPageID(ctx context.Context, pageID uuid.UUID) (*domain.Freshness, error) {
	const q = `
		SELECT id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		       last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		       status, created_at, updated_at
		FROM freshness_records WHERE page_id = $1`
	row := r.db.Pool.QueryRow(ctx, q, pageID)
	return scanFreshness(row)
}

func (r *FreshnessRepo) Create(ctx context.Context, f *domain.Freshness) (*domain.Freshness, error) {
	const q = `
		INSERT INTO freshness_records
		  (id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		   last_reviewed_at, next_review_at, last_verified_at, last_verified_by, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		          last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		          status, created_at, updated_at`
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	var lastVerifiedBy *uuid.UUID
	if f.LastVerifiedBy != (uuid.UUID{}) {
		lastVerifiedBy = &f.LastVerifiedBy
	}
	row := r.db.Pool.QueryRow(ctx, q,
		f.ID, f.PageID, f.OwnerID, f.Score, f.ReviewIntervalDays, f.DecayRate,
		nullTime(f.LastReviewedAt), nullTime(f.NextReviewAt),
		nullTime(f.LastVerifiedAt), lastVerifiedBy, string(f.Status),
	)
	return scanFreshness(row)
}

func (r *FreshnessRepo) Update(ctx context.Context, f *domain.Freshness) (*domain.Freshness, error) {
	const q = `
		UPDATE freshness_records
		SET owner_id=$2, freshness_score=$3, review_interval_days=$4, decay_rate=$5,
		    last_reviewed_at=$6, next_review_at=$7, last_verified_at=$8,
		    last_verified_by=$9, status=$10, updated_at=NOW()
		WHERE id=$1
		RETURNING id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		          last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		          status, created_at, updated_at`
	var lastVerifiedBy *uuid.UUID
	if f.LastVerifiedBy != (uuid.UUID{}) {
		lastVerifiedBy = &f.LastVerifiedBy
	}
	row := r.db.Pool.QueryRow(ctx, q,
		f.ID, f.OwnerID, f.Score, f.ReviewIntervalDays, f.DecayRate,
		nullTime(f.LastReviewedAt), nullTime(f.NextReviewAt),
		nullTime(f.LastVerifiedAt), lastVerifiedBy, string(f.Status),
	)
	return scanFreshness(row)
}

func (r *FreshnessRepo) ListStale(ctx context.Context, threshold float64, limit int) ([]*domain.Freshness, error) {
	const q = `
		SELECT id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		       last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		       status, created_at, updated_at
		FROM freshness_records WHERE freshness_score < $1
		ORDER BY freshness_score ASC
		LIMIT $2`
	rows, err := r.db.Pool.Query(ctx, q, threshold, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Freshness
	for rows.Next() {
		f, err := scanFreshness(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, mapError(rows.Err())
}

// ListNeedingDecay returns pages whose next_review_at has passed, ordered by the
// most overdue first. It is used by the decay worker so that all pages — not just
// those already stale — receive periodic score updates.
func (r *FreshnessRepo) ListNeedingDecay(ctx context.Context, limit int) ([]*domain.Freshness, error) {
	const q = `
		SELECT id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		       last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		       status, created_at, updated_at
		FROM page_freshness
		WHERE next_review_at < NOW()
		ORDER BY next_review_at ASC
		LIMIT $1`
	rows, err := r.db.Pool.Query(ctx, q, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Freshness
	for rows.Next() {
		f, err := scanFreshness(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, mapError(rows.Err())
}

// GetByPageIDs batch-fetches freshness records for a set of page IDs. Any page
// that has no freshness record is simply absent from the returned slice.
func (r *FreshnessRepo) GetByPageIDs(ctx context.Context, pageIDs []uuid.UUID) ([]*domain.Freshness, error) {
	if len(pageIDs) == 0 {
		return nil, nil
	}

	const q = `
		SELECT id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		       last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		       status, created_at, updated_at
		FROM page_freshness
		WHERE page_id = ANY($1)`

	// pgx accepts a []uuid.UUID directly via the pgtype codec.
	rows, err := r.db.Pool.Query(ctx, q, pageIDs)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Freshness
	for rows.Next() {
		f, err := scanFreshness(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, mapError(rows.Err())
}

func (r *FreshnessRepo) Dashboard(ctx context.Context, userID string, status string, sort string, cursor string, limit int) ([]*domain.Freshness, int, string, error) {
	// Simplified implementation — returns recent freshness records.
	const q = `
		SELECT id, page_id, owner_id, freshness_score, review_interval_days, decay_rate,
		       last_reviewed_at, next_review_at, last_verified_at, last_verified_by,
		       status, created_at, updated_at
		FROM freshness_records
		ORDER BY freshness_score ASC
		LIMIT $1`
	rows, err := r.db.Pool.Query(ctx, q, limit)
	if err != nil {
		return nil, 0, "", mapError(err)
	}
	defer rows.Close()

	var out []*domain.Freshness
	for rows.Next() {
		f, err := scanFreshness(rows)
		if err != nil {
			return nil, 0, "", err
		}
		out = append(out, f)
	}
	return out, len(out), "", mapError(rows.Err())
}

func scanFreshness(row pgx.Row) (*domain.Freshness, error) {
	f := &domain.Freshness{}
	var statusStr string
	var lastVerifiedBy *uuid.UUID
	var lastReviewedAt, nextReviewAt, lastVerifiedAt *time.Time

	err := row.Scan(
		&f.ID, &f.PageID, &f.OwnerID, &f.Score, &f.ReviewIntervalDays, &f.DecayRate,
		&lastReviewedAt, &nextReviewAt, &lastVerifiedAt, &lastVerifiedBy,
		&statusStr, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	f.Status = domain.FreshnessStatus(statusStr)
	if lastVerifiedBy != nil {
		f.LastVerifiedBy = *lastVerifiedBy
	}
	if lastReviewedAt != nil {
		f.LastReviewedAt = *lastReviewedAt
	}
	if nextReviewAt != nil {
		f.NextReviewAt = *nextReviewAt
	}
	if lastVerifiedAt != nil {
		f.LastVerifiedAt = *lastVerifiedAt
	}
	return f, nil
}
