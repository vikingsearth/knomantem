# Flutter Performance — Best Practices (2024/2025)

## Profiling First, Optimise Second

Always measure before optimising. Run in **profile mode** — debug mode adds significant overhead and gives misleading results:

```bash
flutter run --profile
flutter build web --profile
```

Use Flutter DevTools (Performance tab, Widget Rebuild Inspector) to identify hot paths before applying any of the patterns below.

---

## 1. Widget Rebuild Optimisation

### Use `const` Constructors Everywhere You Can

`const` widgets are canonicalised by the Dart compiler — Flutter skips them during rebuilds entirely. This is the single highest-value optimisation for most apps.

```dart
// Bad — rebuilt on every parent rebuild
Text('Hello World')
Padding(padding: EdgeInsets.all(16), child: Icon(Icons.home))

// Good — skipped during parent rebuilds
const Text('Hello World')
const Padding(padding: EdgeInsets.all(16), child: Icon(Icons.home))
```

Enable `prefer_const_constructors` and `prefer_const_literals_to_create_immutables` in `analysis_options.yaml` — the linter will flag missing `const`.

### Scope `setState` and `ref.watch` to the Smallest Subtree

Never call `setState` high in the tree when only a leaf widget needs to change. The same principle applies to Riverpod: watch providers as deep in the widget tree as possible.

```dart
// Bad — the entire screen rebuilds on count change
class CounterScreen extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref.watch(counterProvider); // Rebuilds everything below
    return Scaffold(
      body: Column(
        children: [
          ExpensiveHeaderWidget(),     // Rebuilt unnecessarily
          Text('$count'),
          AnotherExpensiveWidget(),    // Rebuilt unnecessarily
        ],
      ),
    );
  }
}

// Good — only the Text rebuilds
class CounterScreen extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Column(
        children: [
          ExpensiveHeaderWidget(),
          _CountDisplay(),             // Isolated consumer
          AnotherExpensiveWidget(),
        ],
      ),
    );
  }
}

class _CountDisplay extends ConsumerWidget {
  const _CountDisplay();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final count = ref.watch(counterProvider);
    return Text('$count');
  }
}
```

### `ref.watch` with `select` — Subscribe to a Slice

When you only need part of a provider's state, use `.select` to prevent rebuilds when unrelated parts of the state change:

```dart
// Rebuilds whenever any part of the User object changes
final user = ref.watch(userProvider);

// Rebuilds ONLY when displayName changes
final name = ref.watch(userProvider.select((u) => u.displayName));

// Example: only rebuild when loading state changes (not on data changes)
final isLoading = ref.watch(articleListProvider.select((s) => s.isLoading));
```

`.select` accepts any value that implements `==`. For custom objects, use `freezed` which auto-generates proper equality.

### Avoid Rebuilding Animated Widgets' Static Children

```dart
// Bad — ExpensiveWidget rebuilds every animation frame
AnimatedBuilder(
  animation: _controller,
  builder: (context, child) {
    return Transform.rotate(
      angle: _controller.value,
      child: const ExpensiveWidget(), // Rebuilt on every frame despite being const
    );
  },
);

// Good — static child is built once and passed in
AnimatedBuilder(
  animation: _controller,
  builder: (context, child) {
    return Transform.rotate(
      angle: _controller.value,
      child: child,
    );
  },
  child: const ExpensiveWidget(), // Built once
);
```

### Extract Widgets, Not Functions

Extracting UI into a function (e.g., `_buildHeader()`) does NOT prevent rebuilds — it is inlined at the call site. Extracting into a `Widget` class allows Flutter to skip rebuilding it when nothing changes.

```dart
// Bad — _buildHeader runs on every parent rebuild
Widget _buildHeader() => const Text('Header');

// Good — Flutter can short-circuit if ArticleHeader's inputs haven't changed
class ArticleHeader extends StatelessWidget {
  const ArticleHeader({super.key});
  @override
  Widget build(BuildContext context) => const Text('Header');
}
```

---

## 2. Lists — `ListView.builder` vs `Column`

### Always Use Lazy Builders for Long Lists

`Column` builds all children eagerly. `ListView.builder` builds only the visible items plus a small buffer.

```dart
// Bad — builds all 1000 items at startup regardless of scroll position
Column(
  children: items.map((item) => ArticleCard(item: item)).toList(),
)

// Good — builds only visible items
ListView.builder(
  itemCount: items.length,
  itemBuilder: (context, index) => ArticleCard(item: items[index]),
)
```

### `ListView.builder` Configuration Tips

