# Knomantem Data Model

> PostgreSQL 16 | pgx v5 driver | Hybrid Relational + JSONB

---

## 1. Design Philosophy

Knomantem's data model is built on four guiding principles that distinguish it from
conventional wiki and knowledge-base schemas.

### 1.1 Hybrid Relational + JSONB

Relational columns handle everything that needs to be queried, filtered, sorted, or
joined: titles, slugs, ownership, timestamps, and foreign keys. Document *content*,
however, lives in a JSONB column as a ProseMirror-compatible block tree. This gives us
the best of both worlds:

- Strong referential integrity and indexing for structural metadata.
- Schemaless flexibility for rich-text content that evolves independently of the
  database schema.
- GIN indexes on the JSONB column for full-document searches when needed.

### 1.2 Freshness as a First-Class Citizen

Every page carries a freshness score, a review schedule, and a verification history.
Freshness is not a bolt-on report; it is a core column with its own table, indexes,
background job, and notification pipeline. Knowledge that is not actively maintained
decays visibly.

### 1.3 Typed Knowledge Graph Edges

Links between pages are not just inline hyperlinks buried in content. Each link is a
first-class row in `graph_edges` with a semantic type (`references`, `depends_on`,
`supersedes`, `related_to`, `child_of`, `backlink`). This enables graph traversal
queries such as "What depends on this page?" or "Show me everything this page
supersedes."

### 1.4 Immutable Versions

Every edit creates a new row in `page_versions`. The `pages` table always holds the
latest snapshot for read performance, but the full history is preserved and diffable.
No content is ever silently overwritten.

---

## 2. Entity-Relationship Overview

The schema consists of **8 primary tables** and **1 junction table**.

```
 ┌────────────┐        ┌────────────┐
 │   users     │───────<│ permissions │
 └─────┬──────┘        └────────────┘
       │ 1
       │
       │ owns / creates
       ▼ N
 ┌────────────┐   1   N ┌────────────────┐
 │   spaces    │───────>│     pages       │
 └────────────┘         └──┬──────┬──────┘
                           │      │
              ┌────────────┤      ├────────────┐
              │            │      │            │
              ▼ N          ▼ N    ▼ N          ▼ N
     ┌──────────────┐ ┌──────────┐ ┌────────────────────┐
     │ page_versions│ │page_tags │ │ freshness_records   │
     └──────────────┘ └────┬─────┘ └────────────────────┘
                           │
                           ▼ N
                      ┌──────────┐
                      │   tags   │
                      └──────────┘

     ┌────────────┐
     │ graph_edges │  (source_page_id ──> target_page_id)
     └────────────┘
```

**Relationship summary:**

| Relationship | Cardinality | Notes |
|---|---|---|
| users -> spaces | 1:N | A user owns many spaces |
| spaces -> pages | 1:N | A space contains many pages |
| pages -> pages | 1:N (self) | Parent-child hierarchy via `parent_id` |
| pages -> page_versions | 1:N | Full version history per page |
| pages -> freshness_records | 1:N | Freshness tracking per page |
| pages <-> graph_edges | N:M | Typed edges between any two pages |
| pages <-> tags | N:M | Via `page_tags` junction table |
| users -> permissions | 1:N | Row-level access control |

---

## 3. Core Schema

### 3.1 `users`

Stores account information and preferences. Referenced as a foreign key by nearly
every other table.

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    display_name    VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    avatar_url      TEXT,
    role            VARCHAR(50)  NOT NULL DEFAULT 'member',
    settings        JSONB        DEFAULT '{}',
    last_active_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**Design decisions:**

- `role` is a simple string enum (`member`, `admin`) rather than a separate roles
  table, since the set of roles is small and stable.
- `settings` is JSONB to allow per-user preferences (theme, notification prefs,
  default space) without schema migrations.
- `password_hash` stores bcrypt or argon2id output; plaintext passwords never touch
  the database.

---

### 3.2 `spaces`

A space is the top-level organizational container, analogous to a workspace or
project.

