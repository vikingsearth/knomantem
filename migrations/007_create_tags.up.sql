CREATE TABLE tags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    color           VARCHAR(7)   NOT NULL DEFAULT '#6B7280',
    is_ai_generated BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_name ON tags (name);

CREATE TABLE page_tags (
    page_id          UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id           UUID         NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    confidence_score DECIMAL(3,2) NOT NULL DEFAULT 1.00
                     CHECK (confidence_score >= 0 AND confidence_score <= 1),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    PRIMARY KEY (page_id, tag_id)
);

CREATE INDEX idx_page_tags_tag ON page_tags (tag_id);
