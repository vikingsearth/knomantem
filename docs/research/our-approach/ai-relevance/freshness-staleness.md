# AI Relevance: Freshness & Staleness Detection

## What the planned use case is

Phase 2 plans "AI staleness detection" — proactively identifying content that may be outdated based on signals beyond simple age. The current implementation (Phase 1) uses a deterministic linear decay formula: `score = 100 × (1 − decayRate × days / reviewIntervalDays)`. A background worker runs every 6 hours, applies decay, and fires a notification when score drops below 30.

The AI ambition is to detect semantic staleness: a page that was edited last week but references a deprecated API, a policy document that conflicts with a newer regulation, or an architecture doc that describes a system that no longer exists.

---

## Does it need AI?

**Split answer: the decay engine does not need AI. Semantic signal enrichment does.**

### What does NOT need AI

The core decay formula is a script. It always was. The current implementation proves this — it works today without any ML. Time-based decay is:

```
score = 100 × (1 − rate × days / interval)
```

Several additional signals can be computed as pure scripts or lightweight jobs:

- **Edit frequency decay adjustment**: pages edited often → slower decay rate. This is a configurable multiplier, not ML.
- **View count signals**: a page viewed 200 times in the last month is probably still relevant. Pure DB query.
- **External URL health**: fetch URLs embedded in page content, return 404 → freshness penalty. This is an HTTP health check script, not AI.
- **Dependency propagation**: if page A depends_on page B and B becomes stale, A's score should decay faster. This is a graph traversal query (recursive CTE in PostgreSQL), not ML.
- **Last verified age**: a page verified 6 months ago decays faster than one verified yesterday. Already implemented.

These improvements would dramatically increase staleness signal quality with zero AI infrastructure.

### What genuinely needs AI

Semantic staleness detection is where simpler approaches fail:

1. **Content-vs-reality divergence**: a page says "we use Redis for caching" but the codebase now uses Memcached. Detecting this requires cross-referencing page content against external systems (GitHub, Jira, etc.) — feasible with lightweight integrations, but interpreting the semantic match requires embeddings or LLM reasoning.
2. **Self-contradiction across pages**: page A says the deployment process takes 5 steps; page B (written later) says 3 steps. Detecting this contradiction requires semantic comparison, not string matching.
3. **Topic drift**: a page's title says "Onboarding Guide" but its content has evolved to cover mostly IT provisioning. Detecting this mismatch requires embeddings.
4. **Reference obsolescence without 404**: a page links to an internal page that still exists but has completely changed its meaning. URL health checks won't catch this.

---

## Recommendation

**Phase 1 (now):** Extend the deterministic decay formula with script-based signals: edit frequency weight, view count weight, external URL health check, dependency propagation via graph CTE. These are high-value, zero-ML improvements that can ship in days.

**Phase 2 (after content mass):** Add an embedding-based similarity layer. Store page embeddings in PostgreSQL (pgvector extension, already possible with pg 16). Use embeddings to:
- Detect when a page's content has drifted significantly from its last-verified snapshot (cosine distance threshold)
- Identify pages with similar content to a recently-stale page (candidates for review)

**Phase 3:** LLM-based semantic staleness reasoning for high-stakes content (policies, security docs). Use Claude claude-haiku-4-5-20251001 for cost efficiency. Only run on pages with high view counts or explicit "critical" flag to control API costs.

**What to avoid:** Don't run LLM on every page on every decay cycle. That's expensive and unnecessary. The formula + scripts handle 90% of cases; reserve LLM for the 10% that require real reasoning.
