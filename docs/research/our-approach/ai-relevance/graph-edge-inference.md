# AI Relevance: Graph Edge Inference

## What the planned use case is

The architecture describes an "AI Service" background worker responsible for "relation extraction" — automatically discovering and suggesting typed edges between pages. Currently all edges are 100% manually created via `POST /pages/:id/graph/edges`. The six edge types are: `reference`, `depends_on`, `supersedes`, `related_to`, `child_of`, `backlink`.

The AI goal is to: (1) auto-create obvious edges without user intervention, (2) suggest likely edges for user approval, (3) eventually infer edge type rather than just that a connection exists.

---

## Does it need AI?

**Strongly split by edge type. Some types need zero AI; others need LLM-level reasoning.**

### Edges that do NOT need AI

**`backlink` / `reference`** — pure string parsing:
- When page content (ProseMirror JSON) includes a node of type `link` or a mention of another page ID, a `reference`/`backlink` edge should be created automatically.
- This is already noted in the architecture sequence diagram: `EdgeRepository.CreateImplicit(page)` is called on page creation.
- Implementation: traverse the JSONB content tree, extract all link nodes, resolve to page IDs, create edges. No ML involved.
- This is the single highest-value, lowest-effort edge automation. It should already be implemented in `page_service.go` but isn't yet.

**`child_of`** — tree structure already exists:
- Pages have a `parent_id` field in the database.
- A `child_of` edge should be automatically created/maintained whenever `parent_id` is set.
- This is a DB trigger or a hook in `page_service.go`. No ML.

### Edges that need embeddings (not LLM)

**`related_to`** — semantic similarity:
- Two pages are "related" if their content vectors are close in embedding space.
- This is a standard ANN (approximate nearest neighbor) query over page embeddings stored in pgvector.
- Threshold: cosine similarity > 0.75 → suggest `related_to` edge.
- No LLM needed. This is a well-understood ML task that embedding models handle reliably.

### Edges that genuinely need LLM reasoning

**`depends_on`** — requires understanding causality and dependency chains:
- "Page A depends on Page B" means: the information in A is only valid if the information in B is also valid/current.
- Examples: "API Authentication Guide" depends on "OAuth Configuration Reference". A page about "Deployment Process" depends on "Infrastructure Architecture".
- Embedding similarity alone is insufficient — a "Retrospective 2024-Q3" page and a "Retrospective 2024-Q4" page will be very similar in embedding space but don't depend on each other.
- Detecting dependency requires understanding the *directionality and causal relationship* between two pages, which requires LLM reasoning.

**`supersedes`** — requires temporal and semantic reasoning:
- "New Onboarding Guide (2025)" supersedes "Old Onboarding Guide (2023)".
- Signals: similar titles with date/version indicators, similar content but one is newer, explicit "this replaces" language in content.
- Heuristic for title pattern matching: doable as a script (regex + recency check). But for detecting when a completely differently-named page covers the same ground and is newer → needs LLM.

---

## Recommendation

**Implement in three distinct phases, don't bundle them:**

**Phase 1 (ship now, no AI):**
- Auto-create `backlink`/`reference` edges from page content link nodes (ProseMirror traversal)
- Auto-create/sync `child_of` edges from `parent_id` on page save

**Phase 2 (embeddings, no LLM):**
- Store page embeddings in pgvector on page create/update
- Background worker: ANN query for each page, suggest `related_to` edges above similarity threshold
- Surface suggestions in the graph screen UI as "suggested edges" awaiting user confirmation (don't auto-create without approval for typed edges)

**Phase 3 (LLM, selective):**
- Run LLM inference only on page pairs that: (a) share high embedding similarity AND (b) have no existing typed edge
- Prompt: given two page titles and their first 300 tokens, classify the relationship: depends_on / supersedes / none
- This is a classification task, not generation — a fine-tuned small model or Claude claude-haiku-4-5-20251001 with a structured prompt works

**Cost control**: never auto-create `depends_on` or `supersedes` edges without user confirmation. LLM cost is justified only for suggestions shown to users, not for bulk background processing of all page pairs.
