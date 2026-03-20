# Flutter Web-Specific Considerations (2024/2025)

## Context

This app targets Flutter web as a first-class platform alongside iOS, Android, and desktop. The guidance here addresses concerns that are unique to the web target or require platform-specific treatment.

---

## 1. Web Renderer Choice

### Renderer Options (Flutter 3.x)

| Build Command | Renderer | Compatibility | Performance |
|---|---|---|---|
| `flutter build web` | CanvasKit | All modern browsers | Good |
| `flutter build web --wasm` | Skwasm (+ CanvasKit fallback) | Chrome 119+, Firefox 120+, Edge 119+ | Better |

### CanvasKit (Default)

- Downloads ~1.5MB of Skia compiled to WASM
- Runs on all modern browsers without WasmGC support
- Single-threaded rendering
- Best choice until all project dependencies support the new JS interop APIs

### Skwasm (WASM Build Mode)

- Downloads ~1.1MB
- Multi-threaded rendering via Web Workers (requires `SharedArrayBuffer`)
- Noticeably better startup time and frame performance
- Falls back to CanvasKit if the browser doesn't support WasmGC
- **Requirement:** All Dart packages must use `dart:js_interop` and `package:web` — the old `dart:html` and `package:js` are incompatible with WASM

**Recommendation:** Use the default build (`CanvasKit`) for now. Adopt `--wasm` when the dependency ecosystem catches up (check each package's WASM compatibility before switching).

### Enabling Multi-Threading for Skwasm

Skwasm's multi-threaded mode requires these HTTP response headers:

```
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
```

Without them, Skwasm runs single-threaded (still works, just not as fast). Configure these on your CDN or web server.

### Overriding Renderer at Runtime

```html
<!-- web/index.html -->
<script>
  {{flutter_js}}
  {{flutter_build_config}}
  _flutter.loader.load({
    config: {
      renderer: "canvaskit",  // or "skwasm" — must be set before load()
    },
  });
</script>
```

---

## 2. URL Strategy

### Path URL Strategy (Use This)

By default, Flutter web uses hash URLs (`example.com/#/articles`). Switch to path URLs (`example.com/articles`) for a better user experience and SEO-friendly URLs.

```dart
// main.dart — before runApp()
import 'package:flutter_web_plugins/url_strategy.dart';

void main() {
  usePathUrlStrategy();
  runApp(const ProviderScope(child: MyApp()));
}
```

### Web Server Configuration

Path URLs require your server to serve `index.html` for any path that doesn't match a static file. This is the standard Single Page App (SPA) configuration.

**Firebase Hosting** (`firebase.json`):
```json
{
  "hosting": {
    "public": "build/web",
    "ignore": ["firebase.json", "**/.*", "**/node_modules/**"],
    "rewrites": [
      { "source": "**", "destination": "/index.html" }
    ]
  }
}
```

**Nginx:**
```nginx
location / {
  try_files $uri $uri/ /index.html;
}
```

**Apache** (`.htaccess`):
```apache
RewriteEngine On
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^ /index.html [L]
```

---

## 3. SEO Considerations

### Flutter Web Has Limited Native SEO Support

Flutter renders to a `<canvas>` element. Search engine crawlers cannot read canvas content, so Flutter web is **not suitable** for content that needs to be indexed (blog posts, public documentation, marketing pages).

**What this means for this app:**

- If the knowledge management system has publicly shareable articles that should appear in Google search results, those pages need special handling.
- For an authenticated internal tool (knowledge base for a team), SEO is typically irrelevant — skip this concern entirely.

### Options If SEO Matters

1. **Server-Side Rendering with Jaspr** — Dart-native SSR framework that generates real HTML/DOM output. Can share domain models with Flutter.
2. **Separate Marketing Site** — Keep the Flutter app as the authenticated app experience; serve marketing/landing pages as plain HTML/Next.js.
3. **Prerendering service** — Use a headless browser (Prerender.io, Rendertron) to render and cache HTML snapshots for crawlers. Complex to maintain.

### Accessibility Semantics (Helps Crawlers Slightly)

Enable Flutter's accessibility/semantics layer to produce a meaningful DOM structure alongside the canvas. This helps screen readers and may marginally help crawlers:

```dart
// main.dart
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  if (kIsWeb) {
    SemanticsBinding.instance.ensureSemantics();
  }
  runApp(const MyApp());
}
```

---

## 4. Progressive Web App (PWA)

### Default Service Worker — Deprecated

The old default Flutter PWA service worker has been deprecated. Disable it to avoid confusing caching behaviour:

```bash
flutter build web --pwa-strategy=none
```

### Custom Service Worker

For offline support, implement a custom service worker using Workbox:

```bash
npm install workbox-cli --global
workbox generateSW workbox-config.js
```

Reference the generated `service-worker.js` in `web/index.html`.

### Web App Manifest

The `web/manifest.json` file controls how the app appears when installed as a PWA. Customise it:

```json
{
  "name": "Knowledge Management System",
  "short_name": "KMS",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#1976D2",
  "description": "Your personal knowledge management system",
  "icons": [
    { "src": "icons/Icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "icons/Icon-512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "icons/Icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```

### Cache-Control Headers for PWA

Ensure returning users get app updates promptly:

```
# Entry point — never cache
index.html: Cache-Control: no-cache, no-store, must-revalidate

# Hashed assets — cache forever (Flutter hashes asset names in release builds)
*.js, *.wasm, *.css: Cache-Control: max-age=31536000, immutable

# Fonts and images — cache with revalidation
*.woff2, *.png: Cache-Control: max-age=86400, stale-while-revalidate=604800
```

---

## 5. Responsive Layouts

### Breakpoint Strategy

Define consistent breakpoints as constants rather than magic numbers:

```dart
// core/theme/breakpoints.dart
abstract class Breakpoints {
  static const double mobile = 600;
  static const double tablet = 900;
  static const double desktop = 1200;
  static const double widescreen = 1600;
}
```

### `LayoutBuilder` — Respond to Parent Constraints

`LayoutBuilder` provides the parent's constraints, making it the correct tool for content-responsive layouts. Use it instead of `MediaQuery` when you want a widget to respond to the space it's given (not the full screen):

```dart
class AdaptiveArticleGrid extends StatelessWidget {
  const AdaptiveArticleGrid({super.key, required this.articles});
  final List<Article> articles;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final columnCount = switch (constraints.maxWidth) {
          < 600 => 1,
          < 900 => 2,
          < 1200 => 3,
          _ => 4,
        };

        return GridView.builder(
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: columnCount,
            childAspectRatio: 16 / 9,
            crossAxisSpacing: 16,
            mainAxisSpacing: 16,
          ),
          itemCount: articles.length,
          itemBuilder: (context, index) => ArticleCard(article: articles[index]),
        );
      },
    );
  }
}
```

### `MediaQuery` — Screen-Level Decisions

Use `MediaQuery` for decisions that require knowledge of the full screen (navigation pattern selection, system UI insets):

```dart
class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    final useDrawer = screenWidth < Breakpoints.tablet;

    return Scaffold(
      // Mobile: hamburger + drawer
      drawer: useDrawer ? const AppDrawer() : null,
      // Tablet+: persistent navigation rail
      body: Row(
        children: [
          if (!useDrawer) const AppNavigationRail(),
          Expanded(child: child),
        ],
      ),
    );
  }
}
```

### Adaptive Navigation Pattern

Material 3 guidelines recommend:
- `BottomNavigationBar` / `NavigationBar` for mobile (< 600dp)
- `NavigationRail` for tablet (600–1200dp)
- `NavigationDrawer` (always expanded) for desktop (> 1200dp)

```dart
class AdaptiveNavigation extends StatelessWidget {
  const AdaptiveNavigation({super.key, required this.child, required this.selectedIndex, required this.onDestinationSelected});

  final Widget child;
  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;

  static const destinations = [
    NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'Home'),
    NavigationDestination(icon: Icon(Icons.book_outlined), selectedIcon: Icon(Icons.book), label: 'Articles'),
    NavigationDestination(icon: Icon(Icons.settings_outlined), selectedIcon: Icon(Icons.settings), label: 'Settings'),
  ];

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;

    if (width < 600) {
      // Mobile: bottom bar
      return Scaffold(
        body: child,
        bottomNavigationBar: NavigationBar(
          selectedIndex: selectedIndex,
          onDestinationSelected: onDestinationSelected,
          destinations: destinations,
        ),
      );
    }

    if (width < 1200) {
      // Tablet: rail
      return Scaffold(
        body: Row(
          children: [
            NavigationRail(
              selectedIndex: selectedIndex,
              onDestinationSelected: onDestinationSelected,
              labelType: NavigationRailLabelType.selected,
              destinations: destinations
                  .map((d) => NavigationRailDestination(icon: d.icon, selectedIcon: d.selectedIcon!, label: Text(d.label)))
                  .toList(),
            ),
            Expanded(child: child),
          ],
        ),
      );
    }

    // Desktop: extended rail or drawer
    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            extended: true,
            selectedIndex: selectedIndex,
            onDestinationSelected: onDestinationSelected,
            destinations: destinations
                .map((d) => NavigationRailDestination(icon: d.icon, selectedIcon: d.selectedIcon!, label: Text(d.label)))
                .toList(),
          ),
          Expanded(child: child),
        ],
      ),
    );
  }
}
```

---

## 6. Keyboard Shortcuts and Focus Management

### Registering Keyboard Shortcuts

Use `Shortcuts` and `Actions` widgets for declarative keyboard shortcut handling:

```dart
// Global shortcuts
Shortcuts(
  shortcuts: {
    LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.keyN):
        CreateArticleIntent(),
    LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.keyF):
        FocusSearchIntent(),
    LogicalKeySet(LogicalKeyboardKey.escape):
        DismissIntent(),
  },
  child: Actions(
    actions: {
      CreateArticleIntent: CallbackAction<CreateArticleIntent>(
        onInvoke: (_) => context.push('/articles/new'),
      ),
      FocusSearchIntent: CallbackAction<FocusSearchIntent>(
        onInvoke: (_) => _searchFocusNode.requestFocus(),
      ),
    },
    child: child,
  ),
)
```

### `Focus` and `FocusNode` — Tab Navigation

On web, users expect to navigate UI with Tab. Ensure interactive elements have proper focus handling:

```dart
class ArticleCard extends StatefulWidget {
  // ...
}

class _ArticleCardState extends State<ArticleCard> {
  final _focusNode = FocusNode();

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Focus(
      focusNode: _focusNode,
      onKeyEvent: (node, event) {
        if (event is KeyDownEvent &&
            event.logicalKey == LogicalKeyboardKey.enter) {
          widget.onTap?.call();
          return KeyEventResult.handled;
        }
        return KeyEventResult.ignored;
      },
      child: GestureDetector(
        onTap: widget.onTap,
        child: FocusableActionDetector(
          focusNode: _focusNode,
          child: buildCard(context),
        ),
      ),
    );
  }
}
```

### `FocusTraversalGroup` — Control Tab Order

```dart
FocusTraversalGroup(
  policy: ReadingOrderTraversalPolicy(),
  child: Column(
    children: [
      const TextField(decoration: InputDecoration(label: Text('Title'))),
      const TextField(decoration: InputDecoration(label: Text('Content'))),
      ElevatedButton(onPressed: _submit, child: const Text('Save')),
    ],
  ),
)
```

### Tooltips with Keyboard Shortcut Hints

Show keyboard shortcut hints in tooltips to help users discover shortcuts:

```dart
Tooltip(
  message: 'New Article (Ctrl+N)',
  child: IconButton(
    icon: const Icon(Icons.add),
    onPressed: () => context.push('/articles/new'),
  ),
)
```

---

## 7. Web-Specific Limitations and Workarounds

### `dart:io` Not Available

Replace `dart:io` imports with web-compatible alternatives:

```dart
// Bad — fails on web
import 'dart:io';
File('path/to/file').readAsString();
Platform.isAndroid;

// Good
import 'package:flutter/foundation.dart';
kIsWeb;      // true on web
kIsAndroid;  // false on web — but use platform checks carefully
```

For file operations on web, use the browser's file picker APIs via `file_picker` or `image_picker` packages.

### Isolates Not Available on Web

Dart isolates are not supported on Flutter web. Heavy compute tasks that use `compute()` or `Isolate.spawn()` will throw on web.

For web, offload compute to a Dart `Future` (runs on the main thread but still async) or — for truly heavy work — use a Web Worker via `dart:js_interop`. Most CRUD knowledge management operations do not require isolates.

```dart
// This works on mobile but throws on web
final result = await compute(expensiveFunction, data);

// Web-safe: run on microtask queue (doesn't block UI if < ~16ms)
final result = await Future(() => expensiveFunction(data));
```

### Text Selection and Copy/Paste

Flutter web supports text selection and copy/paste natively. For custom text rendering, ensure `SelectionArea` is wrapping selectable content:

```dart
SelectionArea(
  child: Column(
    children: articleParagraphs.map((p) => Text(p)).toList(),
  ),
)
```

### Context Menu

Flutter web shows the system context menu on right-click by default. Suppress it for specific areas if you have a custom context menu:

```dart
// Suppress right-click context menu
Listener(
  onPointerDown: (event) {
    if (event.kind == PointerDeviceKind.mouse &&
        event.buttons == kSecondaryMouseButton) {
      // Handle your custom context menu
    }
  },
  child: MyWidget(),
)
```

---

## 8. Build and Deployment Checklist

```bash
# Production build (CanvasKit)
flutter build web --release --dart-define=ENVIRONMENT=production

# WASM build (when all deps support it)
flutter build web --wasm --release

# Verify build output
ls -lh build/web/
```

- [ ] `usePathUrlStrategy()` called in `main.dart`
- [ ] Web server configured for SPA routing (all paths → `index.html`)
- [ ] Cache-Control headers configured (no-cache on `index.html`, immutable on hashed assets)
- [ ] PWA manifest (`web/manifest.json`) customised with correct name, icons, theme
- [ ] Default PWA service worker disabled (`--pwa-strategy=none`) or replaced with custom
- [ ] CORS headers on image CDN for CanvasKit/Skwasm
- [ ] `SemanticsBinding.ensureSemantics()` called for accessibility
- [ ] `Cross-Origin-Opener-Policy` + `Cross-Origin-Embedder-Policy` headers set (for Skwasm threading)
- [ ] Keyboard shortcuts registered for primary actions (new, search, escape)
- [ ] Tab traversal tested in Chrome and Safari
- [ ] Responsive layout tested at 375px (mobile), 768px (tablet), 1280px (desktop), 1920px (wide)
