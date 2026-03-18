# SharePoint: Search Capabilities

## Overview

SharePoint's search landscape is in a significant transition. Microsoft Search (keyword-based) is the established search engine for all Microsoft 365 users, while Copilot Search (AI-powered, semantic) is emerging as the next-generation experience for licensed Copilot users. The search experience varies dramatically depending on licensing tier, Copilot adoption, and whether the organization has invested in search schema customization. Understanding SharePoint search requires examining the search schema infrastructure, Microsoft Search, Copilot Search, and the limitations of each.

## Search Schema: The Foundation

The search schema is the structural backbone of SharePoint search. It determines what users can search for, how content is indexed, and how results are presented.

### Crawled Properties

- When SharePoint crawls content, it discovers metadata and content attributes called **crawled properties**.
- Crawled properties are the raw data extracted from documents, list items, and pages during the crawl process.
- Not all crawled properties are included in the search index. They must be mapped to managed properties to become searchable.
- When a site column is added to a SharePoint list or library, a crawled property and managed property are automatically generated and mapped.

### Managed Properties

- Managed properties are the curated set of content and metadata attributes that are included in the search index.
- Each managed property has configurable settings that control search behavior:
  - **Searchable**: Content is added to the full-text index. A query for "Smith" returns items containing "Smith" anywhere, including in the author field.
  - **Queryable**: Enables property-scoped queries (e.g., `author:Smith` returns only items where the author is Smith).
  - **Retrievable**: Content of the property can be returned in search results. Disabling this hides the property from result displays.
  - **Refinable**: The property can be used as a filter/refiner on search results pages. Only available on built-in managed properties.
  - **Sortable**: Results can be sorted by this property's value.
  - **Safe for Anonymous**: Controls whether anonymous users can see the property value.
- Custom managed properties can only be of type Text or Yes/No. For other data types (integer, date, decimal, double, binary), organizations must reuse built-in unused managed properties (e.g., `RefinableString00` through `RefinableString219`, `RefinableDate00` through `RefinableDate19`, `RefinableInt00` through `RefinableInt49`).
- Built-in managed properties can be "renamed" using the alias setting, but their underlying type and characteristics are fixed.
- The search schema can be configured at the tenant level or at the site collection level. Changes at the tenant level apply globally; changes at the site collection level apply only to that collection.

### Auto-Generated Managed Properties

- When a new site column is added or new metadata is discovered during a crawl, SharePoint automatically generates managed properties.
- Auto-generated properties appear grayed out in the search schema with hidden settings.
- Auto-generated properties are case-sensitive. Queries against them must use exact casing.
- Modifying an auto-generated property converts it to a regular managed property and overrides hidden settings, so all settings must be carefully reviewed before making changes.

### Crawling and Re-indexing

- In SharePoint Online, crawling happens automatically on a defined schedule.
- After changes to the search schema (new managed properties, new mappings), content must be re-crawled before changes take effect.
- Schema changes do not trigger automatic re-crawling. Administrators must request a re-index of the affected library or list.
- Re-indexing large libraries can create significant load on the search system.
- Numeric data in Excel files is not indexed (e.g., "123456789" is not searchable, but "PO123456789" is).

### Language and Tokenization

- Search tokenizes content based on detected language. Content in Chinese data sheets is tokenized differently from English data sheets, which can cause the same product identifier to be unsearchable across languages.
- **Language-neutral tokenization** can be enabled on specific managed properties to ensure consistent tokenization regardless of content language.
- **Finer tokenization** helps with metadata containing special characters (hyphens, dots, hash symbols), improving match rates for partial-term queries.

## Microsoft Search (Default for All Users)

Microsoft Search is the standard search experience available to all Microsoft 365 users. It operates across all Microsoft 365 applications and is powered by the search index and Microsoft Graph signals.

### Core Features