```sql
CREATE TABLE spaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    icon        VARCHAR(50),
    owner_id    UUID         NOT NULL REFERENCES users(id),
    settings    JSONB        DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**Design decisions:**

- `slug` is globally unique so that URLs are human-readable
  (`/spaces/engineering`).
- `settings` (JSONB) holds space-level configuration: default review interval,
  notification rules, AI feature toggles.

---

### 3.3 `pages`

The central entity. Each page belongs to a space, may have a parent page (for
nesting), and stores its rich-text content as a JSONB block tree.

```sql
CREATE TABLE pages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    space_id      UUID         NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    parent_id     UUID         REFERENCES pages(id) ON DELETE SET NULL,
    title         VARCHAR(500) NOT NULL,
    slug          VARCHAR(500) NOT NULL,
    content       JSONB        NOT NULL DEFAULT '{}',
    content_text  TEXT         GENERATED ALWAYS AS (content->>'text') STORED,
    position      INTEGER      NOT NULL DEFAULT 0,
    depth         INTEGER      NOT NULL DEFAULT 0,
    icon          VARCHAR(50),
    cover_image   TEXT,
    is_template   BOOLEAN      DEFAULT FALSE,
    created_by    UUID         NOT NULL REFERENCES users(id),
    updated_by    UUID         NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(space_id, slug)
);

CREATE INDEX idx_pages_space       ON pages(space_id);
CREATE INDEX idx_pages_parent      ON pages(parent_id);
CREATE INDEX idx_pages_content_gin ON pages USING GIN(content);
```

**Design decisions:**

- `content` holds the full ProseMirror-compatible JSON AST (see Section 7).
- `content_text` is a generated column that extracts a plain-text representation for
  full-text search without parsing JSONB at query time.
- `parent_id` with `ON DELETE SET NULL` means deleting a parent promotes children to
  root level rather than cascading a destructive delete.
- `position` and `depth` support drag-and-drop reordering in the sidebar tree.
- The `(space_id, slug)` uniqueness constraint ensures slugs are unique within a
  space but can be reused across spaces.
- The GIN index on `content` enables containment queries (`@>`) and key-existence
  checks against the JSONB block tree.

---

### 3.4 `page_versions`

Immutable audit trail of every edit. Each row is a full snapshot of the page at a
point in time.

```sql
CREATE TABLE page_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id         UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    version         INTEGER      NOT NULL,
    title           VARCHAR(500) NOT NULL,
    content         JSONB        NOT NULL,
    change_summary  TEXT,
    created_by      UUID         NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(page_id, version)
);
```

**Design decisions:**

- Full snapshots (not deltas) are stored for simplicity and fast retrieval. The
  storage cost is acceptable for a knowledge base where individual documents are
  relatively small.
- Diffs are computed at read time by comparing two JSONB snapshots.
- `change_summary` is optional human- or AI-generated text describing what changed.
- `(page_id, version)` uniqueness ensures monotonically increasing version numbers
  per page.

---

### 3.5 `freshness_records`

Tracks the freshness lifecycle of every page. This table is the backbone of
Knomantem's staleness detection and review workflow.

```sql
CREATE TABLE freshness_records (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id              UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    owner_id             UUID         REFERENCES users(id),
    freshness_score      DECIMAL(5,2) NOT NULL DEFAULT 100.00,
    review_interval_days INTEGER      NOT NULL DEFAULT 90,
    last_reviewed_at     TIMESTAMPTZ,
    next_review_at       TIMESTAMPTZ,
    last_verified_by     UUID         REFERENCES users(id),
    last_verified_at     TIMESTAMPTZ,
    status               VARCHAR(20)  NOT NULL DEFAULT 'fresh',
    decay_rate           DECIMAL(5,4) NOT NULL DEFAULT 0.0100,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_status CHECK (status IN ('fresh', 'aging', 'stale', 'expired'))
);

CREATE INDEX idx_freshness_page        ON freshness_records(page_id);
CREATE INDEX idx_freshness_status      ON freshness_records(status);
CREATE INDEX idx_freshness_next_review ON freshness_records(next_review_at);
```

**Design decisions:**

- `freshness_score` is a numeric value (0.00 to 100.00) rather than just a status
  enum so that dashboards can show gradients and trends.
- `decay_rate` is configurable per page: API documentation may decay faster than
  company policies.
- `owner_id` identifies who is responsible for reviewing the page, which may differ
  from the page creator.
- The index on `next_review_at` powers the background job that finds pages due for
  review.

---

### 3.6 `graph_edges`

Stores typed, directed relationships between pages. This table turns Knomantem's
page tree into a knowledge graph.

```sql
CREATE TABLE graph_edges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    target_page_id  UUID        NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    edge_type       VARCHAR(50) NOT NULL,
    metadata        JSONB       DEFAULT '{}',
    created_by      UUID        NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_edge_type CHECK (
        edge_type IN ('references', 'depends_on', 'supersedes', 'related_to', 'child_of', 'backlink')
    ),
    CONSTRAINT uq_edge UNIQUE(source_page_id, target_page_id, edge_type)
);

