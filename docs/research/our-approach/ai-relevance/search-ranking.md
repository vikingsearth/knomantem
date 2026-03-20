# AI Relevance: Search Ranking

## What the planned use case is

The roadmap describes "freshness-weighted search ranking" and "powerful search" as a core MVP differentiator. The architecture also hints at future semantic/vector search. Currently, `search_service.go` is 27 lines — it calls `SearchRepository.Search()` and returns results. The `SearchItem` domain type includes a `Freshness FreshnessBrief` field (score + status), but the ranking does not actually incorporate freshness. Bleve provides BM25-style full-text ranking.

---

## Does it need AI?

**Freshness weighting: no AI needed. Semantic search: needs embeddings, not LLM.**

### Freshness-weighted ranking — pure formula, no AI

This is the single most impactful near-term improvement and requires zero ML.

**Current state:** BM25 score from Bleve, no freshness consideration. A stale page from 3 years ago ranks equally to a fresh page from last week if keyword match is similar.

**Solution:** Post-process Bleve results with a composite score:
```
finalScore = bleveScore × freshnessMultiplier(freshnessScore)

freshnessMultiplier(s):
  s >= 70 (Fresh):  1.0    (no penalty)
  s >= 30 (Aging):  0.85
  s <  30 (Stale):  0.60
```

This is 10 lines of Go in `search_service.go`. The freshness score is already returned by the search repository and exists on the `SearchItem`. This is the lowest-effort, highest-impact change in the entire codebase.

**Additional non-AI ranking signals:**
- **View recency**: pages viewed recently by many users → boost (relevance signal)
- **Edit recency**: recently edited → small boost (author thought it was worth updating)
- **Graph centrality**: pages with many inbound edges → boost (the knowledge base considers this important)
- All computable from existing DB data, no ML

### Search quality improvements — still no AI

Bleve currently uses default configuration. Several improvements don't require ML:

1. **Title field boosting**: already mentioned in architecture diagram (`title > body > tags`), needs verification it's implemented in the Bleve index configuration
2. **Synonym expansion**: define a synonym map (JSON file) for common domain terms. Bleve supports this natively.
3. **Stemming**: ensure the Bleve analyzer uses an English stemmer (Porter2). This improves recall for plurals, verb forms.
4. **Stop word tuning**: domain-specific stop words ("the", "our", "company") for cleaner indexing

### Semantic / vector search — needs embeddings (not LLM)

BM25 fails when users query by concept rather than keyword. "How do we handle auth?" won't find a page titled "JWT Implementation Guide" if the content doesn't repeat the word "auth" enough.

**Vector search approach:**
- Store page embeddings in pgvector alongside Bleve index
- On search query, also generate a query embedding and run ANN similarity search
- Merge BM25 results (keyword relevance) with ANN results (semantic relevance) using Reciprocal Rank Fusion (RRF)
- RRF is a well-understood formula: `1 / (k + rank_bm25) + 1 / (k + rank_semantic)`

**Why not replace Bleve with vector-only search:**
- Vector search has poor precision for exact/known-item queries ("find the page called 'Deployment Checklist'")
- BM25 handles exact matches, quotes, boolean operators better
- Hybrid search gives the best of both worlds
- Bleve remains the right MVP choice; vector becomes an additive layer in Phase 2

### What does NOT need LLM

- Query rewriting / intent detection: overkill for MVP. Users of KMS tools know how to search.
- Query expansion: synonym files handle the common cases (5-10 domain synonyms) without LLM inference on every query.
- Answer generation (RAG): this is a Phase 3+ feature and a fundamentally different product surface. Don't conflate search ranking improvements with chat-over-docs.

---

## Recommendation

**Do now (zero AI, days of work):**
1. Implement freshness-weighted post-processing in `search_service.go`
2. Verify Bleve index uses title field boosting
3. Add English stemmer and basic synonym map to Bleve analyzer config

**Phase 2 (embeddings, no LLM):**
4. Add pgvector, generate embeddings on page save
5. Implement hybrid search (BM25 + ANN merged via RRF)

**Phase 3 (optional LLM):**
6. Evaluate query understanding for large corpora (>50k pages) where intent matters more
7. Consider RAG / "ask the knowledge base" as a distinct product surface, not a search improvement

**Don't build:** LLM-powered search ranking. It's expensive, latency-sensitive (user is waiting), and not meaningfully better than hybrid BM25+vector for a KMS use case.
