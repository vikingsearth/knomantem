# Obsidian: Knowledge Search Capabilities

## Overview
Obsidian provides powerful native search supplemented by community plugins like Omnisearch. Considered "one of the best search tools in any note-taking app" — flexible, powerful, fast, and easy to use.

## Native Search
- Global search (Ctrl/Cmd+Shift+F), Quick Switcher (Ctrl/Cmd+O)
- Real-time results as you type with context around matches
- **Operators**: `file:`, `path:`, `tag:`, `line:()`, `block:()`, `section:()`, `task:`, `task-todo:`, `task-done:`, `match-case:`, `ignore-case:`
- AND/OR/NOT logic for combining operators
- **Regex support**: Full regular expressions (limitation: regex doesn't work with `tag:` and `section:` operators)
- **Embedded search**: Results embedded directly in notes for dynamic dashboards
- **Property search**: `[property:value]` for frontmatter querying

## Omnisearch Plugin (2023 Gems of the Year Winner)
- "Quick Switcher on steroids" — relevance-ranked search using BM25 algorithm
- Indexes notes, Office docs, PDFs, images (via Text Extractor)
- **Fuzzy matching**: Handles typos ("algorthm" → "algorithm")
- Automatic scoring: titles > headings > body text
- Browser extension (Firefox, Chrome) for vault search from browser
- **Limitations**: No regex support, full re-index on vault open, can be slower on large vaults

## Additional Search Plugins
- **Dataview**: SQL-like querying of notes/properties (not text search but structured data)
- **Hydrate**: AI-powered semantic search beyond keywords
- **Regex Find and Replace**: Dedicated regex search/replace with visual interface
- **Smart Random Note**: Surface random notes for serendipitous rediscovery

## Strengths
- Native search is fast with rich operator syntax
- Regex for advanced pattern matching
- Omnisearch adds fuzzy, relevance-ranked search
- Everything runs locally — no network latency
- Plugin ecosystem continuously improving

## Limitations
- Regex incompatible with `tag:` and `section:` operators
- No built-in fuzzy search (Omnisearch fills this)
- No semantic/AI search natively (Hydrate fills this)
- "Decent for content filtering but not for content discovery" — common native search critique
- No cross-vault search
- No team/shared search
- No search analytics
