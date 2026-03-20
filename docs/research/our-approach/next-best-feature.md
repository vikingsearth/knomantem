# Next Best Feature Report
*Knomantem — Prioritised recommendation as of 2026-03-20*

---

## 1. Where We Stand

**The backend is the strongest asset.** The Go API server is production-quality: clean architecture enforced, all major endpoints implemented (auth, spaces, pages, search, freshness, graph, tags, presence WebSocket), background workers running, a mature domain model, and a Docker image that deploys cleanly. Freshness is genuinely a first-class citizen — the decay engine runs, scores are computed, and notifications fire. This is real, not aspirational.

**The Flutter frontend data layer is solid but the UI has critical holes.** `api_service.dart` covers every backend endpoint and the models are well-aligned. Riverpod state management is correctly structured. Auth flows end-to-end. But two screens are broken in ways that make the product undemoable:

- `page_editor_screen.dart` has no rich text editor installed. `appflowy_editor` is absent from `pubspec.yaml`. The editor renders nothing useful. You cannot write content.
- `graph_view.dart` receives graph data correctly from the backend but its canvas rendering quality is unverified. This is the product's single "wow" moment and it may be a placeholder.

**What is missing entirely:** A freshness dashboard screen (the API method `getFreshnessDashboard()` exists but no Flutter screen renders it). No WebSocket client for presence. No Confluence/Notion import tool. The `ai_service.go` worker referenced in the architecture does not exist as a file — it is an aspirational placeholder.

**What is broken but fixable quickly:** The freshness decay worker only processes pages with `score < 30`. Pages in the Aging range (30–70) never decay. This is a bug, not a design choice. Search ranking ignores freshness entirely despite `SearchItem` carrying a `FreshnessBrief` — the multiplier formula is documented, researched, and ready to implement; it just hasn't been wired in.

The product has a strong skeleton. It is missing its face.

---

## 2. The Case for Each Candidate Feature

### Rich Text Editor Integration (AppFlowy Editor or alternative)

This is not a feature — it is a prerequisite. Without a functional editor, users cannot create meaningful content, which means the graph stays empty, the freshness system has nothing to decay, and search has nothing to index. Every other feature on this list depends on content existing in the system. The backend stores ProseMirror-compatible JSONB; the editor decision (AppFlowy vs flutter_quill vs markdown-based) must be made first because it determines the serialization path for all future content. The architecture doc calls for AppFlowy Editor, but `page-pipeline-standards.md` and `next-steps.md` both flag a non-trivial delta-to-ProseMirror adaptation problem. This needs a decision and immediate execution. **This is the single most urgent gap, but it is a gap, not a "feature" in the product differentiation sense.** It belongs in a category of its own: foundation.

### Graph Visualization (GraphView Canvas Rendering)

The graph is Knomantem's most differentiated visual element. No competitor — not Confluence, not Notion, not SharePoint — offers a typed, traversable, interactive knowledge graph. Obsidian has force-directed visualization but it is local-only, untyped, and has no team features. The backend graph API (`/graph/explore`, multi-hop traversal, depth and edge-type filters) is complete and well-designed. The Flutter `graph_screen.dart` correctly loads data, applies filters, and passes it to `GraphView`. The problem is that `GraphView`'s actual canvas rendering is unverified. If it is a placeholder, this is the highest-priority visual fix after the editor. If it renders something usable, it needs polish: freshness-color-coded nodes, visible edge type labels, and smooth navigation on tap. The competitive matrix is unambiguous: typed knowledge graphs are white space in the market. A stakeholder who sees a working graph with freshness-colored nodes will lean forward. This is the demo-closing feature.

### Freshness Dashboard Screen (Flutter UI)

The backend freshness dashboard endpoint (`GET /freshness/dashboard`) returns paginated, freshness-sorted pages with aggregate stats. The Flutter API method `getFreshnessDashboard()` is implemented. What is missing is the screen that renders it. This is the feature that makes freshness visible and actionable — not just a score on a single page, but a workspace-level view of knowledge health: stale pages, pages needing review, ownership gaps. The `how-to-win.md` identifies freshness as the headline differentiator and the competitive matrix confirms that no competitor has content decay analytics. This screen would be the product's "mission control" for knowledge health. It is entirely a Flutter UI task; no backend work is required. Estimated effort: 2–3 days for a solid v1.

### WebSocket Presence (Who's Viewing)

