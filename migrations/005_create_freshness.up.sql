CREATE TABLE freshness_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id             UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    owner_id            UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    freshness_score     DECIMAL(5,2) NOT NULL DEFAULT 100.00
                        CHECK (freshness_score >= 0 AND freshness_score <= 100),
    review_interval_days INTEGER     NOT NULL DEFAULT 30,
    last_reviewed_at    TIMESTAMPTZ,
    next_review_at      TIMESTAMPTZ,
    last_verified_by    UUID         REFERENCES users(id) ON DELETE SET NULL,
    last_verified_at    TIMESTAMPTZ,
    status              VARCHAR(20)  NOT NULL DEFAULT 'fresh'
                        CHECK (status IN ('fresh', 'aging', 'stale', 'unverified')),
    decay_rate          DECIMAL(5,4) NOT NULL DEFAULT 0.0333,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_freshness_page UNIQUE (page_id)
);

CREATE INDEX idx_freshness_status     ON freshness_records (status);
CREATE INDEX idx_freshness_next_review ON freshness_records (next_review_at);
CREATE INDEX idx_freshness_owner      ON freshness_records (owner_id);