```dart
ListView.builder(
  // Provide itemExtent when all items are the same height — enables O(1) scroll position calculation
  itemExtent: 80.0,
  // Or itemExtentBuilder for variable but known heights

  // addRepaintBoundaries: false if items are already simple
  // (default true — usually leave this on)

  itemCount: articles.length,
  itemBuilder: (context, index) {
    return ArticleCard(article: articles[index]);
  },
)
```

### Use `ListView.separated` for Dividers

```dart
ListView.separated(
  itemCount: articles.length,
  separatorBuilder: (_, __) => const Divider(height: 1),
  itemBuilder: (context, index) => ArticleCard(article: articles[index]),
)
```

### `CustomScrollView` + `SliverList` for Mixed Content

```dart
CustomScrollView(
  slivers: [
    const SliverAppBar(expandedHeight: 200, floating: true),
    SliverList.builder(
      itemCount: articles.length,
      itemBuilder: (context, index) => ArticleCard(article: articles[index]),
    ),
  ],
)
```

### Infinite Scroll / Pagination

With Riverpod, implement pagination via a family provider indexed by page or a single `AsyncNotifier` that accumulates pages:

```dart
@riverpod
class PaginatedArticles extends _$PaginatedArticles {
  int _currentPage = 1;

  @override
  Future<List<Article>> build() async {
    return ref.watch(articleRepositoryProvider).getArticles(page: 1);
  }

  Future<void> loadNextPage() async {
    final current = state.valueOrNull ?? [];
    _currentPage++;
    final next = await ref.read(articleRepositoryProvider).getArticles(page: _currentPage);
    state = AsyncData([...current, ...next]);
  }
}
```

---

## 3. Image Caching and Lazy Loading

### Use `cached_network_image` (^3.4.x)

Never use `Image.network` in production — it does not cache to disk, causing redundant network requests on every build.

```yaml
dependencies:
  cached_network_image: ^3.4.1
```

```dart
CachedNetworkImage(
  imageUrl: article.coverImageUrl,
  width: 320,
  height: 180,
  fit: BoxFit.cover,
  // Placeholder shown while loading
  placeholder: (context, url) => Container(
    color: Theme.of(context).colorScheme.surfaceVariant,
    child: const Center(child: CircularProgressIndicator.adaptive()),
  ),
  // Error widget for failed loads
  errorWidget: (context, url, error) => const Icon(Icons.broken_image),
  // Fade-in for smooth appearance
  fadeInDuration: const Duration(milliseconds: 200),
  // Resize in the image cache to avoid memory bloat
  memCacheWidth: 640,
  memCacheHeight: 360,
)
```

### `memCacheWidth` / `memCacheHeight` — Critical for Memory

If you display a 2000×1500px image in a 100×75 thumbnail, the decoded bitmap takes 10× more memory than needed. Always set `memCacheWidth` / `memCacheHeight` to roughly 2× the display size (accounting for device pixel ratio).

### Preloading Critical Images

```dart
// Preload the hero image before navigating to detail screen
await precacheImage(
  CachedNetworkImageProvider(article.coverImageUrl),
  context,
);
context.go('/articles/${article.id}');
```

### Lazy Loading with `ListView.builder`

In a `ListView.builder`, `CachedNetworkImage` loads images only when their tile scrolls into view. No additional configuration needed — this is the correct default pattern.

### Flutter Web — CORS Requirement

In CanvasKit/Skwasm renderers, images must be served with CORS headers (`Access-Control-Allow-Origin: *` or your specific origin). Without this, images fail to load silently. Configure your CDN or API server accordingly.

---

## 4. Expensive Operations to Avoid

### `Opacity` Widget in Animations

`Opacity` forces the Flutter engine to create an offscreen layer on every frame. For animations, use `AnimatedOpacity` or `FadeTransition` instead:

```dart
// Bad in animations
Opacity(opacity: _fadeValue, child: MyWidget())

// Good
FadeTransition(opacity: _fadeAnimation, child: const MyWidget())
AnimatedOpacity(opacity: isVisible ? 1.0 : 0.0, duration: const Duration(milliseconds: 300), child: const MyWidget())
```

### `ClipRRect` in Lists

Clipping is expensive. For rounded corners on list items, prefer `decoration: BoxDecoration(borderRadius: ...)` on a `Container` or `DecoratedBox` — it does not clip the render layer.

### `Intrinsic` Widgets

`IntrinsicHeight` and `IntrinsicWidth` force a two-pass layout, measuring all children before sizing. Avoid in lists. Set fixed heights or use `Expanded`/`Flexible` instead.