- **Unified search box**: Appears in the header bar of all Microsoft 365 apps (SharePoint, Outlook, Teams, Office.com). Users get contextually relevant results based on which app they are searching from.
- **Personalized results**: Microsoft Search uses Microsoft Graph insights to personalize results for each user. Two users searching for the same term may see different results based on their collaboration patterns, recent activity, and role.
- **Search verticals**: Results are organized into verticals such as All, Files, Sites, People, and News. Custom verticals can be created by administrators.
- **Permission-based results**: Only content the user has permission to access appears in results. Search does not change or override permissions.
- **Query understanding**: AI parses search intent from queries, removing superfluous phrases. A search for "how to change my password" triggers results for "change password."
- **Intelligent ranking**: Results are ordered using AI-based relevance algorithms that consider recency, popularity, user signals, and content quality.
- **Suggested results**: As users type, search suggests results based on recent activity and trending content. Users can open results directly from the suggestion dropdown.
- **Search history**: Queries are recorded in the user's search history (personal, not shared with the organization). Users can download or clear their history from the My Account portal.
- **People search**: Integrated with organizational profiles, enabling search for people by name, title, department, or expertise. Results include profile cards with contact information, org chart position, and recent documents.

### Administrator Capabilities

- **Bookmarks**: Promoted results for specific queries (e.g., searching "benefits" surfaces the HR benefits page as a top result).
- **Q&A**: Administrator-defined question-and-answer pairs that appear for relevant queries.
- **Acronyms**: Definitions for organizational acronyms that appear in search results.
- **Floor plans and locations**: Office floor plans and location information for people search.
- **Custom result sources**: Scoped search configurations that limit results to specific site collections, content types, or managed properties.
- **Query rules**: Rules that promote specific results, add result blocks, or change ranking for particular query patterns.
- **Popular query analytics**: Administrators can see which queries are popular across the organization (but not who searched), enabling identification of content gaps.

### Cross-Tenant and Hybrid Search

- Microsoft Search does not search across tenants or show results from other organizations' content.
- In hybrid environments (SharePoint Online + SharePoint Server on-premises), cloud hybrid search can return results from both online and on-premises content, including external content connected to the SharePoint Server environment.

## Copilot Search (For Licensed Users)

Copilot Search is the AI-powered search experience available in the Microsoft 365 Copilot app. It represents a fundamental shift from keyword-based to semantic, conversational search.

### Core Features

- **Natural language queries**: Users can search using conversational language (e.g., "Find the latest Q3 marketing report from the sales team" or "What is our policy on remote work?").
- **Semantic index**: Copilot uses a semantic index that understands intent and meaning beyond exact keyword matches. Synonyms, paraphrases, and conceptual queries work without requiring exact terminology.
- **AI-generated summaries**: Instead of returning a list of links, Copilot generates synthesized answers with key information extracted from multiple source documents.
- **Citations**: Every claim in a Copilot response is linked back to a source document, providing traceability and enabling users to verify information.
- **Conversational follow-up**: Users can ask follow-up questions to refine or expand on initial results, maintaining conversational context.
- **Data source filtering**: Users can scope queries to specific data sources (SharePoint, OneDrive, Teams, Outlook, Power BI, or external sources connected via Copilot connectors).
- **Cross-product synthesis**: Copilot can combine information from SharePoint documents, email threads in Outlook, Teams chat messages, and meeting transcripts into a single answer.

### SharePoint Agents

