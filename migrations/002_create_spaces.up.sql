CREATE TABLE spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    icon        VARCHAR(50),
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    settings    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_spaces_owner ON spaces (owner_id);
CREATE INDEX idx_spaces_slug  ON spaces (slug);
