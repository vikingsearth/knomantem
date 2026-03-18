# SharePoint: Knowledge Staleness and Relevance

## Overview

SharePoint addresses content staleness and relevance through multiple systems: Microsoft Purview for compliance-driven retention and deletion, SharePoint Advanced Management for site lifecycle governance, version history and analytics for freshness indicators, and the emerging Knowledge Agent for AI-powered content maintenance. The approach is heavily governance-oriented, reflecting SharePoint's enterprise compliance DNA. However, there are significant gaps between compliance-grade content lifecycle management and the practical, day-to-day "is this content still accurate?" review workflows that knowledge management practitioners need.

## Microsoft Purview Retention Policies

Retention policies are the cornerstone of SharePoint's content lifecycle management. They are managed through the Microsoft Purview compliance portal and operate across the entire Microsoft 365 ecosystem.

### How Retention Policies Work

- Retention policies can be applied to all SharePoint sites (including archived sites) or scoped to specific sites, mailboxes, or groups.
- All files stored in SharePoint or SharePoint Embedded (used by Microsoft Loop and Copilot Pages) can be retained.
- When content subject to retention is edited, a copy of the original is automatically preserved in a hidden **Preservation Hold library**. This library is a system location not designed for interactive use.
- When content subject to retention is deleted, it follows a defined path through the first-stage Recycle Bin, second-stage Recycle Bin, and eventual permanent deletion.
- A timer job runs every seven days on the Preservation Hold library. Content older than 30 days that has exceeded its retention period is then deleted. This means actual deletion can take up to 37 days after the retention period expires.
- Content in the Recycle Bin is not indexed and cannot be found by search or eDiscovery.
- Permanent deletion is suspended if the same item is subject to another retention policy, retention label, or eDiscovery hold (the "first principle of retention").

### Retention Actions

- **Retain only**: Content is kept for a specified period. If not modified or deleted during the retention period, nothing happens afterward.
- **Retain then delete**: Content is kept for a specified period, then automatically moved through the deletion pathway. This is the most common configuration (e.g., retain email for three years, then delete).
- **Delete only**: Content is automatically deleted after a specified period if it has not already been deleted by the user.
- **Retain indefinitely**: Content is kept with no expiration date.

### Retention Labels

- Retention labels are applied at the individual item level (document, email, list item), providing exceptions to broad retention policies.
- Labels can be manually applied by users or automatically applied based on content inspection (keywords, searchable properties, sensitive information types, or trainable classifiers).
- Labels can mark items as **records** (preventing modification of locked records and deletion) or **regulatory records** (which cannot be edited or deleted under any circumstances).
- Labels support **disposition review**: a manual review step before final deletion, where designated reviewers decide whether to delete, relabel, or extend retention.
- Retention labels are also used with **Adaptive Protection** (integrated with insider risk management) and **Priority Cleanup** for expedited deletion of sensitive information.
- List items are supported by retention labels but NOT by retention policies (with exceptions for system lists).

### Retention for Document Versions

- When a document with versions is subject to retention, all versions are retained in a single file in the Preservation Hold library (changed in July 2022 for performance).
- If the retention period is based on creation date, all versions expire at the same time as the original.
- If the retention period is based on last modification date, all versions expire at the same time as the most recent version.
- For items marked as records, versions are copied as separate files and can expire independently.
- While retention is active, versioning limits are ignored and old versions are not automatically purged. Users are prevented from deleting versions.

### Retention for Cloud Attachments

- Cloud attachments (embedded links to files shared in Outlook, Teams, or Viva Engage, or referenced in Copilot interactions) can be retained using auto-apply retention label policies.
- A copy of the shared file is stored in the Preservation Hold library at the time of sharing, typically within an hour.
- If the original file is deleted before the copy is created, a temporary one-day retention copy is preserved as a safeguard.

## Retention Labels vs. Retention Policies (Key Differences)

- **Retention policies** are broad: they apply to locations (all SharePoint sites, specific sites, all mailboxes) and govern all content within those locations.
- **Retention labels** are granular: they apply to individual items and allow for exceptions, records declaration, and disposition review.
- Both can coexist on the same content, with the "principles of retention" determining which action takes precedence (retention wins over deletion; longer retention wins over shorter; explicit deletion wins over implicit).

## Site Lifecycle Management (SharePoint Advanced Management)

SharePoint Advanced Management (SAM) is a premium add-on providing governance capabilities specifically targeting site-level staleness:

### Inactive Site Detection and Policy

- Automated policies detect inactive sites based on configurable inactivity thresholds.
- Site owners receive email notifications when their site is classified as inactive.
- Clicking the site URL in the notification does NOT count as activity (preventing false positive reactivation from mere curiosity).
- Read actions within one hour of receiving the notification are also not counted as activity.
- Actual edits to content reset the inactivity status.
- Policy actions include: notifying site owners to confirm continued need, recommending archival, and automated escalation if owners do not respond.

### Microsoft 365 Archive

- Inactive sites can be archived using Microsoft 365 Archive, which reduces storage costs while preserving content.
- Archived sites remain subject to retention policies and labels.
- Retention labels, disposition review, adaptive scopes, and Microsoft Graph API for programmatic label management all continue to work on archived sites.
- Users cannot view or interact with items in archived sites, so manual label application and disposition review of content are not possible.

## Content Freshness Indicators