CREATE INDEX idx_edges_source ON graph_edges(source_page_id);
CREATE INDEX idx_edges_target ON graph_edges(target_page_id);
CREATE INDEX idx_edges_type   ON graph_edges(edge_type);
```

**Design decisions:**

- Edges are directed: `source_page_id` -> `target_page_id`. The direction carries
  semantic meaning (A *depends on* B is not the same as B *depends on* A).
- The `uq_edge` constraint prevents duplicate edges of the same type between the same
  pair of pages, while still allowing multiple edge types (a page can both
  *reference* and *depend on* another page).
- `metadata` (JSONB) is reserved for future use: edge weights, AI confidence scores,
  or annotations.
- `ON DELETE CASCADE` on both foreign keys ensures that deleting a page automatically
  removes all its inbound and outbound edges.

---

### 3.7 `tags` and `page_tags`

A two-table design for tagging, supporting both human-applied and AI-suggested tags.

```sql
CREATE TABLE tags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    color           VARCHAR(7),
    is_ai_generated BOOLEAN      DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE page_tags (
    page_id        UUID         NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id         UUID         NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    added_by       UUID         REFERENCES users(id),
    is_ai_suggested BOOLEAN     DEFAULT FALSE,
    confidence     DECIMAL(3,2),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (page_id, tag_id)
);
```

**Design decisions:**

- Tags are globally unique by name to prevent duplicates like "api" vs "API".
  Normalization should happen at the application layer before insert.
- `color` uses a hex string (e.g., `#3B82F6`) for UI rendering.
- `is_ai_suggested` and `confidence` on `page_tags` allow the UI to distinguish
  human-applied tags from AI-suggested ones, and to show confidence levels for the
  latter.
- The composite primary key `(page_id, tag_id)` prevents the same tag from being
  applied to a page twice.

---

### 3.8 `permissions`

Row-level access control for spaces and pages.

```sql
CREATE TABLE permissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type    VARCHAR(20) NOT NULL,
    resource_id      UUID        NOT NULL,
    permission_level VARCHAR(20) NOT NULL,
    granted_by       UUID        REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_resource_type CHECK (resource_type IN ('space', 'page')),
    CONSTRAINT chk_permission    CHECK (permission_level IN ('viewer', 'editor', 'admin', 'owner')),
    UNIQUE(user_id, resource_type, resource_id)
);
```

**Design decisions:**

- Polymorphic reference via `resource_type` + `resource_id` instead of separate
  `space_permissions` and `page_permissions` tables. This keeps the permission model
  in one place and simplifies authorization queries.
- `resource_id` is not a foreign key because it references different tables depending
  on `resource_type`. Referential integrity is enforced at the application layer.
- Permission levels are hierarchical: `owner` > `admin` > `editor` > `viewer`.
  The application layer interprets this ordering.
- `granted_by` provides an audit trail of who gave access.

---

## 4. Freshness Scoring Algorithm

The freshness system is the core differentiator of Knomantem. It models knowledge
decay as a continuous process rather than a binary fresh/stale flag.

### 4.1 Score Calculation

```
score = max(0, 100 - (days_since_review / review_interval_days * 100))
```

| Variable | Description |
|---|---|
| `days_since_review` | `NOW() - last_reviewed_at`, in fractional days |
| `review_interval_days` | Configurable per page (default: 90) |

The score is clamped to the range `[0, 100]`.

### 4.2 Status Thresholds

| Status | Score Range | Meaning |
|---|---|---|
| `fresh` | 80 -- 100 | Recently verified; no action needed |
| `aging` | 50 -- 79 | Review recommended soon |
| `stale` | 20 -- 49 | Likely outdated; review required |
| `expired` | 0 -- 19 | Considered unreliable until re-verified |

### 4.3 Score Reset and Boost Events

The following events reset or boost the freshness score:

| Event | Effect |
|---|---|
| Manual verification (user clicks "Still accurate") | Score reset to 100; `last_verified_at` updated |
| Content edit | Score reset to 100; `last_reviewed_at` updated |
| External link validation passes | Score boosted by +10 (capped at 100) |
| AI review confirms content is current | Score boosted by +15 (capped at 100) |

### 4.4 Background Job

A scheduled job runs **every hour** and performs the following:

