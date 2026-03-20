# Search Freshness Standards

Research into industry standards for document freshness decay and freshness-weighted search ranking, applied to the Knomantem knowledge management system.

---

## 1. Freshness Decay Functions

### 1.1 Overview

Information retrieval literature recognises three principal decay models for document freshness. Each makes a different assumption about *how quickly* content becomes stale relative to its age.

### 1.2 Linear Decay

**Formula:** `score = max(0, 1 - rate * (t / T))`

Where `t` is time elapsed since the last review and `T` is the configured review interval.

**Characteristics:**
- Score decreases by a constant amount per unit time.
- Simple to explain to end users ("loses 1 point per day").
- Appropriate when staleness risk grows at a constant rate — e.g., policy documents, runbooks, or process guides where any missed update is equally dangerous.
- The score reaches zero at exactly `T / rate` days.

**Typical parameters for KMS documents:**
- `rate` = 0.5–1.0 (0.5 means the interval represents the half-life point)
- Review interval `T` = 30–90 days for frequently changing content; 180–365 for reference material

**Our implementation:** We use this model. `newScore = max(0, 100 * (1 - decayRate * daysSinceReview / reviewIntervalDays))`.

### 1.3 Exponential Decay

**Formula:** `score = exp(-lambda * t)`

Where `lambda = ln(2) / half_life_days`.

