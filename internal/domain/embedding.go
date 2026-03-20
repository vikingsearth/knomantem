package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PageEmbedding stores the dense vector representation of a Page's content.
// One record exists per page; the model field identifies which embedding model
// produced the vector (default: all-MiniLM-L6-v2, 384 dimensions).
type PageEmbedding struct {
	// PageID is the primary key and foreign key to pages.id.
	PageID uuid.UUID `json:"page_id"`

	// Embedding is the dense float32 vector produced by the embedding model.
	// Length must match the vector(N) column dimension defined in the schema
	// (currently 384 for all-MiniLM-L6-v2).
	Embedding []float32 `json:"embedding"`

	// Model identifies the embedding model used to produce the vector,
	// e.g. "all-MiniLM-L6-v2" or "text-embedding-3-small".
	Model string `json:"model"`

	// IndexedAt is the timestamp when this embedding was last generated.
	IndexedAt time.Time `json:"indexed_at"`
}

// EmbeddingRepository defines the persistence interface for PageEmbedding records.
// The implementation (backed by pgx v5 + pgvector) lives in internal/repository.
type EmbeddingRepository interface {
	// Upsert inserts or replaces the embedding for the given page.
	// If a record already exists for the page it is overwritten (e.g., after
	// the page content changes and the embedding must be regenerated).
	Upsert(ctx context.Context, e *PageEmbedding) error

	// FindSimilar returns the limit nearest pages to the embedding of pageID,
	// ordered by ascending cosine distance (most similar first).
	// The result excludes the page itself.
	FindSimilar(ctx context.Context, pageID uuid.UUID, limit int) ([]*PageEmbedding, error)
}