The backend presence hub (`WSS /presence/:pageId`) is implemented in Go and works as an in-memory hub. What does not exist is any WebSocket client in Flutter. Building presence requires building the WebSocket layer from scratch on the Flutter side. The `how-to-win.md` positions real-time collaboration as a Phase 2 feature, and the architecture doc explicitly defers full CRDT co-editing to Phase 3 (it is a massive engineering undertaking). Presence — showing "Alice is viewing this page" — is the lightweight version that provides meaningful collaboration awareness with low server-side cost. It is not a demo-closing feature: no stakeholder signs up for a KMS because they can see who else is viewing a page. It is a retention and team-feel feature. The effort (build WebSocket client, manage connection lifecycle, handle reconnection, render presence indicators in the page editor) is non-trivial and the payoff is incremental.

### Confluence/Notion Import Tool

`how-to-win.md` calls this out explicitly: "Build a one-click Confluence import tool early. Every Confluence workspace is a potential migration target." The argument is strategically correct — Confluence users are the largest addressable market segment and freshness scoring applied to imported content will immediately demonstrate value by surfacing years of rot. A Notion import that converts database Relations into graph edges would be a single demo moment that sells the graph feature without requiring the user to have built up their own content first. The import tool is a growth lever and a demo accelerant. However, it is also a non-trivial engineering effort: Confluence uses XML export format (space export ZIP), Notion uses a different JSON/Markdown export, and mapping either to Knomantem's JSONB content format + graph edges requires significant parsing and adaptation work. This belongs in the roadmap, but building it before the editor exists and the graph renders correctly would be building a front door to a house with no interior walls.

### AI Auto-Tagging (Phase 2)

The `auto-tagging.md` research is clear: a useful v1 can ship without AI (keyword matching against existing tags, TF-IDF, Levenshtein distance matching) and is a 2–3 day implementation. The schema already supports `confidence_score` on the `page_tags` junction table — this was designed in from day one. The genuine AI version (embeddings in pgvector) depends on the pgvector infrastructure that is also a prerequisite for graph edge inference and duplicate detection. All three Phase 2 AI features share the same embedding pipeline foundation: generate embeddings on page save, store in a `page_embeddings` table with HNSW index, query by cosine similarity. Building any one of them forces the embedding infrastructure decision (local MiniLM vs Anthropic API vs OpenAI). The `how-to-win.md` makes the right call: AI features in Phase 2, not Phase 1, because users need content mass for AI to provide visible value. Shipping auto-tagging before there are 50+ pages in a workspace produces suggestions nobody cares about.

---

## 3. Recommendation: The Single Next Best Feature

**Build the graph visualization first.**

Specifically: read `graph_view.dart`, assess honestly whether it renders anything compelling, and if not — implement a real force-directed graph canvas using the Flutter `graphview` package or a custom `CustomPainter` with spring physics. Then layer on: freshness-color-coded nodes (fresh = green, aging = amber, stale = red), visible edge type labels, tap-to-navigate, and the depth/edge-type filter controls already wired in `graph_screen.dart`.

The editor must also be addressed — it is a prerequisite, not a feature — but the graph is the strategic priority because it is the feature that wins in a demo room.

Here is why this wins over every other candidate:

**Demo impact is unmatched.** When a stakeholder sees a force-directed knowledge graph with color-coded freshness nodes, typed relationship labels, and multi-hop traversal controlled by a depth slider, there is a visible "this is different" moment that no competitor can match. Confluence has been asked for a knowledge graph on their community forums for years and has never delivered it. Notion has implicit relations but no graph surface. Obsidian has an impressive-looking graph but it is untyped, local-only, and has no team features. A 60-second graph demo closes a conversation faster than any other feature.

**Competitive differentiation is absolute.** The competitive matrix confirms that typed knowledge relationships with visual graph exploration is a capability gap across all four incumbent platforms. This is not incremental improvement over what exists — it is a category that does not exist in the market. Obsidian proves the demand; Knomantem delivers the team version.

**Technical unblocking is significant.** A working graph visualization validates the graph API, confirms the data model for edges is correct, and creates pressure to populate the graph — which in turn drives backlink auto-creation (the implicit edge work from `page-pipeline-standards.md` and `graph-edge-inference.md`), which increases graph density, which makes the graph more impressive in demos. The graph creates a virtuous loop. It also provides the visual surface that makes freshness dashboard integration compelling: a stale node on the graph is more viscerally meaningful than a stale row in a list.