1. Recalculates `freshness_score` for all pages where
   `next_review_at <= NOW()` or where the score has drifted since the last
   calculation.
2. Updates `status` based on the new score and the threshold table above.
3. Sends notifications to `owner_id` when a page transitions from `fresh` to
   `aging` or from `aging` to `stale`.
4. Generates a daily digest email listing all pages in `stale` or `expired` status
   for each space owner.

**SQL for the recalculation step:**

```sql
UPDATE freshness_records
SET
    freshness_score = GREATEST(0, 100 - (
        EXTRACT(EPOCH FROM (NOW() - last_reviewed_at)) / 86400.0
        / review_interval_days * 100
    )),
    status = CASE
        WHEN freshness_score >= 80 THEN 'fresh'
        WHEN freshness_score >= 50 THEN 'aging'
        WHEN freshness_score >= 20 THEN 'stale'
        ELSE 'expired'
    END,
    updated_at = NOW()
WHERE last_reviewed_at IS NOT NULL;
```

---

## 5. Knowledge Graph Model

### 5.1 Edge Types

| Type | Direction | Semantics |
|---|---|---|
| `references` | A -> B | Page A cites or mentions page B |
| `depends_on` | A -> B | Page A requires page B to be accurate/current |
| `supersedes` | A -> B | Page A replaces page B (B is outdated) |
| `related_to` | A <-> B | Bidirectional conceptual relationship |
| `child_of` | A -> B | Page A is a sub-page of page B (structural) |
| `backlink` | A -> B | Auto-generated: page B contains a link to page A |

### 5.2 Automatic Backlink Creation

When a user adds a `page_link` node in the content JSONB that points to page B, the
application layer automatically inserts a `backlink` edge from B to A. This ensures
that every page knows who links to it without requiring a full-text scan.

### 5.3 Graph Traversal Queries

Graph queries use recursive CTEs. Below are the key patterns.

**Find all pages that depend on a given page (direct and transitive):**

```sql
WITH RECURSIVE deps AS (
    SELECT source_page_id, target_page_id, 1 AS depth
    FROM graph_edges
    WHERE target_page_id = $1
      AND edge_type = 'depends_on'

    UNION ALL

    SELECT e.source_page_id, e.target_page_id, d.depth + 1
    FROM graph_edges e
    JOIN deps d ON e.target_page_id = d.source_page_id
    WHERE e.edge_type = 'depends_on'
      AND d.depth < 10  -- guard against cycles
)
SELECT DISTINCT source_page_id, depth
FROM deps
ORDER BY depth;
```

**Find all related content within 2 hops:**

```sql
WITH RECURSIVE related AS (
    SELECT target_page_id AS page_id, 1 AS hops
    FROM graph_edges
    WHERE source_page_id = $1
      AND edge_type = 'related_to'

    UNION ALL

    SELECT e.target_page_id, r.hops + 1
    FROM graph_edges e
    JOIN related r ON e.source_page_id = r.page_id
    WHERE e.edge_type = 'related_to'
      AND r.hops < 2
)
SELECT DISTINCT page_id, MIN(hops) AS min_hops
FROM related
GROUP BY page_id
ORDER BY min_hops;
```

**Find what a page supersedes (full chain):**

```sql
WITH RECURSIVE chain AS (
    SELECT target_page_id, 1 AS depth
    FROM graph_edges
    WHERE source_page_id = $1
      AND edge_type = 'supersedes'

    UNION ALL

    SELECT e.target_page_id, c.depth + 1
    FROM graph_edges e
    JOIN chain c ON e.source_page_id = c.target_page_id
    WHERE e.edge_type = 'supersedes'
      AND c.depth < 20
)
SELECT target_page_id, depth
FROM chain
ORDER BY depth;
```

---

## 6. Versioning Strategy

### 6.1 Write Path

Every save operation follows this sequence:

1. Begin transaction.
2. Compute the next version number: `SELECT COALESCE(MAX(version), 0) + 1 FROM page_versions WHERE page_id = $1`.
3. Insert a new row into `page_versions` with the full content snapshot.
4. Update the `pages` row with the new content, title, `updated_by`, and `updated_at`.
5. Reset the freshness score to 100 and update `last_reviewed_at`.
6. Commit transaction.

### 6.2 Read Path

- **Current content:** Read directly from `pages` (single row lookup, no join).
- **Version history:** Query `page_versions` ordered by `version DESC`.
- **Diff between versions:** Load two `page_versions` rows and compute a JSON diff
  at the application layer. Libraries like `jsondiffpatch` handle nested structure
  comparison.

