# Frontend/Backend Integration Analysis

## Summary

The API contract between the Flutter frontend and Go backend is **remarkably well-aligned** for a project at this stage. The Flutter `api_service.dart` was clearly written with knowledge of the backend route structure. The main gaps are in features that don't exist yet (WebSocket client, freshness dashboard screen, rich text editor) rather than in mismatches between what's built on each side.

---

## Endpoint Alignment

| Feature | Flutter API Method | Backend Route | Match? |
|---|---|---|---|
| Login | `login()` | POST `/auth/login` | ✅ |
| Register | `register()` | POST `/auth/register` | ✅ |
| Get current user | `getMe()` | GET `/auth/me` | ✅ |
| Token refresh | `_refreshToken()` | POST `/auth/refresh` | ✅ |
| List spaces | `getSpaces()` | GET `/spaces` | ✅ |
| Get space | `getSpace(id)` | GET `/spaces/:id` | ✅ |
| Create space | `createSpace()` | POST `/spaces` | ✅ |
| Update space | `updateSpace()` | PUT `/spaces/:id` | ✅ |
| Delete space | `deleteSpace()` | DELETE `/spaces/:id` | ✅ |
| List pages | `getPages(spaceId)` | GET `/spaces/:id/pages` | ✅ |
| Get page | `getPage(id)` | GET `/pages/:id` | ✅ |
| Create page | `createPage()` | POST `/spaces/:id/pages` | ✅ |
| Update page | `updatePage()` | PUT `/pages/:id` | ✅ |
| Delete page | `deletePage()` | DELETE `/pages/:id` | ✅ |
| Move page | `movePage()` | PUT `/pages/:id/move` | ✅ |
| Search | `search()` | GET `/search` | ✅ |
| Get freshness | `getPageFreshness()` | GET `/pages/:id/freshness` | ✅ |
| Verify page | `verifyPage()` | POST `/pages/:id/freshness/verify` | ✅ |
| Freshness dashboard | `getFreshnessDashboard()` | GET `/freshness/dashboard` | ✅ |
| Explore graph | `exploreGraph()` | GET `/graph/explore` | ✅ |
| Get page neighbors | `getPageGraph()` | GET `/pages/:id/graph` | ✅ |
| Create edge | `createEdge()` | POST `/pages/:id/graph/edges` | ✅ |
| List tags | `getTags()` | GET `/tags` | ✅ |
| Create tag | `createTag()` | POST `/tags` | ✅ |
| Add tags to page | `addTagsToPage()` | POST `/pages/:id/tags` | ✅ |
| Presence (WebSocket) | **NOT IMPLEMENTED** | WSS `/presence/:pageId` | ❌ |

---

## Data Model Alignment

### Auth models
Flutter `User` model and backend `User` domain entity: aligned. JWT + refresh token flow is identical.

### Page models
Flutter has `PageSummary` (list view) and `PageDetail` (full view with content). Backend returns different shapes for list vs. detail. Alignment is good — the Flutter model serialization expects `data` wrapper which matches the backend response envelope pattern.

**One discrepancy to verify:** Flutter `createPage()` sends `content` as a hardcoded ProseMirror JSON object with a heading node. The backend stores this in JSONB. This works, but until the page editor is built with a real rich text editor, all pages are created with a trivial single-heading document.

### Freshness models
Flutter `FreshnessInfo`, `FreshnessSummary`, `FreshnessDashboardItem` match the backend `Freshness`, `FreshnessSummaryStats`, `FreshnessPageSummary` domain types. Well-aligned.

### Graph models
Flutter `GraphData` (from `models/edge.dart`) with nodes/edges/totalNodes/totalEdges/truncated maps to backend `GraphExploreResult`. The alignment appears correct — `graph_screen.dart` uses this data to populate `GraphView`.

### Search models
Flutter `SearchResponse` / `SearchItem` maps to backend `SearchResult` / `SearchItem`. Freshness is included in search results on both sides. Tags facets are in the domain model on the backend; unclear if Flutter renders facets.

---

## What's Connected End-to-End

These flows are plausibly testable right now (backend running + frontend loaded):
- Login / Register / token refresh
- List and navigate spaces
- List pages in a space
- Create a new page (with minimal content)
- View graph for a page (data loads, display quality depends on `GraphView` widget)
- Search with filters
- View freshness score for a page
- Verify a page as fresh
- Create typed graph edges
- Tag management

---

## What Has Backend but No Frontend

- **Presence (WebSocket)** — backend has WSS handler and in-memory presence hub; Flutter has no WebSocket client at all
- **Freshness dashboard screen** — `api_service.dart` has `getFreshnessDashboard()` but there is no `freshness_dashboard_screen.dart`
- **Version history** — backend has `GET /pages/:id/versions`; no Flutter screen or API call for it
- **Notifications** — backend has notification creation + DB storage; no Flutter screen or API call
- **Audit log** — DB table exists; no API handler, no frontend
- **Page move** — `movePage()` exists in API service but unclear if there's UI for drag-and-drop reordering

---

## What Has Frontend but No Backend

Nothing significant. The Flutter side is well-disciplined about only calling APIs that exist.

---

## Authentication Flow: Cross-Layer Completeness

The auth flow is the most complete cross-layer integration:

1. Flutter login → backend JWT issuance ✅
2. Tokens stored securely in `flutter_secure_storage` ✅
3. Bearer token attached to every request via Dio interceptor ✅
4. 401 response triggers automatic token refresh ✅
5. Backend JWT middleware validates token, attaches user to context ✅
6. Casbin RBAC middleware enforces permissions ✅

This works end-to-end. The main gap is that the Flutter app doesn't yet gracefully handle refresh token expiry (the catch block in `_refreshToken` silently fails; user would need to manually log in again).

---

## Key Integration Risks

1. **Hard-coded localhost URL** in `api_service.dart` (`const _baseUrl = 'http://localhost:8080/api/v1'`). This works only in local development. Needs environment config before any shared testing.

2. **No OpenAPI spec** (referenced in architecture doc, not in codebase). The API contract exists only by convention between the two codebases. Any drift will be caught only at runtime.

3. **Page content schema** — the backend stores ProseMirror JSON but the Flutter page editor doesn't have a ProseMirror-compatible editor installed. When the real editor is added, the content serialization path needs careful validation.

4. **Graph visualization quality** — `GraphView` widget exists and receives data, but its rendering quality is unknown. This is the #1 demo feature. A placeholder `Text('graph goes here')` in that widget would make the entire graph screen worthless despite the data loading correctly.
