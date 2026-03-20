# AI Relevance: Auto-Tagging

## What the planned use case is

Phase 2 includes "AI-powered auto-tagging" — reducing manual overhead on tag management and improving search and graph quality. Currently tags are 100% manual: the `tag_service.go` is 43 lines, simply stores user-provided tags. There is no suggestion mechanism. The API supports adding tags with a `confidence_score` field (in both the backend domain and the Flutter `addTagsToPage` API call), which shows intent for machine-generated tags was designed into the schema from day one.

---

## Does it need AI?

**Partial: a useful v1 can ship without AI. A genuinely good v2 needs embeddings.**

### What does NOT need AI

**Keyword matching against existing tags** is sufficient for a meaningful first pass. If a user has created tags like "authentication", "PostgreSQL", "deployment", "CI/CD", then:

1. Extract significant n-grams from page title + headings + body text (TF-IDF or simple frequency)
2. Match against the existing tag name corpus (exact match, then fuzzy match via Levenshtein distance)
3. Rank by term frequency × tag popularity (pages already tagged with this → higher confidence)
4. Return top 3-5 suggestions with confidence scores

This approach is deterministic, fast, runs in-process in Go (no external dependency), and works well when: (a) the tag vocabulary is established, (b) pages use direct keyword language.

**Rule-based heuristics** cover other cases:
- Space-level defaults: every page in "Engineering" space auto-gets tag "engineering" unless opted out
- Template inheritance: pages created from a "Meeting Notes" template auto-tag with "meeting-notes"
- Regex patterns: detect code blocks → tag "code-sample"; detect `##` headers with "RFC" → tag "rfc"

### What genuinely needs AI

The keyword approach breaks down when:

1. **Vocabulary mismatch**: a page about "Kubernetes pod eviction policies" should receive a "resource-management" tag that was never mentioned in the content. Embeddings bridge the semantic gap between content language and tag vocabulary.
2. **Implicit topic**: a lengthy page about designing a checkout flow never says "payments" explicitly but it's deeply about payments. Embedding similarity against tag descriptions solves this.
3. **New tag suggestions**: not just matching existing tags, but suggesting that a new tag should be created based on a cluster of pages with similar content. This is a clustering problem (k-means over embeddings), not keyword matching.
4. **Cross-language content**: non-English pages with English tags.

---

## Recommendation

**Ship keyword matching first.** It's a 2-3 day implementation, zero external dependency, and handles the common case where users write "we updated the PostgreSQL schema" and the system suggests the "postgresql" and "database" tags.

**Add embeddings in Phase 2** using pgvector in PostgreSQL 16 (already in the stack). Generate page embeddings on creation/update via a background worker. Compare page embedding against tag embeddings (generate embeddings for tag name + description). Return cosine similarity ranked suggestions.

**Skip LLMs for tagging entirely** — embeddings are sufficient, faster, cheaper, and more predictable. An LLM would add latency and cost for marginal gain over good embeddings.

**Confidence score usage**: the schema already supports it. Keyword matches → 0.6-0.8 confidence. Embedding matches → 0.4-0.9 confidence based on cosine score. Manual user tags → 1.0. Surface suggestions below 0.6 threshold separately as "low confidence" to avoid noise.