### 6.3 Trade-offs

| Approach | Pros | Cons |
|---|---|---|
| Full snapshots (chosen) | Simple reads, no reconstruction | Higher storage per version |
| Delta/diff storage | Lower storage | Complex reconstruction, slower reads |
| Operational transforms | Real-time collaboration | Significant implementation complexity |

Full snapshots were chosen for the initial implementation because knowledge base
documents are typically small (under 100 KB) and the read-to-write ratio is high.
Storage cost is negligible compared to implementation simplicity.

### 6.4 Future Consideration

For real-time collaborative editing, the system could layer operational transforms
(OT) or CRDTs on top of the current model. The snapshot-based version history would
still serve as periodic checkpoints.

---

## 7. JSONB Content Structure

Page content is stored as a ProseMirror-compatible JSON abstract syntax tree (AST).
This structure maps directly to the editor's internal document model, eliminating
serialization overhead.

### 7.1 Document Schema

```json
{
  "type": "doc",
  "content": [
    {
      "type": "heading",
      "attrs": { "level": 1 },
      "content": [
        { "type": "text", "text": "Page Title" }
      ]
    },
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Normal text with " },
        { "type": "text", "text": "bold", "marks": [{ "type": "bold" }] },
        { "type": "text", "text": " formatting." }
      ]
    },
    {
      "type": "page_link",
      "attrs": {
        "page_id": "uuid-here",
        "edge_type": "references"
      }
    }
  ]
}
```

### 7.2 Node Types

| Node Type | Description | Key Attributes |
|---|---|---|
| `doc` | Root node; exactly one per page | -- |
| `heading` | Section heading | `level` (1-6) |
| `paragraph` | Block of inline content | -- |
| `text` | Inline text leaf | `marks` (bold, italic, code, link) |
| `page_link` | Inline link to another Knomantem page | `page_id`, `edge_type` |
| `code_block` | Fenced code block | `language` |
| `blockquote` | Quoted block | -- |
| `bullet_list` | Unordered list | -- |
| `ordered_list` | Ordered list | `start` |
| `list_item` | Item within a list | -- |
| `image` | Embedded image | `src`, `alt`, `title` |
| `table` | Table container | -- |
| `table_row` | Row within a table | -- |
| `table_cell` | Cell within a row | `colspan`, `rowspan` |
| `callout` | Highlighted info/warning box | `variant` (info, warning, error) |

### 7.3 Mark Types

Marks are inline formatting annotations applied to `text` nodes.

| Mark | Rendered As |
|---|---|
| `bold` | **bold** |
| `italic` | *italic* |
| `code` | `inline code` |
| `strike` | ~~strikethrough~~ |
| `link` | Hyperlink (external URL) |
| `highlight` | Background highlight |

### 7.4 Graph Edge Extraction

When content is saved, the application layer traverses the AST looking for
`page_link` nodes and synchronizes the `graph_edges` table:

1. Collect all `page_link` nodes from the new content.
2. Compare against existing edges from this page.
3. Insert new edges, delete removed edges, update changed edge types.
4. For each new edge of type `references`, insert a corresponding `backlink` edge
   in the reverse direction.

This ensures the knowledge graph always reflects the actual content.

---

## 8. Index Strategy Summary

| Index | Table | Type | Purpose |
|---|---|---|---|
| `idx_pages_space` | pages | B-tree | Filter pages by space |
| `idx_pages_parent` | pages | B-tree | Tree navigation queries |
| `idx_pages_content_gin` | pages | GIN | JSONB containment queries on content |
| `idx_freshness_page` | freshness_records | B-tree | Look up freshness by page |
| `idx_freshness_status` | freshness_records | B-tree | Dashboard queries by status |
| `idx_freshness_next_review` | freshness_records | B-tree | Background job: find due reviews |
| `idx_edges_source` | graph_edges | B-tree | Outbound edge lookups |
| `idx_edges_target` | graph_edges | B-tree | Inbound edge lookups |
| `idx_edges_type` | graph_edges | B-tree | Filter edges by semantic type |

Future indexes to consider:

- `GIN(to_tsvector('english', content_text))` on `pages` for full-text search.
- Partial index on `freshness_records` where `status IN ('stale', 'expired')` for
  faster dashboard queries.
- `BRIN` index on `page_versions(created_at)` for time-range queries over version
  history.
