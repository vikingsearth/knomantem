package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/knomantem/knomantem/internal/domain"
)

// NotificationRepo implements domain.NotificationRepository against PostgreSQL.
type NotificationRepo struct {
	db *DB
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(db *DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	const q = `
		INSERT INTO notifications (id, user_id, type, page_id, message, read)
		VALUES ($1,$2,$3,$4,$5,false)
		RETURNING id, user_id, type, page_id, message, read, created_at`
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	row := r.db.Pool.QueryRow(ctx, q, n.ID, n.UserID, n.Type, n.PageID, n.Message)
	out := &domain.Notification{}
	err := row.Scan(&out.ID, &out.UserID, &out.Type, &out.PageID, &out.Message, &out.Read, &out.CreatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return out, nil
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]*domain.Notification, error) {
	q := `
		SELECT id, user_id, type, page_id, message, read, created_at
		FROM notifications WHERE user_id=$1`
	if unreadOnly {
		q += " AND read=false"
	}
	q += " ORDER BY created_at DESC LIMIT 50"

	rows, err := r.db.Pool.Query(ctx, q, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	var out []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.PageID, &n.Message, &n.Read, &n.CreatedAt); err != nil {
			return nil, mapError(err)
		}
		out = append(out, n)
	}
	return out, mapError(rows.Err())
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	const q = `UPDATE notifications SET read=true WHERE id=$1 AND user_id=$2`
	tag, err := r.db.Pool.Exec(ctx, q, id, userID)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
