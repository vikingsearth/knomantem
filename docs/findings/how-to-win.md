# How to Win: Knomantem Strategy and Positioning

## Executive Summary

Knomantem is a knowledge management platform that combines **graph visualization**, **relational databases**, **AI-powered maintenance**, **content freshness tracking**, **powerful search**, and **team collaboration** into a single cohesive product. Built with a **Go backend** and **Flutter frontend**, licensed under **BSL 1.1**, Knomantem targets teams that have outgrown wikis but don't want the overhead of enterprise platforms.

**The core thesis**: Every major knowledge management tool on the market is missing at least two of these capabilities. Knomantem wins by delivering all of them together, with a self-hosted free tier that removes adoption friction entirely.

---

## Competitive Landscape and Attack Vectors

### vs Confluence

**Key weakness**: Confluence is where knowledge goes to die. Pages are created, never updated, and impossible to find six months later.

| Attack Vector | Detail |
|---|---|
| **Content staleness** | Confluence has **zero freshness tracking**. There is no mechanism to surface stale content, flag outdated pages, or prompt reviews. This is the single biggest pain point for Confluence teams at scale. |
| **Search quality** | CQL (Confluence Query Language) is complex and unintuitive. Search results are hit-or-miss, with no semantic understanding. Users resort to Slack to ask "does anyone know where the doc for X is?" |
| **Performance at scale** | Workspaces with 500+ pages routinely see 5-10 second page loads. This compounds over time and drives adoption downward. |
| **Knowledge graph** | No native graph capability. This is the **#1 community-requested feature** on the Atlassian community forums, and Atlassian has shown no indication of building it. |
| **Price** | Confluence is cheap at $5-10/user/month, but Knomantem is **free** for self-hosted. For a 50-person team, that is $3,000-6,000/year saved. |
| **Mobile experience** | Confluence's mobile app is widely criticized as barely functional. |

**Positioning**: *"Your knowledge, alive and connected -- not buried in a wiki graveyard."*

**Actionable takeaway**: Build a one-click Confluence import tool early. Every Confluence workspace is a potential migration target. Freshness scoring applied to imported content will immediately demonstrate value by surfacing years of rot.

---

### vs SharePoint

**Key weakness**: SharePoint is an enterprise platform that requires enterprise resources to operate. Most organizations use 10% of its capabilities and hate every minute of it.

| Attack Vector | Detail |
|---|---|
| **Complexity** | SharePoint requires **dedicated admin staff** to configure, maintain, and troubleshoot. Small-to-mid teams cannot justify this overhead. |
| **UX** | SharePoint's user experience is universally criticized. Navigation is confusing, editing is clunky, and the learning curve is steep. |
| **Cost** | $5-$35/user/month **plus** a Microsoft 365 license. Total cost of ownership is significantly higher than the sticker price suggests. |
| **Wiki deprecation** | Microsoft is actively pushing users away from SharePoint wikis toward Viva Topics, creating uncertainty and migration anxiety. |
| **Search** | Default search configuration is terrible. Getting good search results requires custom configuration that most teams never do. |

**Positioning**: *"Enterprise knowledge management without the enterprise complexity."*

**Actionable takeaway**: Target teams currently stuck on SharePoint who lack dedicated IT support. Marketing should emphasize "deploy in 5 minutes" and "no admin required" messaging. A comparison calculator showing total cost of ownership (SharePoint + M365 + admin hours vs Knomantem free tier) would be a powerful conversion tool.

---

### vs Obsidian

**Key weakness**: Obsidian is a brilliant personal tool with no real team story. Its graph is a visualization gimmick, not a semantic knowledge structure.

| Attack Vector | Detail |
|---|---|
| **No real-time collaboration** | Obsidian is fundamentally an individual tool. There is no native way for two people to work on the same vault simultaneously. |
| **No web deployment** | Desktop-only. Cannot share a knowledge base with someone who hasn't installed the app. |
| **Graph is visualization-only** | Obsidian's graph view has no typed relationships, no semantic understanding, and no queryable structure. It looks impressive but provides limited utility. |
| **No team features** | No permissions, no shared spaces, no team-wide search, no access controls. |
| **Plugin dependency** | Many features that should be core (tables, kanban, templates) require community plugins of varying quality and maintenance status. |

**Positioning**: *"Obsidian's graph vision, built for teams."*

**Actionable takeaway**: Obsidian users are the most graph-aware audience in the market. They already believe in connected knowledge. Target Obsidian power users who are trying (and failing) to use it for team knowledge management. Build an Obsidian vault importer that preserves links and converts them to typed graph edges.

---

### vs Notion

**Key weakness**: Notion has polish but no depth. Relations exist in databases but cannot be visualized or traversed. The knowledge graph is implicit but invisible.

