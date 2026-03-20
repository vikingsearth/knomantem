# Page Creation Pipeline — Industry Standards & Our Approach

*Research date: 2026-03-20*

---

## 1. Slug Generation

### Industry standards

**What a slug is**
A URL slug is a human-readable, ASCII-safe path segment derived from an entity's title. The canonical rules applied across major web frameworks (Django, Rails, WordPress) and CMS platforms are:

1. Lowercase the entire string.
2. Transliterate/strip non-ASCII characters (Unicode normalisation to NFD, then drop combining characters).
3. Replace any sequence of characters that are not `[a-z0-9]` with a single hyphen.
4. Trim leading and trailing hyphens.
5. Truncate to a maximum length — 100 characters is a common ceiling (WordPress: 200, Confluence: 255 bytes, but URL best-practice recommends ≤100 for readability).

**Uniqueness scope**
KMS platforms scope slugs to their nearest container:

- Confluence: slug is unique per space.
- Notion: pages have UUIDs in URLs; human slugs are not enforced.
- Ghost/WordPress: slug is unique globally across posts.
- **Our system**: the schema enforces `UNIQUE (space_id, slug)`, so uniqueness is per-space.

**Collision resolution**
The standard pattern is an append-and-retry loop:

```
candidate = slugify(title)            # "my-page"
for n in [2, 3, 4, ...]:
    try INSERT with candidate
    if unique constraint error:
        candidate = base + "-" + str(n)  # "my-page-2", "my-page-3"
    else:
        break
```

An alternative is to check existence first (`SELECT COUNT(*) WHERE space_id=$1 AND slug=$2`) before inserting, then attempt with suffix if taken. The retry-on-conflict approach is preferred for correctness under concurrent creation (avoids TOCTOU races) and aligns with how PostgreSQL unique constraint errors surface via pgx (`23505`).

**Unicode handling**
The `golang.org/x/text/unicode/norm` package (NFD normalisation + stripping `Mn` (non-spacing mark) category) is the idiomatic Go approach. For our current needs — where titles are typically ASCII or lightly accented — a simple `strings.ToLower` + regex replace is adequate and avoids an extra dependency. The `slugRe` pattern already in `space_service.go` (`[^a-z0-9-]+`) handles this correctly.

**Max length**
We truncate candidates to 100 characters before attempting collision resolution to stay well under the `VARCHAR(500)` column limit and keep URLs scannable.

### Our approach

- Copy the existing `slugify` helper (already in `space_service.go`) into `page_service.go` (same package — one definition is sufficient; both files share `package service`).
- In `PageService.Create`, generate `slug = slugify(req.Title)`.
- Attempt `pages.Create`; if the error wraps `domain.ErrConflict`, append `-2`, `-3`, etc. and retry up to 10 times.
- No new repository method is required — the unique constraint on `(space_id, slug)` in the DB provides the authoritative check.

---

## 2. Freshness Record Initialisation

### Industry standards

**Eager vs lazy initialisation**
Two patterns exist for derived/computed records that must exist alongside a parent entity:

| Pattern | Trigger | Pro | Con |
|---------|---------|-----|-----|
| **Eager (synchronous)** | Inside the same transaction as the parent INSERT | Consistency guaranteed; no "missing record" window | Slightly more work per creation |
| **Lazy (on-demand)** | First read that requires the derived record | Simple creation path | Dashboard/query code must handle NULL; causes 500s if unchecked |
| **Async / event-driven** | Message queue subscriber | Decouples services | Temporal gap; complexity |

Enterprise KMS platforms uniformly use **eager initialisation** for freshness / staleness metadata:

- **Confluence Page Properties**: page-scoped metadata is written at page creation time, not deferred.
- **SharePoint "Last Reviewed" policies**: columns are populated by the system at document creation.
- **Notion**: last-edited timestamps and owner fields are set at creation with no deferred step.

The reason is simple: a missing freshness record causes immediate downstream failures (dashboards, background decay workers, notification queues). Lazy initialisation imposes defensive null-handling everywhere.

**Default values rationale**

