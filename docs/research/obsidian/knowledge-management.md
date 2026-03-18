# Obsidian: Knowledge Management Capabilities

## Overview
Obsidian is a local-first, markdown-based knowledge management app that positions itself as a "second brain." All notes are plain-text Markdown files stored on the user's local filesystem, giving full data ownership and portability. Built around interlinked notes within "vaults" with extensive customization via a thriving plugin ecosystem. Free for personal use, 100+ million note-takers and knowledge workers in 2025.

## Vaults (Core Architecture)
- A vault is simply a folder containing notes, subfolders, media, and attachments
- Every note is a lightweight, human-readable Markdown file
- Open/edit in any text editor — no proprietary format, zero lock-in
- Unlimited vaults for different contexts (work, personal, research)
- Each vault has own settings, plugins, themes in hidden `.obsidian` directory
- Cross-platform: Windows, macOS, Linux, iOS, Android
- Local-first: no cloud dependency for core functionality
- "Just text files that could easily fit on a flash drive"

## Notes & Organization
- Each note is a single `.md` file (filename = note title)
- Hierarchical folder structures within vaults
- No imposed structure — folder-heavy, tag-heavy, link-heavy, or hybrid
- File explorer sidebar with drag-and-drop
- Bookmarks for frequently accessed notes
- Renaming auto-updates all internal links
- Canvas (v1.1+): infinite visual workspace for spatial arrangement

## Linking & Knowledge Connections
- **Wikilinks**: `[[Note Name]]` creates links between notes
- **Backlinks**: Auto-tracks all notes linking to current note
- **Unlinked mentions**: Surfaces text matching a note's name that aren't yet linked
- **Block references**: `[[Note^block-id]]` links to specific paragraphs
- **Embeds/Transclusion**: `![[Note]]` embeds content inline
- **Link to headings**: `[[Note#Heading]]` for deep linking
- Bidirectional linking creates a living web of knowledge

## Tags
- Inline tags: `#tag` syntax, nested tags: `#project/alpha`
- Tags in YAML frontmatter: `tags: [tag1, tag2]`
- Tag pane shows all tags with usage counts
- **AI Tagger Universe plugin**: Automates tag generation using local or cloud LLMs
- **Tag Group Manager plugin**: Structured tag organization with custom groups
- Best results come from hybrid approaches — "hard sciences lean on metadata accuracy, while humanities benefit from folder-tag flexibility"

## Properties / Frontmatter
- YAML frontmatter for structured metadata
- Properties view (v1.4+): Visual form instead of raw YAML
- Property types: text, number, date, datetime, checkbox, list, tags
- Aliases: multiple names for a single note
- Property search: `[property:value]` operator
- Queryable via Dataview plugin

## Templates
- Core Templates plugin: designated template folder, dynamic variables
- **Templater** (community): JavaScript execution, conditional logic, cursor placement
- **Daily Notes**: Auto-create dated notes with templates
- **Periodic Notes**: Weekly, monthly, quarterly, yearly notes

## Plugin Ecosystem (2025)
- ~25 core plugins + 2,000+ community plugins (growing: 21 new + 83 updates in one week)
- **Dataview**: SQL-like querying of notes/properties
- **Kanban**: Drag-and-drop task boards
- **Tasks**: Advanced task management with due dates
- **Omnisearch**: Enhanced fuzzy full-text search
- **Zotero Integration**: Academic citation management
- **Hydrate**: AI-driven chat with notes, semantic search
- **InfraNodus**: AI-enhanced knowledge graph visualization
- **Excalidraw**: Embedded whiteboard/drawing

## Sync & Collaboration
- **Obsidian Sync**: $8/month, end-to-end encrypted, version history
- **Third-party sync**: Dropbox, iCloud, Google Drive, Syncthing, Git
- **No real-time collaboration**: Deliberate single-user design
- **Obsidian Publish**: Paid service for publishing notes as website

## Pricing
- Free for personal use (no feature limitations, no paywalls)
- Sync: $8/month, Publish: $8/month
- Commercial license: $50/user/year
- "No onboarding funnel, no update pop-ups, no pressure to upgrade"

## Strengths
- True data ownership, plain Markdown, zero vendor lock-in
- Handles 50,000+ files with 40+ active plugins smoothly
- "Infinitely customizable" via plugins, themes, CSS
- Active, passionate community
- Cross-platform with offline access

## Limitations
- No native database/structured data (Dataview fills gap)
- No built-in task management beyond checkboxes (Tasks plugin)
- Single-user design — no team collaboration
- Steep learning curve for advanced features
- Plugin dependency for "essential" features
- Mobile less polished than desktop
