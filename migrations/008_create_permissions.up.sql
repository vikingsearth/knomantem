CREATE TABLE permissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type    VARCHAR(50) NOT NULL
                     CHECK (resource_type IN ('space', 'page')),
    resource_id      UUID        NOT NULL,
    permission_level VARCHAR(50) NOT NULL
                     CHECK (permission_level IN ('owner', 'editor', 'commenter', 'viewer')),
    granted_by       UUID        NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permission UNIQUE (user_id, resource_type, resource_id)
);

CREATE INDEX idx_permissions_user     ON permissions (user_id);
CREATE INDEX idx_permissions_resource ON permissions (resource_type, resource_id);
