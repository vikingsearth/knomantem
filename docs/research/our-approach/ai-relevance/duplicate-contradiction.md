# AI Relevance: Duplicate & Contradiction Detection

## What the planned use case is

Phase 2/3 roadmap includes "AI duplicate/contradiction detection" — identifying pages that cover the same content (duplicates) and pages that make conflicting claims (contradictions). This is listed as a key differentiated AI feature: no competitor does this automatically at the platform level.

---

## Does it need AI?

**Yes, but different levels for duplicates vs. contradictions.**

### Duplicate detection — embeddings are sufficient (no LLM)

Finding semantically similar pages is a well-solved problem with dense embeddings + cosine similarity. It does NOT require an LLM.

**Why embeddings suffice:**
- "Deployment Guide" and "How to Deploy to Production" will have high cosine similarity even with different wording
- Structural similarity (both have H1 headings, numbered lists, code blocks) can be computed from the ProseMirror AST without any ML
- Title similarity (Levenshtein distance + TF-IDF overlap) catches obvious duplicates fast and cheap

**Implementation approach:**
1. On page create/update, generate embedding (e.g., via a locally-hosted small model like `all-MiniLM-L6-v2` via a sidecar, or via Claude's embedding API)
2. Store in pgvector column on the `pages` table
3. Background worker: for each new/updated page, query `ORDER BY embedding <=> $1 LIMIT 10` (ANN search)
4. Pages with cosine similarity > 0.85: flag as "potential duplicate"
5. Surface in the freshness dashboard as a "Maintenance" alert type

**What simple approaches miss:**
- Two pages that duplicate conceptually but use completely different terminology (e.g., "IAM Policies" vs "Access Control Configuration" covering the same AWS topic)
- Embeddings still catch these if the model is good; it's a model quality question, not an architecture question

**Threshold tuning matters:** 0.95+ = almost certainly duplicate. 0.80-0.95 = strong candidate, require user review. Below 0.80 = noise, ignore.

---

### Contradiction detection — LLM is required

Detecting contradictions is fundamentally different from detecting similarity. Two pages can be highly similar in embedding space (both about authentication) but make *opposite* claims ("use bcrypt" vs "use argon2 for new services"). Embedding similarity finds pages that are *about the same thing* — contradiction detection requires understanding what each page *claims*.

**Why simpler approaches fail:**
- String matching: looking for negation patterns ("don't", "never", "incorrect") is brittle and has terrible precision
- Embedding distance: low similarity ≠ contradiction; high similarity ≠ no contradiction
- Structured NLI (natural language inference) models can classify "entailment / neutral / contradiction" for sentence pairs, but require extracting the key claims from long documents first — which itself requires language understanding

**Genuine LLM use cases:**
1. **Claim extraction**: given a page, extract its key factual claims (e.g., "The deployment takes 3 steps", "The API uses OAuth 2.0", "Redis is required")
2. **Claim comparison**: given two claim sets from pages covering similar topics (identified by embeddings first), use LLM to evaluate if any claims contradict
3. **Confidence scoring**: output a structured `{contradiction: bool, claim_a: str, claim_b: str, confidence: float}` JSON

**Practical approach:**
- Only run contradiction detection on page pairs with embedding similarity > 0.70 (same topic area)
- Only process pages tagged as "factual" content types (architecture docs, policies, guides) — not meeting notes, personal logs
- Run as async background job, not in request path
- Use Claude claude-haiku-4-5-20251001 for cost; Sonnet for high-stakes content (security, legal policies)

---

## Recommendation

**Implement duplicate detection in Phase 2 with embeddings only.** This is achievable, cost-effective, and provides real value. Use pgvector, set conservative thresholds, surface in the freshness dashboard as "potential duplicates needing review."

**Defer contradiction detection to Phase 3.** It requires:
1. A working embedding pipeline (Phase 2 prerequisite)
2. LLM API integration (new infrastructure)
3. Claim extraction prompt engineering and validation
4. UX for presenting contradictions in a way users trust and act on

Don't ship contradiction detection without a high-quality UX for it. A false positive ("these two pages contradict each other!") that doesn't actually contradict will destroy user trust in the feature faster than not having it at all.

**Infrastructure shared with auto-tagging and graph inference:** The embedding pipeline (generate → store in pgvector → query by ANN) is the foundation for all three Phase 2 AI features. Build it once, use it everywhere.
