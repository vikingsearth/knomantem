# Obsidian: Knowledge Staleness & Relevance Determination

## Overview
Obsidian has **no built-in content staleness or review management features**. Staleness management is entirely left to the user and community plugins, consistent with Obsidian's philosophy of user agency over opinionated workflows.

## What Obsidian Provides Natively
- File metadata: created/modified dates visible in file explorer
- Properties/frontmatter: manual review dates, status fields, freshness indicators
- Search by date: find notes modified before/after specific dates
- No review reminders, no content lifecycle management, no analytics

## Spaced Repetition Plugins (Primary Staleness Mechanism)
As of 2025, 26 curated spaced repetition plugins exist:

### Obsidian Spaced Repetition (st3v3nmw) — Most Popular
- Review flashcards AND entire notes using SM2 algorithm (Anki variant)
- Fights the "forgetting curve" for note content
- Cloze cards with multiple pattern support

### Obsidian Repeat Plugin (prncc)
- Review entire notes on periodic or spaced schedules
- Mark notes with `repeat` property in frontmatter
- Dedicated Repeat view shows what needs review
- Designed for surfacing notes, not just flashcards

### Spaced Repetition AI (SRAI)
- AI-powered flashcard generation with FSRS algorithm
- Auto-generates flashcards from content using OpenAI
- Combines AI + spaced repetition for maximum retention

### Obsidian Recall / Incremental Writing
- Modular SRS with pluggable algorithms
- Incremental writing: prioritized queues reviewed spaced-repetition style

## Practical User Approaches
- **Manual frontmatter**: `reviewed: 2025-01-15`, `review-interval: 90` — query with Dataview
- **Template-based tracking**: Templates include review date fields automatically
- **Tag-based status**: `#status/current`, `#status/needs-review`, `#status/archived`
- **Dataview dashboards**: Auto-updating views surfacing notes needing attention

## What's Missing
- No organizational staleness management or admin view
- No content owner assignment
- No automated review workflows or notifications
- No usage analytics (view counts, engagement metrics)
- No AI-powered freshness assessment
- No "Draft → Current → Review → Archived" workflow
- No team-level content health monitoring
- Everything is opt-in and manual

## The Philosophy
Provide the primitives (files, links, metadata) and let users build their own systems. Works well for individuals who enjoy designing workflows; creates a gap for teams wanting lifecycle management out of the box.