| Attack Vector | Detail |
|---|---|
| **No graph visualization** | Notion has Relations and Rollups in databases, but there is no way to visualize or navigate these connections as a graph. The structure is there; the interface is not. |
| **No offline mode** | Notion is entirely cloud-dependent. No internet means no access to your knowledge base. |
| **Vendor lock-in** | Limited export capabilities and a proprietary internal format make migration painful. |
| **Performance** | Large workspaces with thousands of pages experience significant performance degradation. |
| **Cost** | $8-$15/user/month for teams. No free tier for team use. |
| **Limited freshness** | Notion's Wiki Verification feature exists but is narrow in scope -- manual, binary (verified/not), and lacks scoring, automation, or staleness detection. |

**Positioning**: *"Notion's polish with knowledge graph intelligence."*

**Actionable takeaway**: Notion users value design and UX. Knomantem's Flutter frontend must match or exceed Notion's visual quality. The graph visualization is the differentiator, but UX quality is the table stakes. Build a Notion import tool that converts database Relations into visible graph edges -- this single demo will sell the product.

---

## Product Roadmap

### Phase 1: MVP (Months 1-4)

**Goal**: Ship a usable product that demonstrates the core thesis -- connected, fresh, searchable knowledge for teams.

| # | Feature | Purpose |
|---|---|---|
| 1 | Rich text editor (AppFlowy Editor) with slash commands | Core editing experience; must feel modern and fast |
| 2 | Markdown import/export | Reduces migration friction; ensures no vendor lock-in |
| 3 | Spaces and page tree hierarchy | Organizational structure for teams |
| 4 | Full-text search with filters (Bleve) | Search must work well from day one; this is a key differentiator |
| 5 | Content freshness tracking and scoring | **Unique differentiator** -- no competitor does this natively |
| 6 | Knowledge graph visualization (typed edges) | **Unique differentiator** -- the visual "wow" moment |
| 7 | Bidirectional linking | Foundation for the graph; automatic backlink creation |
| 8 | Page versioning | Safety net for edits; required for team trust |
| 9 | User authentication (JWT) | Required for any team product |
| 10 | Role-based permissions (Casbin) | Required for any team product |
| 11 | REST API with OpenAPI docs | Enables integrations and demonstrates openness |
| 12 | Cross-platform (Desktop + Web via Flutter) | Maximizes addressable audience from launch |

**Critical success criteria**: A team of 5-10 people can replace their Confluence workspace with Knomantem and have a better experience within the first week.

---

### Phase 2: Growth (Months 5-10)

**Goal**: Add the features that drive retention, expand use cases, and enable word-of-mouth growth.

- **AI-powered auto-tagging** -- Reduces manual overhead, improves search and graph quality
- **AI staleness detection** -- Proactively identifies content that may be outdated based on signals beyond simple age
- **Real-time collaboration** -- WebSocket-based cursors showing who is editing where
- **Templates library** -- Accelerates content creation and enforces consistency
- **Comments and inline discussions** -- Enables asynchronous collaboration on specific content
- **API for integrations** -- Webhooks, events, and endpoints for connecting to external tools
- **Notification system** -- Alerts for staleness, mentions, comments, and content changes
- **Mobile app release** -- Flutter enables this with shared codebase; attacks Confluence's weak mobile story

**Key insight**: AI features in Phase 2 are strategically timed. By this point, users will have enough content in the system for AI to provide real value. Launching AI features too early (before content mass exists) would underwhelm.

---

### Phase 3: Moonshot (Months 11-18)

**Goal**: Build the features that justify enterprise pricing and create defensible moats.

- **CRDT-based real-time co-editing** -- True simultaneous editing (Google Docs-level)
- **AI knowledge gap detection** -- Identifies topics that should be documented but aren't
- **AI content summarization** -- Auto-generated summaries for long pages and page clusters
- **Plugin/extension system** -- Enables community-driven feature development
- **SSO (SAML/OIDC)** -- Enterprise requirement; gates Pro tier adoption
- **Audit logging** -- Compliance requirement for regulated industries
- **Advanced analytics dashboard** -- Usage patterns, knowledge health scores, team engagement metrics
- **Public page sharing** -- External documentation, public knowledge bases

**Key insight**: The plugin system is the most strategically important feature in Phase 3. It transforms Knomantem from a product into a platform, creates an ecosystem moat, and enables long-tail features that the core team would never build.

---

## Monetization Strategy

### License Model

**BSL 1.1** (Business Source License 1.1): Source code is publicly available. Self-hosting is free. The license converts automatically to **Apache 2.0 after 4 years**, ensuring long-term community trust and eliminating "rug pull" concerns.

