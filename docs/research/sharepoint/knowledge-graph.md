# SharePoint: Knowledge Graph Capabilities

## Overview

SharePoint's knowledge graph story is complex and in active transition. Microsoft built a dedicated knowledge graph product (Viva Topics) that was fully retired on February 22, 2025. The underlying data infrastructure (Microsoft Graph) remains as the platform layer, and AI-powered knowledge discovery has shifted to Copilot. Understanding where SharePoint stands on knowledge graph capabilities requires examining what the Microsoft Graph provides, what Viva Topics offered before its retirement, what remains today, and what gaps exist compared to a true knowledge graph system.

## Microsoft Graph (The Platform Layer)

Microsoft Graph is the unified API and data layer that connects all Microsoft 365 services. It is not a knowledge graph in the traditional sense (it does not perform entity extraction or ontology management), but it provides the relational data fabric on which knowledge-related features are built.

- **Single endpoint**: `https://graph.microsoft.com` provides REST API access to data across Microsoft 365, Windows, and Enterprise Mobility + Security services.
- **Core data entities exposed**: Users, groups, calendar events, emails, files (SharePoint and OneDrive), chats (Teams), contacts, tasks (Planner, To Do), OneNote notebooks, and organizational hierarchies.
- **Relationship modeling**: Microsoft Graph models relationships between entities (e.g., a user's manager, a user's group memberships, a file's author, a file's sharing history, people a user frequently collaborates with). These relationships power people cards, org charts, and "related people" suggestions.
- **Signals and insights**: The Graph captures behavioral signals such as which documents a user has recently worked on, trending content across the organization, and collaboration patterns. These signals feed into Microsoft Search relevance ranking and Copilot responses.
- **People API**: Returns people relevant to a given user based on collaboration patterns (emails, meetings, document co-authoring), not just the org chart. This provides implicit expertise identification.
- **Trending and used insights**: The `/insights/trending` and `/insights/used` endpoints surface content that is gaining organizational attention or that a specific user has been interacting with.
- **SharePoint integration**: Microsoft Graph exposes SharePoint sites, lists, list items, drives (document libraries), and drive items. Programmatic access to SharePoint content and metadata is available through the Graph API.
- **Security trimming**: All Graph API responses are security-trimmed to the calling user's permissions. Users can only access data they are authorized to see.
- **SDKs and client libraries**: Available for .NET, JavaScript, Java, Python, Go, PHP, and PowerShell, enabling custom application development on top of the Graph.

## Copilot Connectors (Formerly Graph Connectors)

Copilot connectors extend the Microsoft Graph by bringing external data into the Microsoft 365 index:

- **Incoming data direction**: Connectors pull data from external sources (Salesforce, ServiceNow, Jira, SAP, databases, file shares, etc.) into Microsoft Graph services.
- **Indexing**: External data is indexed as "external items" and becomes searchable alongside native Microsoft 365 content in Microsoft Search and Copilot.
- **Over 150 pre-built connectors** from Microsoft and partners, covering common enterprise data sources.
- **Custom connectors**: Organizations can build their own connectors using the Microsoft Graph connectors API to index proprietary data sources.
- **Relevance**: External items participate in the same ranking and relevance algorithms as native content, meaning Copilot can synthesize answers from both SharePoint documents and external knowledge bases.

## Microsoft Graph Data Connect

For large-scale analytics and data science on Microsoft 365 data:

- **Bulk data export**: Provides scalable delivery of Microsoft Graph data to Azure data stores (Azure Synapse, Azure Data Factory) for offline analysis.
- **Consent and governance**: Administrators grant granular consent at the group and resource-type level, with the ability to exclude specific users.
- **Use cases**: Knowledge base automation, collaboration pattern analysis, expertise mapping at scale, meeting utilization analysis, and fraud detection.
- **Not real-time**: Data Connect operates on a recurrent schedule with cached data, unlike the real-time Graph API.

## Viva Topics (Retired February 22, 2025)

Viva Topics was Microsoft's most ambitious attempt at a true knowledge graph within the Microsoft 365 ecosystem. Understanding what it offered is essential for understanding the current gap.

