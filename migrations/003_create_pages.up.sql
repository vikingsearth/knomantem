CREATE TABLE pages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id    UUID         NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id   UUID         REFERENCES pages(id) ON DELETE SET NULL,
    title       VARCHAR(500) NOT NULL,
    slug        VARCHAR(500) NOT NULL,
    content     JSONB        NOT NULL DEFAULT '{"type":"doc","content":[]}',
    position    INTEGER      NOT NULL DEFAULT 0,
    depth       INTEGER      NOT NULL DEFAULT 0,
    icon        VARCHAR(50),
    cover_image TEXT,
    is_template BOOLEAN      NOT NULL DEFAULT FALSE,
    created_by  UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by  UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_pages_space_slug UNIQUE (space_id, slug)
);

CREATE INDEX idx_pages_space    ON pages (space_id);
CREATE INDEX idx_pages_parent   ON pages (parent_id);
CREATE INDEX idx_pages_position ON pages (space_id, parent_id, position);
CREATE INDEX idx_pages_created  ON pages (created_by);
