package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// SpaceRepo implements domain.SpaceRepository against PostgreSQL.
type SpaceRepo struct {
	db *DB
}

// NewSpaceRepo creates a new SpaceRepo.
func NewSpaceRepo(db *DB) *SpaceRepo {
	return &SpaceRepo{db: db}
}

func (r *SpaceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Space, error) {
	const q = `
		SELECT id, name, slug, description, icon, owner_id, settings, created_at, updated_at
		FROM spaces WHERE id = $1`
	row := r.db.Pool.QueryRow(ctx, q, id)
	return scanSpace(row)
}

func (r *SpaceRepo) GetBySlug(ctx context.Context, slug string) (*domain.Space, error) {
	const q = `
		SELECT id, name, slug, description, icon, owner_id, settings, created_at, updated_at
		FROM spaces WHERE slug = $1`
	row := r.db.Pool.QueryRow(ctx, q, slug)
	return scanSpace(row)
}

func (r *SpaceRepo) List(ctx context.Context) ([]*domain.Space, error) {
	const q = `
		SELECT id, name, slug, description, icon, owner_id, settings, created_at, updated_at
		FROM spaces ORDER BY created_at ASC`
	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	return collectSpaces(rows)
}

func (r *SpaceRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Space, error) {
	const q = `
		SELECT id, name, slug, description, icon, owner_id, settings, created_at, updated_at
		FROM spaces WHERE owner_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.Pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	return collectSpaces(rows)
}

func (r *SpaceRepo) Create(ctx context.Context, s *domain.Space) (*domain.Space, error) {
	const q = `
		INSERT INTO spaces (id, name, slug, description, icon, owner_id, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, slug, description, icon, owner_id, settings, created_at, updated_at`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	row := r.db.Pool.QueryRow(ctx, q,
		s.ID, s.Name, s.Slug, s.Description, s.Icon, s.OwnerID, []byte(s.Settings),
	)
	return scanSpace(row)
}

func (r *SpaceRepo) Update(ctx context.Context, s *domain.Space) (*domain.Space, error) {
	const q = `
		UPDATE spaces
		SET name=$2, slug=$3, description=$4, icon=$5, settings=$6, updated_at=NOW()
		WHERE id=$1
		RETURNING id, name, slug, description, icon, owner_id, settings, created_at, updated_at`
	row := r.db.Pool.QueryRow(ctx, q,
		s.ID, s.Name, s.Slug, s.Description, s.Icon, []byte(s.Settings),
	)
	return scanSpace(row)
}

func (r *SpaceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM spaces WHERE id=$1`
	tag, err := r.db.Pool.Exec(ctx, q, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanSpace(row pgx.Row) (*domain.Space, error) {
	var s domain.Space
	var settings []byte
	var description, icon *string

	err := row.Scan(
		&s.ID, &s.Name, &s.Slug, &description, &icon,
		&s.OwnerID, &settings, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	s.Description = description
	s.Icon = icon
	if len(settings) > 0 {
		s.Settings = domain.Settings(settings)
	}
	return &s, nil
}

func collectSpaces(rows pgx.Rows) ([]*domain.Space, error) {
	var out []*domain.Space
	for rows.Next() {
		s, err := scanSpace(rows)
		if err != nil {
			return nil, mapError(err)
		}
		out = append(out, s)
	}
	return out, mapError(rows.Err())
}