### `saveLayer()` — Avoid Custom Painters That Call It

`saveLayer` forces GPU render target switching. It is called internally by `Opacity`, `ShaderMask`, `ColorFilter`, `BackdropFilter`, and some `CustomPainter` implementations. Check the DevTools Performance overlay's "Offscreen layers" diagnostic.

---

## 5. Web-Specific Performance

### Renderer Choice

| Mode | Command | Renderer | Best For |
|---|---|---|---|
| Default | `flutter build web` | CanvasKit | Broad browser support, legacy packages |
| WASM | `flutter build web --wasm` | Skwasm (+ CanvasKit fallback) | Performance-critical apps, modern browsers |

**Recommendation for this app**: Start with the default build (`CanvasKit`). Switch to `--wasm` once all dependencies support the new JS interop (`dart:js_interop` + `package:web`). The WASM build gives noticeably better frame performance and startup time due to multi-threaded rendering via Web Workers.

To force Skwasm in the HTML bootstrap (override auto-selection):
```html
const config = { renderer: 'skwasm' };
_flutter.loader.load({ config });
```

Note: Skwasm requires the server to set `Cross-Origin-Opener-Policy: same-origin` and `Cross-Origin-Embedder-Policy: require-corp` headers for `SharedArrayBuffer` (multi-threading). Without these, it falls back to single-threaded Skwasm.

### Tree Shaking and Icon Fonts

Flutter automatically tree-shakes Dart code. However, icon fonts (like Material Icons) are not tree-shaken unless you enable it explicitly:

In `pubspec.yaml`:
```yaml
flutter:
  fonts:
    - family: MaterialIcons
      fonts:
        - asset: fonts/MaterialIcons-Regular.otf
  # Tree-shake icons — only include used icon glyphs
```

Or use SVG icons via `flutter_svg` for precise control over bundle size.

### Deferred Loading — Split the Bundle

Use Dart's `deferred` keyword to load rarely-used screens only when needed:

```dart
// In routes or screen entry points
import 'package:my_app/features/admin/admin_panel.dart' deferred as admin;

// Somewhere in your route builder
FutureBuilder<void>(
  future: admin.loadLibrary(),
  builder: (context, snapshot) {
    if (snapshot.connectionState != ConnectionState.done) {
      return const CircularProgressIndicator();
    }
    return admin.AdminPanelScreen();
  },
)
```

This creates a separate `*.js` chunk that is only downloaded when the admin route is first accessed. Ideal for large features used by a subset of users.

### Cache-Control Headers

Ensure your web server serves flutter_bootstrap.js and main.dart.js with aggressive caching for returning users:

```
# Static assets (hashed filenames) — cache for 1 year
*.js, *.wasm, *.css, *.png → Cache-Control: max-age=31536000, immutable

# Entry point — never cache (or short TTL)
index.html, flutter_bootstrap.js → Cache-Control: no-cache
```

### Reduce Initial Load — Font Strategy

Google Fonts loaded via `google_fonts` package makes network requests on first load. For web, prefer bundling the font files as assets to eliminate the latency:

```dart
// Slower on web — fetches from Google servers
GoogleFonts.inter(fontSize: 16)

// Faster on web — serve from your own assets
// Bundle fonts in assets/fonts/ and declare in pubspec.yaml
TextStyle(fontFamily: 'Inter', fontSize: 16)
```

### `RepaintBoundary` for Frequently Animating Widgets

Wrap independently animating widgets in `RepaintBoundary` to isolate them from the parent layer:

```dart
RepaintBoundary(
  child: AnimatedWidget(controller: _controller),
)
```

Flutter DevTools "Highlight repaints" diagnostic shows which areas are repainting on every frame.

---

## Performance Checklist

- [ ] Profile in `--profile` mode, not debug
- [ ] `const` on all leaf widgets
- [ ] Providers watched at the lowest possible widget level
- [ ] `ref.watch(...).select(...)` to limit rebuilds to relevant state slices
- [ ] `ListView.builder` for all lists longer than ~20 items
- [ ] `CachedNetworkImage` with `memCacheWidth`/`memCacheHeight`
- [ ] No `Opacity` in animations (use `FadeTransition`)
- [ ] No `IntrinsicHeight`/`IntrinsicWidth` in lists
- [ ] Deferred imports for large, rarely-accessed features (web)
- [ ] `RepaintBoundary` around isolated high-frequency animations
- [ ] CORS headers on CDN for web image loading