### What Viva Topics Provided

- **Automatic entity extraction**: The Alexandria engine (from Microsoft Research) used probabilistic programming and NLP to automatically identify topics (entities) from documents, emails, conversations, and other Microsoft 365 content.
- **Entity resolution**: The system merged hundreds or thousands of references to the same concept into a single, robust topic entry, handling synonyms, abbreviations, and ambiguity.
- **Topic cards (knowledge cards)**: When a user encountered a topic term in SharePoint pages, Office documents, Teams, or Outlook, a hover card appeared showing the topic's description, related documents, related sites, and identified subject matter experts.
- **Topic pages**: Dedicated SharePoint pages for each topic, containing an AI-generated description, curated resources, related topics, and identified experts. Topic pages could be manually curated by designated knowledge managers.
- **Topic Center**: A dedicated SharePoint site serving as the central management hub for all discovered and curated topics, with analytics on topic usage and coverage.
- **Relationship mapping**: AI-suggested relationships between topics, providing a navigable web of organizational knowledge.
- **Cross-application surfacing**: Topics appeared inline across Microsoft Search results, SharePoint modern pages, Office apps, Outlook, and Teams, making knowledge discovery ambient and contextual.
- **Subject matter expert identification**: The system automatically identified people with expertise on a given topic based on their content contributions and collaboration patterns.

### What Was Lost at Retirement

- No automatic entity extraction from documents or conversations.
- No topic cards or inline knowledge cards in Office apps, Teams, or Outlook.
- No AI-suggested relationships between concepts.
- No visual knowledge graph or topic relationship map.
- No automatic subject matter expert identification tied to topics.
- No Topic Center analytics or management views.
- Published topic pages were converted to standard SharePoint pages (still editable, but no longer AI-enhanced).
- The integration between Viva Topics and Viva Engage was retired; Engage returned to a simplified public topics model.

## Current State of Knowledge Graph Capabilities (Post-Viva Topics)

### What Remains

- **Microsoft Graph as data fabric**: The relational data layer connecting people, files, emails, events, and activities continues to function. Search relevance, people recommendations, and Copilot grounding all depend on it.
- **People profiles and expertise signals**: Microsoft Graph still surfaces collaboration patterns and trending content, providing implicit expertise signals. However, these are not organized into a formal topic taxonomy.
- **SharePoint metadata and taxonomy**: The managed metadata service (term store) provides controlled vocabularies and hierarchical taxonomies. This is the closest remaining feature to a formal ontology, but it requires manual creation and maintenance, and it is not automatically connected to content through entity extraction.
- **SharePoint pages as knowledge artifacts**: Existing topic pages (converted from Viva Topics) and manually created knowledge pages remain discoverable through Microsoft Search and serve as grounding content for Copilot.
- **Microsoft Search**: Continues to use Graph signals for relevance ranking and personalization.

### What Is Missing (Compared to a True Knowledge Graph)

- **No automatic entity extraction**: Documents are not automatically parsed to identify and extract entities, concepts, or topics. Any tagging must be done manually or through Autofill Columns (which populates metadata columns, not a knowledge graph).
- **No relationship typing**: There is no formal mechanism to define typed relationships between concepts (e.g., "Technology X is-used-in Project Y" or "Person A is-expert-in Domain B"). Microsoft Graph models fixed relationship types (manager-of, member-of, author-of) but does not support custom relationship definitions.
- **No graph visualization**: There is no visual representation of knowledge connections. Users cannot browse a visual map of how topics, documents, and people relate to each other.
- **No ontology management**: While the managed metadata term store provides taxonomy, it does not function as an ontology (no relationship types, no inference rules, no property definitions on concepts).
- **No inference or reasoning**: The system cannot derive implicit relationships from explicit ones. For example, it cannot infer that if Person A is an expert in Machine Learning and Machine Learning is a subfield of AI, then Person A has some expertise in AI.
- **No knowledge curation workflow**: There is no built-in workflow for knowledge managers to review, approve, or curate AI-discovered knowledge (as Viva Topics provided).

## Copilot as Knowledge Graph Successor

