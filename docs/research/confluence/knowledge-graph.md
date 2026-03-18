# Confluence: Knowledge Graph Capabilities

## Overview
Confluence does **not** have a native, dedicated knowledge graph feature. This is a highly requested capability in the Atlassian Community. However, Confluence has several features that approximate graph-like knowledge connectivity, and Atlassian's broader platform (especially the Teamwork Graph and Rovo AI) provides some graph-adjacent capabilities.

## What Confluence Has Today

### Page Linking
- Wiki-style internal links between pages (`[Page Title]` syntax)
- Links are bidirectional in practice — you can see "incoming links" on pages via macros
- Smart Links provide rich previews when linking to external URLs (Jira, Google Docs, Figma, etc.)
- Anchor links within pages for deep linking to specific sections
- Include Page macro for content transclusion (embedding one page's content in another)

### Labels as Lightweight Taxonomy
- Labels create informal groupings across pages and spaces
- Content By Label macro surfaces all pages sharing a label
- Nested label structures can simulate simple categorization hierarchies
- No formal ontology or relationship typing between labels

### Space Cross-Referencing
- Pages can link across spaces freely
- Global search finds content regardless of space boundaries
- Content By Label macro can pull from all spaces or specific ones

### Atlassian Intelligence (AI Features)
- Powered by the **Teamwork Graph** — connects service work, project data, third-party apps, and team information
- AI search understands context and provides answers (not just document lists)
- Auto-definitions for company-specific jargon and acronyms
- Content summarization across linked pages
- Q&A search (beta) answers questions using workspace content as grounding

### Rovo AI (Cross-Product)
- Rovo spans Confluence, Jira, Loom, and connected third-party tools
- Unlocks knowledge across all apps, tools, and data sources
- Provides relevant, accurate answers by understanding relationships between content across products
- Agents can be configured to perform domain-specific knowledge retrieval

## What's Missing (Community Requests)

### Formal Knowledge Graph
- Community members have explicitly requested knowledge graph capabilities
- Users want AI-assisted knowledge graphs with human curation
- Desired features include:
  - Entity extraction from pages, PDFs, and diagrams
  - Relationship mapping (temporal, spatial, causal, object-relational)
  - Visual graph exploration of how knowledge connects
  - AI-suggested relationships for human review and approval
  - Cross-content-type graph (pages + attachments + Jira issues + diagrams)

### Specific Community Feedback
- "Knowledge Graphs - AI Assisted with Human Curation are a NECESSITY" — titled post on Atlassian Community forums
- Users report lots of unstructured information across spaces and want ways to discover hidden connections
- Requests for graph visualization tools that show how pages relate to each other
- Interest in using Confluence content as source material for external knowledge graph tools

## Comparison to True Knowledge Graph Tools
- **No entity extraction**: Confluence doesn't automatically identify and tag entities (people, products, concepts) within page content
- **No relationship typing**: Links between pages are untyped — there's no way to say "page A depends on page B" or "page A supersedes page C"
- **No graph visualization**: No built-in view showing the network of connections between pages
- **No semantic layer**: The Teamwork Graph is more about work context (who worked on what, when) than semantic knowledge relationships
- **No inference**: Cannot derive implicit relationships from explicit ones

## Third-Party Options
- Some Marketplace apps offer basic relationship mapping
- External tools (Neo4j, etc.) can ingest Confluence content via API for graph analysis
- Rovo's AI provides some semantic understanding but not a formal graph structure
- The gap between what users want and what Confluence provides represents a significant opportunity

## The Teamwork Graph (Atlassian Platform)
- Connects work artifacts across Jira, Confluence, Bitbucket, and third-party tools
- Maps people to projects, teams, and work items
- Powers AI features with contextual understanding of organizational work
- More of an activity/work graph than a knowledge/concept graph
- Enables features like "who knows about X" and "what's related to this project"
