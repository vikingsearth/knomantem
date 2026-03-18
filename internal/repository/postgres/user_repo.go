package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/knomantem/knomantem/internal/domain"
)

// UserRepo implements domain.UserRepository against PostgreSQL.
type UserRepo struct {
	db *DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, display_name, password_hash, avatar_url, role,
		       settings, created_at, updated_at
		FROM users WHERE id = $1`
	row := r.db.Pool.QueryRow(ctx, q, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, display_name, password_hash, avatar_url, role,
		       settings, created_at, updated_at
		FROM users WHERE email = $1`
	row := r.db.Pool.QueryRow(ctx, q, email)
	return scanUser(row)
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	const q = `
		INSERT INTO users (id, email, display_name, password_hash, avatar_url, role, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, display_name, password_hash, avatar_url, role,
		          settings, created_at, updated_at`
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	row := r.db.Pool.QueryRow(ctx, q,
		u.ID, u.Email, u.DisplayName, u.PasswordHash, u.AvatarURL,
		string(u.Role), []byte(u.Settings),
	)
	return scanUser(row)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) (*domain.User, error) {
	const q = `
		UPDATE users
		SET email=$2, display_name=$3, avatar_url=$4, role=$5, settings=$6, updated_at=NOW()
		WHERE id=$1
		RETURNING id, email, display_name, password_hash, avatar_url, role,
		          settings, created_at, updated_at`
	row := r.db.Pool.QueryRow(ctx, q,
		u.ID, u.Email, u.DisplayName, u.AvatarURL, string(u.Role), []byte(u.Settings),
	)
	return scanUser(row)
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const q = `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`
	tag, err := r.db.Pool.Exec(ctx, q, id, passwordHash)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdateLastActive(ctx context.Context, id uuid.UUID, t time.Time) error {
	const q = `UPDATE users SET last_active_at=$2, updated_at=NOW() WHERE id=$1`
	_, err := r.db.Pool.Exec(ctx, q, id, t)
	return mapError(err)
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM users WHERE id=$1`
	tag, err := r.db.Pool.Exec(ctx, q, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var role string
	var settings []byte
	var avatarURL *string

	err := row.Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash,
		&avatarURL, &role, &settings, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}
	u.Role = domain.Role(role)
	u.AvatarURL = avatarURL
	if len(settings) > 0 {
		u.Settings = domain.Settings(settings)
	}
	return &u, nil
}
