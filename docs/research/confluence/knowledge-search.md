# Confluence: Knowledge Search Capabilities

## Overview
Confluence provides multiple search mechanisms ranging from basic keyword search to the powerful Confluence Query Language (CQL), supplemented by AI-powered search via Atlassian Intelligence and Rovo.

## Basic Search
- Global search bar available from any page
- Searches across all accessible spaces, pages, blog posts, attachments, and comments
- Type-ahead suggestions as you type
- Results ranked by relevance (combination of text match, recency, and popularity)
- Filter results by space, contributor, type (page/blog/attachment), and date range
- Recent searches saved for quick re-access

## Confluence Query Language (CQL)
CQL provides SQL-like syntax for precise content querying:

### Query Capabilities
- **Content type filtering**: Search for pages, attachments, comments, or labels specifically
- **Logical operators**: AND, OR, NOT for complex queries
- **Wildcards**: `*` and `?` for partial matching (not allowed as first character)
- **Fuzzy matching**: `~` operator for approximate string matching
- **Case insensitive**: All queries are case insensitive by default
- **Field-specific search**: Query by title, text, label, space, creator, created date, modified date, contributor, type, and more
- **Custom search fields**: Extend CQL with custom fields for instance-specific tracking
- **Sorting**: Results can be ordered ascending/descending by supported fields
- **Date ranges**: Search by creation/modification date ranges

### CQL Operators
- `=`, `!=` for exact match/exclusion
- `~` for contains (text search)
- `>`, `<`, `>=`, `<=` for date and number comparisons
- `IN`, `NOT IN` for set membership
- Combinable with AND/OR/NOT logic

### CQL Integration Points
- REST API search endpoint
- Content By Label macro (uses CQL under the hood)
- Automation rules can use CQL to find content
- Third-party apps can leverage CQL for custom search experiences

## AI-Powered Search (Atlassian Intelligence / Rovo)

### Q&A Search (Beta)
- Natural language questions get direct answers, not just document lists
- AI reads and synthesizes information from across the workspace
- Grounded in actual Confluence content to reduce hallucination
- Available to users with Atlassian Intelligence enabled

### Smart Suggestions
- Auto-suggestions while typing queries
- AI understands context and intent beyond keyword matching
- Definition tooltips for company-specific terminology

### Rovo AI Search
- Cross-product search spanning Confluence, Jira, Loom, and connected tools
- Semantic understanding of queries
- Provides contextual answers drawing from multiple data sources
- Agent-based search for domain-specific retrieval

## Search Within Spaces
- Scope search to a specific space using the space filter
- Livesearch macro embeds space-scoped search directly on pages
- Space sidebar includes search functionality
- CQL supports `space = "SPACEKEY"` for programmatic space-scoped queries

## Content By Label Macro
- Dynamic content listing based on label queries
- Supports AND/OR/NOT label combinations
- Can pull from all spaces or specific ones
- Configurable display (list, table, card)
- Serves as a curated search result for specific topics

## Known Limitations

### Performance & Result Limits
- REST API CQL search returns max 50 results when "body" expansion is requested
- Without expansions: limited to 1000 results
- With non-body expansions: limited to 200 results
- `body.export_view` or `body.styled_view` expansions: limited to 25 results
- The `limit` parameter cannot exceed 200
- Total result count unavailable in API — must paginate through results

### Wildcard Restrictions
- Cannot use `*` or `?` as the first character of a search term
- Limits some pattern-matching use cases

### Ordering Limitations
- Not all fields support ordering
- Fields with multiple values per content (e.g., labels) cannot be used for sorting

### Autocomplete Limits
- CQL autocomplete results capped at 20 suggestions
- May need to type more characters to narrow results

### Deprecated Fields
- User-specific CQL fields (`user`, `user.fullname`, `user.accountid`) no longer supported on the main search endpoint
- Must use dedicated user search endpoint for people queries

### General Search Quality Issues (User Feedback)
- "Search can be hit-or-miss" — common complaint in user reviews
- Finding content often requires browsing hierarchies rather than relying on search
- Relevance ranking doesn't always surface the most useful results
- Search struggles with large instances (thousands of pages across many spaces)
- Attachment content search quality varies by file type
- No faceted search without third-party apps