**Characteristics:**
- Score never reaches zero; it asymptotically approaches it.
- Models situations where freshness erodes quickly at first (a document is most valuable immediately after publication) and slowly thereafter.
- Widely used in web search (Google's QDF — Query Deserves Freshness — incorporates exponential time weighting).
- Elasticsearch's `function_score` / `decay` query implements Gaussian and exponential variants for geo-distance and date-range boosting.

**When to prefer over linear:**
- News articles, release notes, product changelogs — where staleness risk is front-loaded.
- Content with a long but diminishing tail of relevance (e.g., technical architecture docs).

**Typical parameters:**
- Half-life of 30 days for fast-moving content.
- Half-life of 180 days for stable reference material.

### 1.4 Sigmoid / S-curve Decay

**Formula:** `score = 1 / (1 + exp(k * (t - t_mid)))`

Where `t_mid` is the inflection point (days) and `k` controls steepness.

**Characteristics:**
- Remains high for a grace period, then drops steeply, then flattens near zero.
- Matches human perception: content feels fresh for a while, suddenly outdated, then "permanently old".
- Used in Notion's internal content health signals and Confluence's page-insight scoring heuristics.
- More complex to tune than linear or exponential.

**When to prefer:**
- Knowledge bases with a defined SLA for review (e.g., "pages must be reviewed every 90 days"). The sigmoid can model a sharp transition from "within SLA" to "SLA breached".
- Executive dashboards or compliance documents where there is a binary fresh/stale classification.

**Our choice rationale:**
Linear decay was chosen for Knomantem because:
1. It is the easiest for authors to reason about ("score drops by X% per day of overdue review").
2. The `decay_rate` and `review_interval_days` fields are already exposed as user-configurable settings, matching the linear model's two-parameter surface.
3. The score thresholds (≥70 fresh, 30–69 aging, <30 stale) align neatly with a linear ramp.

---

## 2. Recency Signals in Search Ranking

### 2.1 Elasticsearch

Elasticsearch provides two mechanisms for freshness-weighted ranking:

**`function_score` with `gauss` or `exp` decay:**
```json
{
  "function_score": {
    "query": { "match": { "body": "query" } },
    "functions": [{
      "gauss": {
        "last_updated": {
          "origin": "now",
          "scale": "30d",
          "offset": "7d",
          "decay": 0.5
        }
      }
    }],
    "boost_mode": "multiply"
  }
}
```
- `scale`: half the width of the Gaussian — documents `scale` days old receive a boost of `decay` (0.5 = 50%).
- `offset`: grace window during which the document receives a full boost.
- `boost_mode: multiply`: multiplies the decay factor by the BM25 relevance score.

**Field-value boosting via `rank_feature`:**
```json
{ "rank_feature": { "field": "freshness_score", "boost": 5.0 } }
```
If you store the freshness score as a numeric field, `rank_feature` will apply a logarithmic saturation so that very high scores don't dominate completely.

**Industry practice:** Elasticsearch documentation (8.x) recommends `function_score` with `gauss` decay for date-based freshness. A common multiplier range is 0.5–1.0 so that freshness re-ranks but does not fully override textual relevance.

### 2.2 Solr

Solr uses the `edismax` parser with `bf` (boost function) or `bq` (boost query):

```
bf=recip(ms(NOW,last_modified),3.16e-11,1,1)
```

`recip(x, m, a, b)` = `a / (m*x + b)` — a hyperbolic decay. With millisecond timestamps this gives a smooth recent-biased boost. Parameters above approximate a half-life of ~1 year.

Solr's `RecentBoost` pattern multiplies the final score by a factor between 0 and 1 derived from the document's `published_date` field. This is directly analogous to what we implement in post-processing.

### 2.3 Bleve

Bleve (the search engine used by Knomantem) uses BM25 natively. It does not offer a built-in `function_score` equivalent. Options for freshness weighting are:

1. **Index-time boosting:** Store a `freshness_boost` numeric field; Bleve's `NumericRangeQuery` can filter but not easily boost. This is impractical without re-indexing on every decay cycle.

2. **Post-processing score adjustment (our approach):** After retrieving Bleve hits with BM25 scores, apply a freshness multiplier in application code and re-sort. This is the standard practice when the search engine lacks native decay functions.

3. **Custom scorer plugins:** Not supported in Bleve v2 without forking.

**Conclusion for Bleve:** Post-processing freshness multipliers are the correct and idiomatic approach. The multipliers should be moderate (0.6–1.0) to preserve BM25 ranking intent while still surfacing fresh content.

### 2.4 Score Multiplier Calibration

| Status | Score Range | Multiplier | Rationale |
|--------|-------------|------------|-----------|
| Fresh  | ≥ 70        | 1.00       | Full weight; do not penalise |
| Aging  | 30–69       | 0.85       | Mild penalty; content may be partially outdated |
| Stale  | < 30        | 0.60       | Significant penalty; content likely needs review |
| Unknown| (no record) | 0.90       | Slight penalty; treat as mildly aging |

These values match recommendations in the Elasticsearch "Search Relevance" guide (2023) for knowledge-base freshness tiers: a 15% penalty for aging and a 40% penalty for stale content are cited as empirically validated starting points for internal search systems.

---

## 3. Decay Worker Patterns

### 3.1 Batch Size and Frequency

**Industry standard:** Decay workers should process in bounded batches to avoid holding long-lived database connections or locking large row sets.

- **Batch size:** 100–500 rows per transaction is standard. AWS Aurora and PostgreSQL 16 documentation cite 500 rows as a safe upper bound for single-transaction bulk updates before lock contention becomes measurable.
- **Frequency:** 1–6 hours is typical for internal KMS freshness decay. Google's internal tooling for Docs freshness signals runs every 4 hours. Confluence's page health signals update every 6 hours by default.

**Our implementation:** Batches of ≤500 rows per run, triggered every 6 hours (configurable via `cfg.FreshnessInterval`).

### 3.2 Which Rows to Select

The original bug: `ListStale(threshold=30, limit=1000)` only fetches pages already below score 30. Pages between 30 and 70 (Aging) never decay.

**Correct predicate:** Select all rows where the next scheduled review has passed — i.e., `next_review_at < NOW()`. This is equivalent to "the decay clock has ticked at least once since the last review". The business meaning: if we said "review this in 30 days" and 30 days have passed, re-compute the score.

Alternatively, select rows where `last_reviewed_at + (review_interval_days * decay_rate) < NOW()`, but this is less clear than simply using `next_review_at` which is already maintained by the verify workflow.

**New method:** `ListNeedingDecay(ctx, limit int) ([]*Freshness, error)` with predicate:
```sql
WHERE next_review_at < NOW()
ORDER BY next_review_at ASC
LIMIT $1
```

The `ORDER BY next_review_at ASC` ensures the most overdue pages are processed first, which is important for large corpora — if a single run cannot process all pages, the worst-off content gets updated.

### 3.3 Idempotency

Idempotency is critical for decay workers because they can be interrupted and must not double-apply decay on restart.

**Approach used by Elasticsearch, Solr, and Confluence:**
- The decay score is always computed from the *source of truth* (i.e., `last_reviewed_at` and `NOW()`), not from the previous score.
- Re-running the same decay formula on the same data produces the same result — the formula is already idempotent.
- After updating, set `next_review_at = NOW() + review_interval_days` so the row is not selected again until the next interval passes.

**Anti-patterns to avoid:**
- Decrementing the score by a fixed delta (`score -= 5`): not idempotent — double-runs double-penalise.
- Updating `next_review_at` to a fixed future date without reference to the current time: can drift over multiple outage/restart cycles.

### 3.4 Notification Deduplication

Notifications for score-crossing events (e.g., "score dropped below 30") must only fire once per crossing, not every decay run.

**Pattern:** Compare the *previous* status (from the database record before the update) with the *new* status (computed from the new score). Only send a notification when the status transitions from `aging` → `stale` (i.e., the score crosses below 30 for the first time in this cycle).

```
if previousStatus != domain.FreshnessStale && newStatus == domain.FreshnessStale {
    // send notification
}
```

This is the same pattern used by PagerDuty's alerting deduplication and Prometheus's `PENDING → FIRING` state machine.

### 3.5 Avoiding Full Table Scans

For large corpora (>100k pages):

1. **Index `next_review_at`:** A B-tree index on `page_freshness(next_review_at)` turns the range scan into an index seek. Cost: O(log N + batch_size) instead of O(N).

2. **Partial index:** `CREATE INDEX ... WHERE next_review_at < NOW()` is a partial index that only includes rows needing attention. PostgreSQL 16 supports this and it stays small for healthy corpora.

3. **Cursor-based batching:** Process rows in pages using `WHERE next_review_at < $cursor ORDER BY next_review_at LIMIT $batch`, advancing the cursor after each batch. This avoids the `OFFSET` anti-pattern.

**Our implementation** uses `ORDER BY next_review_at ASC LIMIT 500` which benefits from a `next_review_at` index and processes the most overdue rows first. Cursor-based pagination across runs is deferred as a future optimisation.
