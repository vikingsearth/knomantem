# Frontend Analysis: Knomantem Flutter App

## Overview

The Flutter app lives in `web/` and targets web, desktop, and mobile from a shared codebase. State management is Riverpod 2.6 (not Riverpod 3.0 as the architecture doc states — pubspec shows `^2.6.1`). Routing uses `go_router`. API calls use `dio` with a JWT interceptor and automatic token refresh. No rich text editor library is included in `pubspec.yaml`.

---

## Screens

### Login (`screens/login_screen.dart`)
Status: **Functional scaffold** — renders UI, calls API, handles auth state. Likely complete for MVP purposes.

### Home (`screens/home_screen.dart`)
Status: **Unknown completeness** — lists spaces, entry point to the app. Not read in detail but provider is wired.

### Space (`screens/space_screen.dart`)
Status: **Functional** — lists pages in a space, navigates to page editor. Page tree widget available.

### Page Editor (`screens/page_editor_screen.dart`)
Status: **Critical gap.** The `pubspec.yaml` contains no rich text editor dependency. `appflowy_editor` is not installed. The architecture doc says "Flutter + AppFlowy Editor" but the package is missing. The editor screen almost certainly renders a plain `TextField` or nothing useful. This is the most important missing piece in the frontend.

### Search (`screens/search_screen.dart`)
Status: **Likely functional scaffold** — search provider is implemented, API call is wired in `api_service.dart`.

### Graph (`screens/graph_screen.dart`)
Status: **Surprisingly complete for a scaffold.** The screen:
- Has a proper `StateNotifier` with loading/error/data states
- Calls `api.exploreGraph()` with depth and edge type filter controls
- Has a depth slider (1–5 hops) and edge type dropdown filter
- Passes data to `GraphView` widget
- Shows node count, edge count, truncation warning in status bar
- Navigates to page on node tap

The missing piece is `widgets/graph_view.dart` — this widget receives the data but its internal canvas rendering is unknown quality.

---

## Widgets

### `widgets/graph_view.dart`
Receives `GraphData` with nodes/edges. Likely renders something, but how well (force-directed layout? static grid? custom painter?) is unknown without reading it. This is the "wow" demo feature and its quality matters enormously.

### `widgets/page_tree.dart`
Hierarchical page list for space navigation. Status unknown but referenced in space screen.

### `widgets/freshness_badge.dart`
Visual indicator of freshness status. Referenced but likely a simple color-coded chip.

### `widgets/editor_toolbar.dart`
Rich text editor toolbar buttons. Without `appflowy_editor` installed, this toolbar has nothing to attach to.

### `widgets/search_bar.dart`
Search input widget. Likely functional.

---

## State Management

Riverpod providers exist for:
- `auth_provider.dart` — auth state, login/logout
- `space_provider.dart` — space list + CRUD
- `page_provider.dart` — page CRUD + current page
- `search_provider.dart` — search query + results
- `freshness_provider.dart` — freshness data + verify

**No provider for:**
- Graph state (handled locally in `graph_screen.dart` with a `StateNotifierProvider.family`)
- Tags (no tag provider)
- Presence/WebSocket (not implemented)
- Notifications (not implemented)

The provider layer is well-structured and handles loading/error states correctly.

---

## API Service

`api_service.dart` is the most complete file in the frontend. It covers:
- Auth: login, register, getMe, token management (refresh interceptor)
- Spaces: full CRUD
- Pages: full CRUD + move
- Search: with all filter params
- Freshness: get, verify, dashboard
- Graph: explore, getPageGraph, createEdge
- Tags: list, create, addToPage

**Notable:** the `createEdge` method posts to `/pages/:sourcePageId/graph/edges` — this matches the backend route structure. Models are also well-matched (see `current-bff.md`).

---

## Code Quality

- Clean, consistent Dart: proper null safety, named records for complex return types, private helper `_handleRequest`
- No linting violations apparent in reviewed files
- `json_annotation` / `json_serializable` used for model serialization — correct approach
- Token refresh interceptor is production-quality (retries original request after refresh)
- Hard-coded `http://localhost:8080` base URL — needs environment variable or config before any real deployment

---

## What a Developer Picking This Up Needs to Know

1. **Install `appflowy_editor`** in `pubspec.yaml` before touching `page_editor_screen.dart`. Without it the editor is non-functional.
2. **Inspect `graph_view.dart`** before assuming graph visualization works. It may be a placeholder `CustomPainter` or a real force-directed layout.
3. The API service is the most stable part — trust it, extend it carefully.
4. No WebSocket client exists anywhere. Presence requires building the WS layer from scratch.
5. `baseUrl` is hardcoded to localhost — add an `ApiConfig` or Flutter flavor system before first shared build.
6. Riverpod version is 2.6, not 3.0 as the architecture doc states. Don't follow architecture doc examples for Riverpod 3.0 syntax.
