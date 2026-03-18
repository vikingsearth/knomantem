CREATE TABLE graph_edges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    target_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    edge_type       VARCHAR(50) NOT NULL DEFAULT 'reference'
                    CHECK (edge_type IN ('reference', 'parent', 'related', 'depends_on', 'derived_from')),
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_by      UUID        NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_graph_edge UNIQUE (source_page_id, target_page_id, edge_type),
    CONSTRAINT chk_no_self_link CHECK (source_page_id != target_page_id)
);

CREATE INDEX idx_graph_source ON graph_edges (source_page_id);
CREATE INDEX idx_graph_target ON graph_edges (target_page_id);
CREATE INDEX idx_graph_type   ON graph_edges (edge_type);
