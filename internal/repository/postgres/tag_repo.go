package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// Ensure TagRepo implements domain.TagRepository.
var _ domain.TagRepository = (*TagRepo)(nil)

// TagRepo implements domain.TagRepository against PostgreSQL.
type TagRepo struct {
	db *DB
}

// NewTagRepo creates a new TagRepo.
func NewTagRepo(db *DB) *TagRepo {
	return &TagRepo{db: db}
}

func (r *TagRepo) List(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	var rows pgx.Rows
	var err error

	if query != "" {
		const q = `
			SELECT id, name, color, is_ai_generated, created_at
			FROM tags WHERE name ILIKE $1 ORDER BY name ASC LIMIT $2`
		rows, err = r.db.Pool.Query(ctx, q, query+"%", limit)
	} else {
		const q = `
			SELECT id, name, color, is_ai_generated, created_at
			FROM tags ORDER BY name ASC LIMIT $1`
		rows, err = r.db.Pool.Query(ctx, q, limit)
	}
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Tag
	for rows.Next() {
		t := &domain.Tag{}
		if scanErr := rows.Scan(&t.ID, &t.Name, &t.Color, &t.IsAIGenerated, &t.CreatedAt); scanErr != nil {
			return nil, mapError(scanErr)
		}
		out = append(out, t)
	}
	return out, mapError(rows.Err())
}

func (r *TagRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	const q = `SELECT id, name, color, is_ai_generated, created_at FROM tags WHERE id=$1`
	row := r.db.Pool.QueryRow(ctx, q, id)
	t := &domain.Tag{}
	err := row.Scan(&t.ID, &t.Name, &t.Color, &t.IsAIGenerated, &t.CreatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return t, nil
}

func (r *TagRepo) Create(ctx context.Context, t *domain.Tag) (*domain.Tag, error) {
	const q = `
		INSERT INTO tags (id, name, color, is_ai_generated)
		VALUES ($1,$2,$3,$4)
		RETURNING id, name, color, is_ai_generated, created_at`
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	row := r.db.Pool.QueryRow(ctx, q, t.ID, t.Name, t.Color, t.IsAIGenerated)
	out := &domain.Tag{}
	err := row.Scan(&out.ID, &out.Name, &out.Color, &out.IsAIGenerated, &out.CreatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return out, nil
}

func (r *TagRepo) AddToPage(ctx context.Context, pageID uuid.UUID, assignments []domain.PageTagAssignment) ([]domain.TagWithScore, error) {
	var out []domain.TagWithScore
	for _, a := range assignments {
		const q = `
			INSERT INTO page_tags (page_id, tag_id, confidence_score)
			VALUES ($1,$2,$3)
			ON CONFLICT (page_id, tag_id) DO UPDATE SET confidence_score=$3`
		if _, err := r.db.Pool.Exec(ctx, q, pageID, a.TagID, a.ConfidenceScore); err != nil {
			return nil, mapError(err)
		}
		t, err := r.GetByID(ctx, a.TagID)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.TagWithScore{
			ID:              t.ID,
			Name:            t.Name,
			Color:           t.Color,
			ConfidenceScore: a.ConfidenceScore,
		})
	}
	return out, nil
}

func (r *TagRepo) ListByPage(ctx context.Context, pageID uuid.UUID) ([]domain.Tag, error) {
	const q = `
		SELECT t.id, t.name, t.color, t.is_ai_generated, t.created_at
		FROM tags t
		JOIN page_tags pt ON pt.tag_id = t.id
		WHERE pt.page_id=$1`
	rows, err := r.db.Pool.Query(ctx, q, pageID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []domain.Tag
	for rows.Next() {
		t := domain.Tag{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.IsAIGenerated, &t.CreatedAt); err != nil {
			return nil, mapError(err)
		}
		out = append(out, t)
	}
	return out, mapError(rows.Err())
}
