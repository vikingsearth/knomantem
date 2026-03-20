# AI Relevance: Content Summarization

## What the planned use case is

Phase 3 roadmap includes "AI content summarization" — auto-generated summaries for long pages and page clusters. This would appear in: (1) search result excerpts that are more meaningful than raw text snippets, (2) page cards in the graph view showing what a page is about without opening it, (3) space-level "what's in this space" overviews, (4) staleness notifications ("this page covers X, Y, Z — is it still accurate?").

---

## Does it need AI?

**Yes, but scope it carefully. Generic summarization of ProseMirror content genuinely requires LLM.**

### What does NOT need AI

**Excerpt generation for search results** — the current `SearchItem.Excerpt` field is already populated. A good excerpt is the first paragraph + highlighted keyword context. This is extractive, not generative, and works well without AI. Bleve can return highlighted text snippets natively.

**"First N words" page cards** — for graph node tooltips showing what a page is about, extracting the first heading + first paragraph is good enough in most cases. This is pure string extraction from the ProseMirror JSONB AST.

**Structure-based summaries** — a page with H1-H4 headings already has an implicit outline. Extracting headings and rendering them as a structured summary is a tree traversal, not ML. Useful for the "page overview" sidebar.

### What genuinely needs AI

**Coherent prose summaries** — when users need a 2-3 sentence summary of a long, complex page (e.g., an architecture decision record, a design spec), extracting the first paragraph often gives the background/context, not the conclusion. An LLM can read the full page and write a summary that captures the key point.

**Page cluster summarization** — "summarize everything in the 'Authentication' subgraph" requires understanding multiple related pages and synthesizing them into a coherent overview. This is beyond extractive methods.

**Staleness context in notifications** — "Your page is becoming stale. It covers: JWT token issuance, refresh token rotation, and bcrypt password hashing. Has this changed?" — generating the "it covers X, Y, Z" part coherently from long pages benefits from LLM.

**Cross-page synthesis** — "What does the knowledge base say about our deployment process?" is a RAG-style question. The summary would be synthesized from multiple pages.

---

## Where simpler ML models could work

For single-page summarization, smaller fine-tuned models (e.g., BART, T5-small, Pegasus-XSum) can produce reasonable summaries at a fraction of LLM cost. Consider:

- **Hosted small models** (via HuggingFace Inference API or similar): cheaper than Claude, good enough for 80% of pages
- **LLM only for complex/long content**: if page word count < 500 words, use extractive (no ML). If 500-3000 words, use a small summarization model. If > 3000 words or multi-document, use LLM.
- **On-device for privacy**: for self-hosted deployments where users don't want content leaving their server, a locally-run quantized model (Ollama + llama3.2:3b) is viable. Go has gRPC bindings for Ollama.

---

## Recommendation

**Don't prioritize this for Phase 2.** Summarization is a "nice to have" that doesn't drive core adoption. Users choose a KMS for graph + freshness + search, not for auto-summaries.

**When you do build it (Phase 3), tier it:**
1. Excerpt/outline: extractive from ProseMirror AST (zero AI, ship first)
2. Single-page prose summary: small summarization model (cost-effective)
3. Cluster/synthesis summary: Claude claude-haiku-4-5-20251001 or Sonnet depending on content sensitivity

**Privacy consideration for open source:** self-hosters will not want their knowledge base content sent to an external AI API. Design the AI service as a pluggable backend:
- Default: extractive summaries (no API calls)
- Optional: configure an AI provider (Anthropic, OpenAI, or local Ollama endpoint)
- Make it explicit in the UI when AI is being used and where content goes

**Infrastructure note:** the architecture already shows an `ai_service.go` worker placeholder. When building it, expose it via an interface so the implementation (Anthropic API, Ollama, no-op) is swappable without changing the service layer.
