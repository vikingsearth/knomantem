# Next Steps: Path to Working MVP Demo

## Critical Gaps (must close for a demo that sells the product)

The backend is production-quality. The frontend data layer is solid. The two integration gaps that make the product undemoable are:

### 1. Rich Text Editor (Blocking)

`appflowy_editor` is not in `pubspec.yaml`. The page editor screen renders nothing useful. Without this, the product cannot function as a knowledge base — you can't write anything.

**Resolution:** Add `appflowy_editor` to pubspec and wire it into `page_editor_screen.dart`. The backend stores ProseMirror-compatible JSON (JSONB); AppFlowy Editor uses its own document format. Either:
- (a) Use AppFlowy Editor and write a serialization adapter between its delta format and ProseMirror JSON on save/load
- (b) Switch to `flutter_quill` which is closer to the delta format and more actively maintained
- (c) Use a simpler `markdown_editor_plus` or similar for the MVP and accept reduced formatting richness

This is a non-trivial decision that affects the content model long-term. Pick it explicitly.

### 2. Graph View Rendering (Blocking for the "wow" demo moment)

`graph_screen.dart` loads data correctly. `GraphView` widget receives nodes and edges. But the actual canvas rendering in `graph_view.dart` is unverified. A force-directed graph layout is non-trivial in Flutter. This is the single most important visual feature.

**Resolution:** Read `graph_view.dart` and assess honestly. If it's a placeholder:
- Option A: Use `graphview` Flutter package (force-directed, tree, layered layouts)
- Option B: Custom `CustomPainter` with spring simulation (2-3 days for a basic version)
- Option C: Render via a web view embedding a JavaScript graph library (d3-force, vis.js) — works well for web target

The graph visualization must be functional and visually polished before any demo. Everything else can be rough.

---

## Priority Sequence

Order based on "what unblocks what":

| Priority | Work | Unblocks |
|---|---|---|
| 1 | Decide on and install rich text editor | Page creation, editing, all content workflows |
| 2 | Implement/verify GraphView canvas rendering | The core demo feature |
| 3 | Integrate rich text editor into page_editor_screen | Full page CRUD cycle |
| 4 | Freshness-weighted search ranking (10 lines in search_service.go) | The #2 differentiator becomes real |
| 5 | Auto-create backlink edges from page content | Graph population without manual work |
| 6 | Freshness dashboard screen in Flutter | Full freshness workflow visible to user |
| 7 | Fix freshness worker to decay Aging pages too (score 30-70) | Correct staleness behavior |
| 8 | Replace localhost hardcode with env config in Flutter | Any shared/deployed testing |
| 9 | WebSocket client in Flutter for presence | Collaboration awareness feature |
| 10 | Write integration tests for critical paths | Confidence in stability as features are added |

---

## Architectural Decisions Needed Before Going Further

### 1. Rich text editor choice (decide this week)

The entire content strategy depends on this. Options:
- **AppFlowy Editor**: as planned, Flutter-native, actively developed, but document format is not ProseMirror
- **flutter_quill**: Quill Delta format, more mature ecosystem, adapters exist for conversion
- **Markdown-based**: simpler, lower fidelity, but avoids complex serialization problems

The backend JSONB schema is flexible — it can store any JSON document format. The decision should be driven by editor capability and long-term maintenance, not backend constraints.

### 2. Embedding model strategy (before Phase 2 AI work)

Before building auto-tagging, graph inference, or duplicate detection, decide:
- Will we call the Anthropic API for embeddings? (cost, privacy concerns for self-hosters)
- Will we run a local embedding model? (self-hosted, needs sidecar process)
- pgvector vs. external vector DB?

This decision affects the AI service architecture and should be documented before anyone starts building Phase 2.

### 3. OpenAPI spec as source of truth

The API contract currently exists only as convention. Before adding more features, generate or write `docs/openapi.yaml`. Consider using `swaggo/swag` for Go to auto-generate from handler annotations. This prevents drift and enables Flutter client code generation.

---

## Suggested Worktree Work Split

| Worktree | Branch | Suggested Focus |
|---|---|---|
| `wt1` / `feat/start1` | AI research | ✅ Done — AI relevance analysis |
| `wt2` / `feat/start2` | Documentation | ✅ Done — frontend/backend/integration/next-steps analysis |
| `wt3` / `feat/start3` | Rich text editor | Add `appflowy_editor` (or chosen alternative), wire into page_editor_screen, implement ProseMirror serialization adapter, test round-trip create/read/update |
| `wt4` / `feat/start4` | Graph visualization | Implement `GraphView` canvas rendering (force-directed layout), freshness status color-coding on nodes, edge type display, node click navigation |

A fifth parallel track if available: **freshness-weighted search** — it's backend-only, 1-2 days, high impact. Could be a quick win in any free worktree.

---

## Risks and Open Questions

1. **AppFlowy Editor delta ↔ ProseMirror JSONB**: content stored as ProseMirror JSON in DB but editor uses different format. Conversion is lossy if not carefully engineered. Validate this round-trip before committing to AppFlowy Editor.

2. **GraphView performance at scale**: a force-directed layout with 200+ nodes in Flutter Canvas can be slow. Needs performance validation early. If slow, switch to a canvas-with-LOD approach or WebView + d3.

3. **Freshness decay bug**: the `RunDecay` worker only fetches pages with `score < 30` (already Stale) and applies decay to those. Pages in the Aging range (30-70) never get decay applied by the worker. This means a page at score 50 will stay at 50 indefinitely until it hits the stale threshold some other way. Likely a bug — the worker should process all pages, not just already-stale ones.

4. **Test coverage**: 7 test files for ~50 Go files. Any significant refactor or feature addition is flying blind. Before Phase 2, write at minimum: page service tests, search service tests, graph service tests, and one end-to-end test using Docker Compose.

5. **Flutter Riverpod version**: pubspec has `^2.6.1`, architecture doc says "Riverpod 3.0". The API differences between 2.x and 3.0 are significant (AsyncNotifier, etc.). Ensure all new Flutter development uses 2.x patterns or explicitly migrates to 3.0.
