package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// PageRepo implements domain.PageRepository against PostgreSQL.
type PageRepo struct {
	db *DB
}

// NewPageRepo creates a new PageRepo.
func NewPageRepo(db *DB) *PageRepo {
	return &PageRepo{db: db}
}

func (r *PageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Page, error) {
	const q = `
		SELECT id, space_id, parent_id, title, slug, content, icon, cover_image,
		       position, depth, is_template, version, created_by, updated_by,
		       created_at, updated_at
		FROM pages WHERE id = $1`
	row := r.db.Pool.QueryRow(ctx, q, id)
	return scanPage(row)
}

func (r *PageRepo) ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]*domain.Page, error) {
	const q = `
		SELECT id, space_id, parent_id, title, slug, content, icon, cover_image,
		       position, depth, is_template, version, created_by, updated_by,
		       created_at, updated_at
		FROM pages WHERE space_id = $1
		ORDER BY depth ASC, position ASC`
	rows, err := r.db.Pool.Query(ctx, q, spaceID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	return collectPages(rows)
}

func (r *PageRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Page, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `
		SELECT id, space_id, parent_id, title, slug, content, icon, cover_image,
		       position, depth, is_template, version, created_by, updated_by,
		       created_at, updated_at
		FROM pages WHERE id = ANY($1)`
	rows, err := r.db.Pool.Query(ctx, q, ids)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	return collectPages(rows)
}

func (r *PageRepo) Create(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	const q = `
		INSERT INTO pages
		  (id, space_id, parent_id, title, slug, content, icon, cover_image,
		   position, depth, is_template, version, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, space_id, parent_id, title, slug, content, icon, cover_image,
		          position, depth, is_template, version, created_by, updated_by,
		          created_at, updated_at`
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	row := r.db.Pool.QueryRow(ctx, q,
		p.ID, p.SpaceID, p.ParentID, p.Title, p.Slug, []byte(p.Content),
		p.Icon, p.CoverImage, p.Position, p.Depth, p.IsTemplate,
		p.Version, p.CreatedBy, p.UpdatedBy,
	)
	return scanPage(row)
}

func (r *PageRepo) Update(ctx context.Context, p *domain.Page) (*domain.Page, error) {
	const q = `
		UPDATE pages
		SET title=$2, content=$3, icon=$4, cover_image=$5, version=$6,
		    updated_by=$7, updated_at=NOW()
		WHERE id=$1
		RETURNING id, space_id, parent_id, title, slug, content, icon, cover_image,
		          position, depth, is_template, version, created_by, updated_by,
		          created_at, updated_at`
	row := r.db.Pool.QueryRow(ctx, q,
		p.ID, p.Title, []byte(p.Content), p.Icon, p.CoverImage, p.Version, p.UpdatedBy,
	)
	return scanPage(row)
}

func (r *PageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM pages WHERE id=$1`
	tag, err := r.db.Pool.Exec(ctx, q, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PageRepo) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, position int) (*domain.Page, error) {
	const q = `
		UPDATE pages SET parent_id=$2, position=$3, updated_at=NOW()
		WHERE id=$1
		RETURNING id, space_id, parent_id, title, slug, content, icon, cover_image,
		          position, depth, is_template, version, created_by, updated_by,
		          created_at, updated_at`
	row := r.db.Pool.QueryRow(ctx, q, id, parentID, position)
	return scanPage(row)
}

func (r *PageRepo) ListVersions(ctx context.Context, pageID uuid.UUID) ([]*domain.PageVersion, error) {
	const q = `
		SELECT id, page_id, version, title, change_summary, created_by, created_at
		FROM page_versions WHERE page_id=$1 ORDER BY version DESC`
	rows, err := r.db.Pool.Query(ctx, q, pageID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.PageVersion
	for rows.Next() {
		v := &domain.PageVersion{}
		if err := rows.Scan(
			&v.ID, &v.PageID, &v.Version, &v.Title,
			&v.ChangeSummary, &v.CreatedBy, &v.CreatedAt,
		); err != nil {
			return nil, mapError(err)
		}
		out = append(out, v)
	}
	return out, mapError(rows.Err())
}

func (r *PageRepo) GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*domain.PageVersion, error) {
	const q = `
		SELECT id, page_id, version, title, content, change_summary, created_by, created_at
		FROM page_versions WHERE page_id=$1 AND version=$2`
	row := r.db.Pool.QueryRow(ctx, q, pageID, version)

	v := &domain.PageVersion{}
	var content []byte
	err := row.Scan(
		&v.ID, &v.PageID, &v.Version, &v.Title,
		&content, &v.ChangeSummary, &v.CreatedBy, &v.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	v.Content = json.RawMessage(content)
	return v, nil
}

func (r *PageRepo) CreateVersion(ctx context.Context, v *domain.PageVersion) error {
	const q = `
		INSERT INTO page_versions (id, page_id, version, title, content, change_summary, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	_, err := r.db.Pool.Exec(ctx, q,
		v.ID, v.PageID, v.Version, v.Title,
		[]byte(v.Content), v.ChangeSummary, v.CreatedBy,
	)
	return mapError(err)
}

func scanPage(row pgx.Row) (*domain.Page, error) {
	p := &domain.Page{}
	var content []byte
	err := row.Scan(
		&p.ID, &p.SpaceID, &p.ParentID, &p.Title, &p.Slug, &content,
		&p.Icon, &p.CoverImage, &p.Position, &p.Depth, &p.IsTemplate,
		&p.Version, &p.CreatedBy, &p.UpdatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	if len(content) > 0 {
		p.Content = json.RawMessage(content)
	}
	return p, nil
}

func collectPages(rows pgx.Rows) ([]*domain.Page, error) {
	var out []*domain.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, mapError(rows.Err())
}
