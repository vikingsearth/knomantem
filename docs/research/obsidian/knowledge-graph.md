# Obsidian: Knowledge Graph Capabilities

## Overview
Obsidian's graph view is one of its most famous features — a visual map of all notes as nodes and links as edges. Combined with backlinks, unlinked mentions, and community plugins, it provides one of the richest knowledge graph experiences among personal KM tools.

## Core Graph View
- Visual representation of all notes and connections
- Interactive: zoom, pan, click to navigate
- Color-coded by folder, tag, or custom groups
- Filter by tags, paths, or search queries
- Adjustable forces (repulsion, attraction, centering)
- Orphan note detection
- Global graph (entire vault) and local graph (current note's neighborhood)

## Backlinks System
- **Linked mentions**: All notes explicitly linking to current note
- **Unlinked mentions**: Text matching note's name that isn't linked yet — discovery system
- Advanced filtering using search operator syntax
- Block references (`[[Note^block-id]]`) for paragraph-level connections
- Link integrity maintained automatically — renaming updates all references

## Community Graph Plugins
- **InfraNodus**: AI-enhanced knowledge graph — reveals clusters, key concepts, gaps, generates research questions
- **3D Graph**: Interactive 3D force-directed graph with pan/zoom/rotate
- **Folders to Graph**: Integrates folder hierarchy into graph visualization
- **Export Graph View**: Export as .mmd (Mermaid) or .dot (GraphViz)

## Strengths
- One of the most visually impressive graph implementations in personal KM
- Backlinks + unlinked mentions create powerful discovery engine
- Community plugins extend capabilities significantly
- Graph built from bottom up — every link enriches it
- Encourages "networked thinking" and serendipitous discovery
- Free for personal use, local-first

## Limitations
- **Scalability**: Graph becomes "tangled web" at scale — "more fun to look at than navigate"
- **Basic built-in analysis**: Only rudimentary visualization natively; no cluster detection, centrality, gap analysis
- **Untyped links**: No way to distinguish "depends on" from "related to" from "contradicts"
- **No semantic understanding**: Links are structural, not semantic
- **No automatic connection discovery** (natively) — graph only shows explicit links
- **Single-user only**: No collaborative graph building
- Third-party plugins (InfraNodus, Hydrate) needed for real graph analysis

## Comparison
- **vs. Notion**: Far superior graph; Notion has no native graph view
- **vs. Confluence**: Confluence has no graph at all
- **vs. Roam/Logseq**: Comparable, but Obsidian has larger plugin ecosystem
- **vs. Dedicated graph databases**: Lightweight and note-centric, not a full graph DB