**Effort vs payoff ratio is excellent.** The backend is done. The data load in Flutter is done. The filter controls are wired. The remaining work is the canvas rendering widget. Using the `graphview` Flutter package this is a 2–4 day implementation for a functional v1, and an additional 2–3 days for freshness-color-coding, edge labels, and polish. The payoff — a demo-closing visual feature that differentiates the product from every competitor — is disproportionate to the effort.

---

## 4. Why Not the Others

**Rich text editor:** It must be done, but it is a prerequisite that enables the product to function, not a differentiator. Every KMS has an editor. Nobody chooses a KMS because the editor is slightly better. Get it working to a functional standard quickly (pick `flutter_quill` for delta format alignment, accept the serialization adapter cost, and ship something that lets users write), then move on. Do not over-invest in editor perfection at this stage.

**Freshness dashboard screen:** This is the second priority (see Section 5), not the first. The reason it loses: a dashboard of stale pages requires users to have pages, which requires the editor to work, which requires the editor to be built first. The dashboard is also more compelling if users can see stale nodes on the graph simultaneously — building the graph first makes the freshness dashboard more impactful when it ships. The dashboard alone, before the graph exists, is a useful admin view but not a demo moment.

**WebSocket presence:** This is a quality-of-life feature for teams already using the product. It does not appear in a demo. It does not differentiate Knomantem in the market — Confluence, Notion, and SharePoint all have it, so matching them is table stakes, not a win. The effort required (build WebSocket client from scratch in Flutter, manage reconnection logic, render presence indicators, handle stale presence cleanup) is substantial for a feature that has no acquisition value. Build it after the core product sells itself.

**Confluence/Notion import tool:** This is the right tool at the wrong time. An import tool that dumps content into a broken editor and an unverified graph renderer produces a worse impression than starting fresh. Build the import tool after the editor and graph are solid. Then it becomes a genuine conversion accelerant: "import your Confluence space, and watch your knowledge rot become visible." Before that, it is a pipeline into a construction site.

**AI auto-tagging:** The keyword matching v1 is fast to build and genuinely useful, but it requires content to suggest tags for. Without content (because the editor is not working), there is nothing to tag. The embedding-based v2 requires a pgvector infrastructure decision that should be made deliberately, not in a rush. Phase 2 timing is correct: AI features earn their keep after users have enough content to make the suggestions meaningful. Building auto-tagging now would produce suggestions nobody sees and teach nobody anything useful.

---

## 5. After That: The Next 2

### Second: Rich Text Editor (functional, not perfect)

The editor is the product's prerequisite. Once the graph visualization proves the concept in a demo, the next conversation is "can I try it?" — and right now the answer is "you can create pages with trivial single-heading content." That is not good enough. Pick an editor, resolve the serialization question (strong recommendation: `flutter_quill` with a Quill Delta to ProseMirror JSONB adapter layer, since flutter_quill is more actively maintained and has more adapters in the ecosystem), wire it into `page_editor_screen.dart`, validate the round-trip (create page → save → load → content intact), and ship it. Simultaneously: implement slug generation and freshness record initialisation on page creation (documented in `page-pipeline-standards.md` and ready to code), and wire automatic backlink edge creation from content link nodes (documented in `graph-edge-inference.md`). These backend tasks should happen in the same sprint as the editor work because they depend on content being created.

### Third: Freshness Dashboard Screen

Once users can write content (editor) and see it connected (graph), the freshness dashboard becomes the feature that creates ongoing value and drives retention. It answers the question every engineering manager eventually asks: "where is our stale knowledge?" The backend endpoint is complete. The Flutter API method exists. The Riverpod provider structure just needs a `freshness_provider.dart` for dashboard data and a new screen. Build a two-panel layout: aggregate stats at the top (total pages, percentage fresh/aging/stale, average score), followed by a sortable, filterable list of pages ranked by freshness score ascending. Add a "verify" action inline. Estimated effort: 2–3 days. The payoff is converting freshness from an invisible backend concept into a visible product surface that users check regularly — which is what turns a demo feature into a retention feature.

After these three — graph, editor, freshness dashboard — the product is demonstrable, usable, and retentive. The import tool becomes the acquisition accelerant. Presence becomes the collaboration layer. AI auto-tagging arrives when there is content to tag. The sequence holds.

---

*This report is based on research conducted across 13 documents in `docs/research/our-approach/` and the strategic findings in `docs/findings/how-to-win.md`, `docs/findings/architecture.md`, and `docs/findings/competitive-matrix.md`.*