| Field | Default | Rationale |
|-------|---------|-----------|
| `freshness_score` | 100.0 | Brand-new content is fully fresh; the decay worker will reduce it over time |
| `review_interval_days` | 30 | One month is the standard first-review cycle used by Confluence and Notion staleness prompts |
| `decay_rate` | 0.0333 | ≈ 1/30; score decays roughly 1 point per day, reaching ~0 after one interval |
| `status` | "fresh" | Matches the DB constraint enum; consistent with score=100 |
| `last_reviewed_at` | now() | The act of creating is itself the first review |
| `next_review_at` | now() + 30 days | Triggers the first review prompt after one interval |
| `owner_id` | creating user | Ownership defaults to author; can be reassigned later |

**Transactional safety**
Because both the page INSERT and freshness INSERT must succeed or both must fail, they should ideally run in the same PostgreSQL transaction. However, using a lightweight retry-on-conflict approach (catching `domain.ErrConflict` from the freshness insert if the page was created twice) is acceptable if full transactional wiring is not yet in place. The uniqueness constraint on `page_freshness(page_id)` provides the backstop.

### Our approach

- After `pages.Create` succeeds in `PageService.Create`, build a `domain.Freshness` with the default values above and call `freshness.Create`.
- The error from `freshness.Create` is treated as non-fatal if it is `domain.ErrConflict` (idempotent re-creation), but is propagated for all other error types.
- `FreshnessRepository` is injected into `PageService` as a new struct field and constructor parameter.
- `cmd/server/main.go` already instantiates `freshnessRepo`; it just needs to be passed to `NewPageService`.

---

## 3. Implicit Link Extraction (Backlink Edges)

### Industry standards

**How modern KMS tools handle auto-linking**

| Tool | Trigger | Link format | Edge direction | Cycle detection |
|------|---------|-------------|---------------|-----------------|
| **Obsidian** | On save (file write) | `[[Page Title]]` wikilinks; resolves to file path | Bidirectional metadata; backlinks panel lists all inbound links | No enforcement; cycles are valid in a knowledge graph |
| **Roam Research** | On block save / real-time | `[[Page Title]]` creates or navigates to page | Bidirectional; every link auto-creates a backlink | No enforcement; cycles are intentional |
| **Notion** | On publish / content save | `@mention` or inline page links (UUID-based) | Backlinks shown in page; underlying directed edge source→target | No enforcement |
| **Confluence** | On page save | Inline hyperlinks; "Page Links" feature | Directed; backlinks shown in "Linked pages" panel | No enforcement |

**Key design decisions observed**

1. **Trigger**: link extraction runs at **save time** (page create or update), not asynchronously. This keeps backlinks current with content.
2. **Link format**: modern tools trend toward ID-based references (`/pages/UUID`) rather than title-based (`[[Title]]`) because IDs survive renames. Title-based linking requires re-scanning on rename.
3. **Cycle detection**: none of the major tools prevent cycles in the knowledge graph. Cycles are semantically valid (A references B which references A is a legitimate bidirectional relationship). Cycle prevention is only applied in strict hierarchies (parent-child trees), not in reference graphs.
4. **Deduplication**: all tools prevent duplicate edges (same source, target, type). The standard mechanism is a unique constraint on `(source_page_id, target_page_id, edge_type)` plus idempotent upsert logic. Catching a `23505` unique-violation and treating it as success is the canonical pattern.
5. **Backlink maintenance**: when a link is removed from content, old edges should be deleted or flagged. A clean implementation re-diffs: delete all `reference` edges from the page, then re-insert current ones. A simpler implementation (insert-only, no delete) accumulates stale edges but is acceptable for an initial implementation.

**ProseMirror link node format**
ProseMirror (used by our system) encodes hyperlinks as marks on text nodes:

```json
{
  "type": "text",
  "text": "See this page",
  "marks": [
    {
      "type": "link",
      "attrs": {
        "href": "/pages/550e8400-e29b-41d4-a716-446655440000",
        "title": "Page Title"
      }
    }
  ]
}
```

UUID extraction uses a regex: `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`.

### Our approach

- On `PageService.Create`, after persisting the page, walk the ProseMirror JSON content depth-first.
- For every `text` node that carries a `link` mark, extract the `href` attribute.
- Parse UUIDs from the href string (support both `/pages/UUID` and bare `UUID` formats).
- Skip self-links (source == target).
- For each valid target UUID, call `edges.Create` with `edge_type = "reference"` and `created_by = uid`.
- Catch `domain.ErrConflict` errors (unique constraint) silently — the edge already exists.
- `EdgeRepository` is already a field on `PageService`; no new injection is needed.
- This same logic should eventually be called from `PageService.Update` when content changes, but for this task it is wired at creation time only.
