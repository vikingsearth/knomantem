# Notion: Knowledge Staleness & Relevance Determination

## Overview
Notion has the **most developed built-in staleness management** of the four products researched, primarily through its Wiki verification feature. However, these features are limited to Business and Enterprise plans, and the overall approach still relies heavily on manual processes.

## Wiki Verification (Business/Enterprise Only)
Notion's most direct staleness feature — "Stale documentation leads to unnecessary thrash at work."

### How It Works
- Convert any page/database to a "wiki" to enable verification
- Page owners can verify content for a specific period or indefinitely
- Verified pages display a blue check mark in search results, @mentions, and citations
- When verification expires, owners are notified via Notion inbox AND email
- Verified pages get **priority in search results**
- Any editor can verify pages they own (not just wiki pages)

### Verification Settings
- Verify "until a specific date" or "indefinitely"
- Configurable per page
- Automatic expiration tracking
- Re-verification reminders

### Wiki Properties
- **Page owner**: Clear accountability for content accuracy
- **Verification status**: Verified, unverified, or needs re-verification
- **Last verified date**: When content was last confirmed accurate
- **Verification expiration**: When re-verification is due

## Page Analytics
- View counts and engagement metrics at page level
- Available across paid plans
- Shows how team members interact with pages
- **Limitations**: Only basic page views and limited edit tracking
- Cannot segment by content type, user role, or time periods
- Can't explore why edit frequency is low for specific teams

### Third-Party Analytics
- **Count**: Transforms Notion data into Page Edit Frequency analysis, auto-tracks editing patterns
- **Notionlytics**: Analytics toolkit providing content usage analytics for Notion pages

## "Last Edited" Tracking
- Every page shows last edited time and author
- Database properties: `Last edited time` and `Last edited by` auto-populated
- Useful for identifying stale content in database views
- Can create filtered views showing "not edited in 90+ days"

## AI-Powered Maintenance (2025)
- Notion AI can summarize pages, identify outdated content
- AI Autofill can flag pages needing review based on content analysis
- Notion 3.0 Agents can be configured to monitor and report on content health
- AI Q&A surfaces answers from verified content preferentially

## Content Lifecycle via Databases
Users can build lifecycle workflows using database properties:
- Status property: Draft → In Review → Published → Needs Update → Archived
- Date properties: `review_due`, `last_reviewed`, `published_date`
- Formula properties: Calculate days since last review
- Filtered views: "Pages needing review this week"
- Automations (via buttons or agents): Update status, notify owners

## What Works Well
- Wiki verification is the most explicit "is this content fresh?" feature among the four products
- Page ownership creates clear accountability
- Verification badges in search create visible trust signals
- Integration with notifications ensures owners are reminded
- Database properties enable custom lifecycle tracking

## What's Missing
- **Free/Plus users excluded**: Verification only on Business/Enterprise
- **No automated staleness detection**: Must manually set verification periods
- **No content comparison**: Can't detect when page X contradicts newer page Y
- **No AI-driven freshness scoring**: No automatic relevance assessment
- **No bulk staleness management**: Must manage page by page
- **Analytics are limited**: Can't create sophisticated content health dashboards
- **No archival workflow**: No auto-archive for expired content
- **Verification fatigue**: If set too aggressively, becomes notification noise

## Comparison
- **vs. Confluence**: Notion's wiki verification is significantly better than Confluence's lack of native review features; Confluence relies on third-party plugins
- **vs. SharePoint**: SharePoint has enterprise-grade retention policies but lacks the simple "is this accurate?" verification that Notion provides
- **vs. Obsidian**: Obsidian has nothing built-in; relies entirely on community plugins
- **Notion's advantage**: Best balance of simplicity and effectiveness for content freshness among the four