**Why BSL 1.1**: It provides the transparency and trust benefits of open source while protecting against cloud providers (AWS, GCP, Azure) offering a competing hosted version without contributing back.

### Pricing Tiers

| Tier | Price | Target | Key Features |
|---|---|---|---|
| **Free (Self-hosted)** | $0 | Small teams, startups, developers | All core features, unlimited users, community support |
| **Pro (Self-hosted)** | $5/user/month | Mid-size teams, security-conscious orgs | SSO, audit logs, priority support, advanced analytics |
| **Cloud** | $8/user/month | Teams that don't want to self-host | Managed hosting, automatic updates, 99.9% SLA |
| **Enterprise** | $15/user/month | Large organizations | Custom deployment, dedicated support, compliance features, custom SLA |

**Key pricing insight**: The free tier must be genuinely free with no artificial limits on core features. The upgrade triggers should be **enterprise requirements** (SSO, audit, compliance) not **feature gates** that frustrate users. A team of 200 should be able to use the free tier happily. When their security team mandates SSO, that's the natural upgrade moment.

**Revenue projection context**: At $8/user/month Cloud tier with 1,000 paying users, monthly recurring revenue reaches $8,000. The goal for Year 1 is not profitability -- it is adoption velocity.

---

## Community Strategy

### Principles

1. **Open development from day 1** -- Public GitHub repository, public roadmap, public issue tracker. Nothing is hidden.
2. **Documentation-first** -- Every feature is documented before it ships. Documentation is a first-class product, not an afterthought.
3. **Contributor-friendly** -- "Good first issue" labels, contributor guide, and clear PR review process to lower the barrier to participation.

### Tactics

- **Monthly development updates** -- Blog posts covering what was shipped, what's next, and why. Transparency builds trust.
- **Discord community** -- Real-time discussions, support, feature requests, and community bonding.
- **Plugin API (Phase 3)** -- The single most important community investment. A plugin ecosystem turns users into contributors and creates switching costs.
- **Dogfooding** -- Knomantem's own documentation should be built in Knomantem. This forces the team to experience the product as users do and provides a public showcase.

### Community-Led Growth Channels

- **Hacker News** -- Launch post targeting the "Confluence is terrible" sentiment (broad resonance)
- **Reddit** -- r/selfhosted, r/ObsidianMD, r/notion, r/KnowledgeManagement
- **Dev.to / Hashnode** -- Technical articles on graph-based knowledge management
- **YouTube** -- Demo videos comparing Knomantem graph navigation vs competitor search

---

## Key Strategic Decisions

### 1. Self-hosted first, Cloud second

**Rationale**: Self-hosted users generate community, bug reports, and credibility. Cloud generates revenue. The free self-hosted tier is the top of the funnel; Cloud is the conversion target for teams that grow or want convenience.

### 2. Freshness as the headline differentiator

**Rationale**: Every competitor ignores content freshness. This is not a nice-to-have -- stale content is the #1 reason knowledge bases fail. Leading with freshness positions Knomantem as the solution to a universal, deeply felt pain point.

### 3. Graph as the visual differentiator

**Rationale**: The graph visualization creates an immediate "this is different" reaction in demos and screenshots. It is the most shareable, most memorable feature. The graph sells the product; freshness retains the users.

### 4. AI features in Phase 2, not Phase 1

**Rationale**: AI features require content mass to be useful. Shipping them in MVP would underwhelm because users don't have enough content yet. Phase 2 timing means AI features arrive when users have 3-6 months of content and can immediately see value.

### 5. BSL 1.1, not open source

**Rationale**: Pure open source (MIT/Apache) invites cloud providers to compete with a hosted version. BSL 1.1 prevents this while maintaining source transparency and converting to true open source after 4 years. This is the same model used by MariaDB, CockroachDB, and Sentry.

---

## Immediate Next Steps

1. **Finalize tech stack decisions** -- Confirm AppFlowy Editor integration, Bleve search configuration, and Casbin permission model.
2. **Build the editor and graph first** -- These are the two features that sell the product in demos. Everything else is supporting infrastructure.
3. **Create a Confluence import tool** -- The largest addressable market is unhappy Confluence teams. Make migration as painless as possible.
4. **Set up public GitHub repository** -- Even before MVP, an active repo with a clear README, roadmap, and contributing guide signals seriousness.
5. **Write the "Why Knomantem" landing page** -- Crystallize the positioning for each competitor into a page that answers "why should I switch?"
6. **Establish the freshness scoring algorithm** -- This is the core intellectual property. Define what "fresh" means, how scores are calculated, and how decay works. Document it publicly to build credibility.

---

*This document should be revisited and updated monthly as competitive landscape, product progress, and market feedback evolve.*
