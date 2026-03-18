CREATE TABLE page_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id         UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    version         INTEGER      NOT NULL,
    title           VARCHAR(500) NOT NULL,
    content         JSONB        NOT NULL,
    change_summary  TEXT,
    created_by      UUID         NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_page_versions UNIQUE (page_id, version)
);

CREATE INDEX idx_page_versions_page ON page_versions (page_id, version DESC);