SharePoint provides several mechanisms for assessing whether content is current:

### Version History

- Every document and page maintains a full version history showing when changes were made and by whom.
- "Last modified" timestamps and author attribution are visible on all items.
- Major/minor versioning can be configured for controlled publishing workflows (drafts vs. published versions).
- Version comparison enables side-by-side diff views showing what changed between edits.

### Page and Site Analytics

- SharePoint provides view counts and traffic trends for pages (unique viewers, total views, time period trends).
- Site usage analytics show overall site activity, storage consumption, and visitor patterns.
- These metrics help identify high-traffic content (which likely needs to be kept current) and ignored content (which may be candidates for review or retirement).

### Search Activity Signals

- Microsoft Search captures query patterns, popular queries, and zero-result queries.
- Administrators can see what users are searching for (but not who searched), which helps identify content gaps or areas where existing content may be insufficient or outdated.
- The Knowledge Agent (preview) uses search behavior analysis to detect content gaps and unmet user needs.

## What SharePoint Lacks for Knowledge Freshness

### No Built-in Review Cycles

- There is no native "this document must be reviewed every X days/months" feature in SharePoint.
- There are no scheduled review reminders at the document level without using Power Automate or a third-party tool.
- There is no concept of a "verification status" (e.g., "verified by subject matter expert on [date]") built into the platform. Organizations can create custom metadata columns for this purpose, but there is no system-enforced workflow.
- Content ownership must be tracked through custom metadata or external processes.

### No Staleness Scoring

- There is no algorithmic relevance or freshness score for individual documents.
- There is no AI-driven content health dashboard that rates content by currency, accuracy, or completeness. (The Knowledge Agent is heading in this direction but remains in preview.)
- There is no automatic comparison of old vs. new content on the same topic to surface potential duplicates or superseded documents.
- There is no "confidence score" indicating how likely a document is to be accurate or current.

### No Native Content Lifecycle Workflow

- Unlike dedicated knowledge management platforms, SharePoint does not have a built-in content lifecycle workflow (create -> review -> approve -> publish -> review -> retire).
- Implementing such a workflow requires combining Purview retention policies, Power Automate flows, custom metadata columns, and potentially custom SharePoint Framework (SPFx) solutions.
- This combination works but requires significant configuration effort and ongoing maintenance, and the pieces are not integrated into a unified experience.

### No Automated Freshness Decay

- SharePoint does not automatically reduce the visibility or search ranking of content based on its age or lack of recent updates.
- A document published five years ago with no edits appears in search results with the same treatment as a document published yesterday, assuming both match the query terms.
- Search ranking considers recency as one of many signals, but there is no explicit "freshness decay" mechanism that knowledge managers can configure.

## Power Automate Workarounds

Organizations commonly use Power Automate to build review workflows that address SharePoint's gaps:

- **Scheduled review reminders**: Flows that check a "next review date" metadata column and send reminder emails to content owners when review is due.
- **Escalation flows**: If a review is not completed within a specified period, the flow escalates to a manager or knowledge management team.
- **Staleness flagging**: Flows that check "last modified" dates against a threshold and automatically tag items as "needs review" or notify owners.
- **Approval workflows**: Multi-stage approval flows for content publishing with designated reviewers and approvers.
- These workarounds are functional but fragmented. They require custom configuration per library or content type, and they add complexity that can be difficult to maintain at scale.

## Knowledge Agent Site Maintenance (Preview, 2025)

The Knowledge Agent introduces AI-powered capabilities that begin to address the staleness gap:

- Analyzes search behavior to detect content gaps (queries with no good results).
- Automatically identifies and fixes broken links.
- Identifies inactive pages that may need retirement.
- Surfaces content that needs attention based on usage patterns and content age.
- Provides AI-powered content health assessments.
- Still in preview with limited availability; GA timeline is not confirmed.

## Information Governance Integration

SharePoint's staleness management is tightly integrated with Microsoft's broader information governance ecosystem:

- **Sensitivity labels** (Microsoft Purview Information Protection): Classify documents by sensitivity level (Public, Internal, Confidential, Highly Confidential), controlling access and handling requirements throughout the content lifecycle.
- **Data Loss Prevention (DLP)**: Policies that detect sensitive information in SharePoint content and enforce handling rules (block sharing, restrict access, notify compliance officers).
- **eDiscovery**: Legal hold and content search capabilities that interact with retention policies. Content under eDiscovery hold cannot be permanently deleted regardless of retention policy settings.
- **Audit logs**: Comprehensive audit trails for all content actions (creation, modification, deletion, sharing, permission changes), accessible through the Purview compliance portal.
- **Compliance Manager**: Risk assessment tool that evaluates the organization's compliance posture across various regulatory frameworks (GDPR, HIPAA, SOX, etc.) and provides recommendations.

## Summary Assessment

SharePoint's approach to knowledge staleness is fundamentally governance-centric. It excels at compliance-driven retention and deletion (regulated industries, legal requirements, audit readiness) but lacks the lighter-weight, KM-practitioner-focused workflows for routine content freshness management. The gap between "delete this document after three years for regulatory compliance" and "is this document still accurate and useful?" is significant. Organizations that need practical knowledge freshness management must supplement SharePoint's native capabilities with Power Automate workflows, custom metadata schemes, and potentially third-party tools. The Knowledge Agent represents a promising step toward closing this gap, but it remains in preview.