- Copilot-powered agents can be built in SharePoint and scoped to specific sites, libraries, or folders.
- Agents can be deployed to Teams chats, channels, and meetings.
- Agents use SharePoint content as their knowledge base, providing focused Q&A experiences for specific domains (e.g., an HR policy agent grounded in the HR site's content).

## Restricted SharePoint Search (RSS)

RSS is a governance mechanism for controlling what Copilot can access:

- Limits Copilot and Microsoft Search to a curated allowlist of up to 100 SharePoint sites.
- Designed as a temporary measure for organizations preparing their content and permissions for Copilot deployment.
- Does NOT guarantee that only allowed sites appear in results; recently accessed or shared content from non-allowed sites can still surface.
- If RSS is enabled, SharePoint is blocked entirely in Copilot Studio (preventing agent creation on SharePoint content).
- The 100-site limit is considered "severely restrictive" for larger organizations with thousands of sites and teams.

## SharePoint REST API and KQL

Programmatic search access is available for custom application development:

- **SharePoint Search REST API**: Enables programmatic querying against the SharePoint search index.
- **Keyword Query Language (KQL)**: Structured query syntax for advanced search operations, including property filters, Boolean operators, proximity operators, and wildcards.
- **Managed property queries**: KQL supports querying specific managed properties (e.g., `author:"Jane Smith" AND filetype:docx`).
- **Refiners and facets**: Programmatic access to refinement filters for building custom search interfaces.
- **Pagination**: Support for paging through large result sets.
- **Custom search applications**: Full-featured search experiences can be built on top of the REST API, including custom result rendering, filtering, and analytics.

## Knowledge Agent Search Capabilities (Preview, 2025)

The Knowledge Agent adds content understanding capabilities beyond traditional search:

- Summarizes pages and compares documents to help users understand content without reading entire documents.
- Generates FAQs from content, making knowledge more accessible.
- Creates audio overviews for accessibility.
- Analyzes search behavior to detect content gaps where users are searching but not finding results.
- Represents a convergence of search and content intelligence.

## Known Limitations and Issues

### Copilot Search Limitations

- **License dependency**: Copilot requires a separate per-user license. Non-licensed users have no access to AI-powered search, creating a two-tier experience.
- **Content scope with RSS**: The 100-site limit is insufficient for large organizations, and non-allowed content can still leak into results.
- **Subfolder indexing**: Agents created at a folder level do not automatically index nested subfolders, potentially missing relevant content.
- **Word processing limit**: Copilot has a lower effective word limit than SharePoint's 30,000-word rich text editor capacity. Microsoft recommends keeping page content under 3,000 words for optimal Copilot processing.
- **RSS and Copilot Studio conflict**: Enabling RSS blocks SharePoint use in Copilot Studio entirely.
- **Guest users**: AI-generated answers from SharePoint are not available to guest users in SSO-enabled applications.

### Oversharing Risk

- Copilot surfaces content based on user permissions. If permissions are overly broad (as is common in many organizations), Copilot can expose sensitive content that users technically have access to but were never intended to see.
- Microsoft recommends proactive permission auditing using SharePoint Advanced Management and Microsoft Purview before Copilot deployment.
- This is one of the most frequently cited concerns about Copilot adoption in enterprise settings.

### Search Indexing Lag

- New or modified content may not appear in search results immediately after creation or upload.
- Indexing delays vary by content type, site activity level, and overall tenant load.
- Schema changes require explicit re-indexing requests for affected libraries and lists.
- This can frustrate users who expect real-time discoverability.

### Microsoft Search Limitations

- Purely keyword-based with no semantic understanding (unlike Copilot Search).
- Some features have been retired or are being deprecated (Locations answers, Q&A, Search in Bing retired March 2025).
- Custom search experiences beyond basic configuration require SPFx development expertise.
- Complex permission models can cause search results to appear incomplete when security trimming excludes content the user expected to see.

### General Search Issues

- Search across classic vs. modern SharePoint sites can produce inconsistent results and experiences.
- No built-in faceted search for modern sites without custom development.
- File sync with File Explorer does not always update search indexes promptly.
- The transition between Microsoft Search and Copilot Search creates organizational friction where different users have fundamentally different search experiences for the same content.

## Search Architecture Summary

| Aspect | Microsoft Search | Copilot Search |
|--------|-----------------|----------------|
| **Availability** | All M365 users | Copilot-licensed users only |
| **Query type** | Keyword-based | Natural language, semantic |
| **Results** | List of links | AI-generated summaries with citations |
| **Personalization** | Graph-based relevance | Graph-based + semantic understanding |
| **Cross-product** | Limited (contextual to app) | Full cross-product synthesis |
| **Customization** | Bookmarks, Q&A, verticals, query rules | Agent creation, data source filtering |
| **Follow-up** | New query required | Conversational context maintained |
| **Cost** | Included in M365 | Additional per-user license |

## Summary Assessment

SharePoint's search capabilities are powerful but in transition. The search schema infrastructure (crawled properties, managed properties, query rules) provides deep customization for keyword-based search. Microsoft Search delivers solid, permission-aware, personalized search across Microsoft 365. Copilot Search adds a transformative semantic and conversational layer, but is gated behind additional licensing and introduces new governance challenges (oversharing risk, content scope control). The two-tier experience (Copilot vs. non-Copilot users) creates organizational friction. Organizations investing in proper search schema configuration, metadata tagging, and content governance will see the best search results regardless of which search tier their users are on.
