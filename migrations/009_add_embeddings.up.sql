-- Enable the pgvector extension (idempotent).
-- Requires pgvector >= 0.5.0 installed on the PostgreSQL server.
CREATE EXTENSION IF NOT EXISTS vector;

-- page_embeddings stores a single dense vector per page (one embedding model).
-- If support for multiple models per page is needed later, add a unique index
-- on (page_id, model) and drop the PRIMARY KEY constraint on page_id alone.
CREATE TABLE page_embeddings (
    page_id     UUID         PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    embedding   vector(384),                                  -- MiniLM-L6-v2 dimensions
    model       VARCHAR(100) NOT NULL DEFAULT 'all-MiniLM-L6-v2',
    indexed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- HNSW index for approximate nearest-neighbour search using cosine distance.
-- HNSW is preferred over ivfflat for real-time inserts and high recall (≥0.98).
-- vector_cosine_ops: use the <=> (cosine distance) operator in queries.
CREATE INDEX ON page_embeddings USING hnsw (embedding vector_cosine_ops);
