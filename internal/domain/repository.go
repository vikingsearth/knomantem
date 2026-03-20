package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines persistence operations for User entities.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, u *User) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateLastActive(ctx context.Context, id uuid.UUID, t time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SpaceRepository defines persistence operations for Space entities.
type SpaceRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Space, error)
	GetBySlug(ctx context.Context, slug string) (*Space, error)
	List(ctx context.Context) ([]*Space, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Space, error)
	Create(ctx context.Context, s *Space) (*Space, error)
	Update(ctx context.Context, s *Space) (*Space, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PageRepository defines persistence operations for Page entities.
type PageRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Page, error)
	ListBySpace(ctx context.Context, spaceID uuid.UUID) ([]*Page, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*Page, error)
	Create(ctx context.Context, p *Page) (*Page, error)
	Update(ctx context.Context, p *Page) (*Page, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, position int) (*Page, error)
	ListVersions(ctx context.Context, pageID uuid.UUID) ([]*PageVersion, error)
	GetVersion(ctx context.Context, pageID uuid.UUID, version int) (*PageVersion, error)
	CreateVersion(ctx context.Context, v *PageVersion) error
}

// FreshnessRepository defines persistence operations for Freshness records.
type FreshnessRepository interface {
	GetByPageID(ctx context.Context, pageID uuid.UUID) (*Freshness, error)
	Create(ctx context.Context, f *Freshness) (*Freshness, error)
	Update(ctx context.Context, f *Freshness) (*Freshness, error)
	ListStale(ctx context.Context, threshold float64, limit int) ([]*Freshness, error)
	// ListNeedingDecay returns all pages whose next_review_at is in the past,
	// ordered by next_review_at ASC so the most-overdue pages are processed first.
	ListNeedingDecay(ctx context.Context, limit int) ([]*Freshness, error)
	// GetByPageIDs batch-fetches freshness records for a set of page IDs.
	GetByPageIDs(ctx context.Context, pageIDs []uuid.UUID) ([]*Freshness, error)
	Dashboard(ctx context.Context, userID string, status string, sort string, cursor string, limit int) ([]*Freshness, int, string, error)
}

// EdgeRepository defines persistence operations for Edge entities.
type EdgeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Edge, error)
	ListByPage(ctx context.Context, pageID uuid.UUID, direction string, edgeType string) ([]*Edge, error)
	Create(ctx context.Context, e *Edge) (*Edge, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Explore(ctx context.Context, rootID uuid.UUID, depth int, edgeType string, limit int) (*GraphExploreResult, error)
}

// TagRepository defines persistence operations for Tag entities.
type TagRepository interface {
	List(ctx context.Context, query string, limit int) ([]*Tag, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Tag, error)
	Create(ctx context.Context, t *Tag) (*Tag, error)
	AddToPage(ctx context.Context, pageID uuid.UUID, assignments []PageTagAssignment) ([]TagWithScore, error)
	ListByPage(ctx context.Context, pageID uuid.UUID) ([]Tag, error)
}

// NotificationRepository defines persistence operations for Notification entities.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) (*Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool) ([]*Notification, error)
	MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// SearchRepository defines full-text search operations backed by Bleve.
type SearchRepository interface {
	Index(ctx context.Context, p *Page) error
	Delete(ctx context.Context, pageID uuid.UUID) error
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
}