Microsoft's strategic position is that conversational AI (Copilot) makes explicit knowledge graphs less necessary. Instead of building and maintaining a graph, users ask Copilot questions and it finds answers from the underlying data.

- **Natural language queries**: Users can ask questions like "Who is the expert on our Azure migration project?" and Copilot synthesizes an answer from SharePoint content, emails, Teams chats, and meeting transcripts.
- **Semantic index**: Copilot uses a semantic index that goes beyond keyword matching to understand intent and meaning. This provides some of the contextual understanding that a knowledge graph would offer.
- **Citation-based answers**: Copilot responses include citations linking back to source documents, providing traceability.
- **Cross-product synthesis**: Copilot can pull information from SharePoint, OneDrive, Teams, Outlook, and external sources (via connectors) to compose answers.
- **Data source filtering**: Users can scope Copilot queries to specific data sources (e.g., only SharePoint, only a specific site).

### Limitations of Copilot as a Knowledge Graph Replacement

- **No persistent structure**: Copilot answers are ephemeral. There is no persistent, navigable knowledge structure that accumulates over time. Each query starts fresh.
- **No explicit relationship mapping**: Copilot can answer relationship questions conversationally, but there is no stored, queryable graph of relationships.
- **No visual exploration**: Users cannot visually browse connections between topics, as they could with Viva Topics' topic pages and relationship views.
- **Accuracy depends on content quality**: Copilot is only as good as the underlying content. If SharePoint content is stale, disorganized, or poorly tagged, Copilot's answers will reflect that.
- **License dependency**: Copilot requires a separate per-user license. Users without Copilot access have no AI-powered knowledge discovery capability beyond basic Microsoft Search.
- **No entity normalization**: Copilot does not automatically resolve that "ML," "machine learning," and "Machine Learning" all refer to the same concept. A knowledge graph would handle this through entity resolution.

## Knowledge Mining with SharePoint Syntex / SharePoint Premium

SharePoint Premium (formerly Syntex) provides some knowledge mining capabilities that partially address the knowledge graph gap:

- **Document understanding models**: AI models trained to classify documents and extract specific fields from structured and semi-structured documents (invoices, contracts, forms).
- **Prebuilt models**: Out-of-the-box models for common document types (invoices, receipts, tax forms).
- **Content assembly**: Template-based document generation using extracted metadata.
- **Taxonomy tagging**: Automatic tagging of documents using terms from the managed metadata service.
- These capabilities are closer to document processing and content intelligence than a traditional knowledge graph, but they contribute to the metadata layer that enables better search and discovery.

## Third-Party and Custom Options

The retirement of Viva Topics has created market space for knowledge graph solutions that integrate with SharePoint:

- **Graph databases** (Neo4j, Amazon Neptune, Azure Cosmos DB with Gremlin API) can ingest SharePoint content via the Microsoft Graph API and build custom knowledge graphs.
- **Enterprise knowledge graph platforms** (PoolParty, Ontotext GraphDB, Stardog) offer SharePoint integrations for taxonomy management, entity extraction, and graph-based navigation.
- **ISV solutions** specifically targeting the post-Viva Topics gap are emerging in the Microsoft partner ecosystem.
- **Custom development**: Organizations with development resources can build knowledge graph layers using the Microsoft Graph API, Azure Cognitive Services for entity extraction, and a graph database for storage and querying.

## Summary Assessment

Microsoft's knowledge graph strategy has undergone a fundamental shift. The retirement of Viva Topics eliminated the only built-in knowledge graph capability in the Microsoft 365 ecosystem, and the replacement strategy centers on Copilot's conversational AI rather than structured knowledge representation. Microsoft Graph provides a rich relational data layer, but it is an API platform, not a knowledge graph in the traditional KM sense. For organizations that need structured knowledge representation, entity extraction, ontology management, or visual knowledge exploration, SharePoint now requires supplementation with third-party tools or custom development. The KM community remains divided on whether Copilot's conversational approach is an adequate substitute for structured knowledge graphs, with many practitioners arguing that both are needed for comprehensive knowledge management.
