# Riverpod 2.x State Management — Best Practices (2024/2025)

## Package Versions (as of 2025)

```yaml
dependencies:
  flutter_riverpod: ^2.6.1
  riverpod_annotation: ^2.3.5

dev_dependencies:
  riverpod_generator: ^2.4.3
  riverpod_lint: ^2.3.13
  build_runner: ^2.4.9
```

Enable `riverpod_lint` in `analysis_options.yaml` to catch common mistakes at compile time:

```yaml
analyzer:
  plugins:
    - riverpod_lint
```

---

## 1. Provider Types — When to Use Each

Riverpod 2.x offers code-generated providers via `@riverpod` annotation. The old manual `Provider(...)` syntax still works but the annotation approach is preferred for new code.

### Quick Reference

| Scenario | Provider Type | Annotation |
|---|---|---|
| Synchronous, read-only value | `Provider` | `@riverpod` returning `T` |
| Async data (one-shot fetch) | `FutureProvider` | `@riverpod` returning `Future<T>` |
| Real-time stream | `StreamProvider` | `@riverpod` returning `Stream<T>` |
| Mutable synchronous state | `NotifierProvider` | `class extends Notifier<T>` |
| Mutable async state | `AsyncNotifierProvider` | `class extends AsyncNotifier<T>` |

### `FutureProvider` — Read-Only Async Data

Use for data that is fetched once and does not require mutation from the UI:

```dart
// features/knowledge_base/presentation/article_list_provider.dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'article_list_provider.g.dart';

@riverpod
Future<List<Article>> articleList(ArticleListRef ref) async {
  final repository = ref.watch(articleRepositoryProvider);
  return repository.getArticles();
}
```

Consume in a widget:
```dart
final articlesAsync = ref.watch(articleListProvider);

return articlesAsync.when(
  data: (articles) => ArticleListView(articles: articles),
  loading: () => const Center(child: CircularProgressIndicator()),
  error: (error, stackTrace) => ErrorView(error: error, onRetry: () => ref.invalidate(articleListProvider)),
);
```

### `AsyncNotifier` — Mutable Async State (Preferred Pattern)

Use when you need to load data AND mutate it (create, update, delete):

```dart
// features/knowledge_base/presentation/article_notifier.dart
@riverpod
class ArticleList extends _$ArticleList {
  @override
  Future<List<Article>> build() async {
    final repository = ref.watch(articleRepositoryProvider);
    return repository.getArticles();
  }

  Future<void> addArticle(CreateArticleRequest request) async {
    // Optimistic update: set loading while preserving current data
    final previous = state;
    state = const AsyncLoading<List<Article>>().copyWithPrevious(previous);

    state = await AsyncValue.guard(() async {
      final repository = ref.read(articleRepositoryProvider);
      final newArticle = await repository.createArticle(request);
      final current = previous.valueOrNull ?? [];
      return [newArticle, ...current];
    });
  }

  Future<void> deleteArticle(String id) async {
    final repository = ref.read(articleRepositoryProvider);
    await repository.deleteArticle(id);
    // Refresh from source after mutation
    ref.invalidateSelf();
  }
}
```

Note the distinction:
- `ref.watch` in `build()` — sets up reactive dependencies, provider auto-rebuilds when dependency changes
- `ref.read` in mutation methods — one-shot read without tracking (never use `ref.watch` inside non-build methods)

### `Notifier` — Synchronous State

Use for pure UI state or state that never involves async operations:

```dart
@riverpod
class SelectedTags extends _$SelectedTags {
  @override
  Set<String> build() => {};

  void toggle(String tag) {
    final current = state;
    if (current.contains(tag)) {
      state = current.difference({tag});
    } else {
      state = {...current, tag};
    }
  }

  void clear() => state = {};
}
```

---

## 2. Provider Granularity — Split vs Combine

### Rule of Thumb

**Split** providers when:
- Different parts of the UI subscribe to different parts of the state
- One piece of state changes much more frequently than another
- You want to invalidate/refresh one part independently

**Combine** (keep in one Notifier) when:
- State changes are always coordinated (you always update A when B changes)
- The data is conceptually one unit (e.g., a user's profile)
- Splitting would require complex cross-provider synchronisation

### Example: Splitting Search State

```dart
// Split: query and results are independent — results depend on query
@riverpod
class SearchQuery extends _$SearchQuery {
  @override
  String build() => '';

  void update(String query) => state = query;
}

@riverpod
Future<List<Article>> searchResults(SearchResultsRef ref) async {
  final query = ref.watch(searchQueryProvider);
  if (query.isEmpty) return [];
  // Debounce: cancel previous request
  final cancelToken = CancelToken();
  ref.onDispose(cancelToken.cancel);
  return ref.watch(articleRepositoryProvider).search(query, cancelToken: cancelToken);
}
```

### Example: Keeping Auth State Together

```dart
// Keep together: authenticated user and token are always updated atomically
@riverpod
class AuthNotifier extends _$AuthNotifier {
  @override
  AuthState build() => const AuthState.unauthenticated();

  Future<void> login(String email, String password) async {
    state = const AuthState.loading();
    state = await AsyncValue.guard(() async {
      final result = await ref.read(authServiceProvider).login(email, password);
      return AuthState.authenticated(user: result.user, token: result.token);
    }) as AuthState; // or handle the AsyncValue result
  }
}
```

---

## 3. Handling Loading / Error / Data States Correctly

### The Three-State Pattern with `.when()`

```dart
articlesAsync.when(
  data: (articles) => ArticleGrid(articles: articles),
  loading: () => const ArticleGridSkeleton(),   // Skeleton preferred over spinner
  error: (error, stackTrace) {
    // Log the stackTrace — do not swallow it
    log('Failed to load articles', error: error, stackTrace: stackTrace);
    return ErrorCard(
      message: _userMessage(error),
      onRetry: () => ref.invalidate(articleListProvider),
    );
  },
);
```

### Showing Stale Data During Refresh

When you want to keep showing old data while refreshing (pull-to-refresh pattern):

```dart
final articlesAsync = ref.watch(articleListProvider);

// `isRefreshing` is true when a refresh is happening but previous data exists
if (articlesAsync.isRefreshing) {
  return Stack(
    children: [
      ArticleGrid(articles: articlesAsync.requireValue),
      const LinearProgressIndicator(), // Non-blocking indicator
    ],
  );
}

return articlesAsync.when(
  data: (articles) => RefreshIndicator(
    onRefresh: () => ref.refresh(articleListProvider.future),
    child: ArticleGrid(articles: articles),
  ),
  loading: () => const ArticleGridSkeleton(),
  error: (e, st) => ErrorView(error: e),
);
```

### Preserving Previous Data on Error

```dart
// In AsyncNotifier — show error toast but keep existing list visible
state = await AsyncValue.guard(fetchNewData);
if (state.hasError) {
  // Restore previous data, surface error separately
  state = AsyncData(previousData);
  // Show snackbar or error banner in UI
}
```

### `AsyncValue.guard` vs Try/Catch

Always prefer `AsyncValue.guard` in Notifiers — it automatically wraps exceptions into `AsyncError` and preserves the stack trace:

```dart
// Good
state = await AsyncValue.guard(() => repository.fetch());

// Bad — loses automatic state management
try {
  final data = await repository.fetch();
  state = AsyncData(data);
} catch (e, st) {
  state = AsyncError(e, st);
}
```

---

## 4. ref.watch vs ref.read vs ref.listen

| Method | Where to Use | Triggers Rebuild? |
|---|---|---|
| `ref.watch(provider)` | Inside `build()` or `AsyncNotifier.build()` | Yes |
| `ref.read(provider)` | Inside event handlers / mutation methods | No |
| `ref.listen(provider, callback)` | Side effects (navigation, snackbars) | No |

```dart
// ref.listen — for side effects (navigation, showing dialogs)
@override
Widget build(BuildContext context, WidgetRef ref) {
  ref.listen(authStateProvider, (previous, next) {
    if (next is AuthStateUnauthenticated) {
      context.go('/login');
    }
  });
  // ...
}
```

**Critical rule**: Never call `ref.watch` inside a method body that is not `build`. This causes memory leaks because the subscription is never cleaned up.

---

## 5. Auto-Dispose and Provider Lifecycle

By default, `@riverpod` generates auto-disposing providers. The provider is destroyed when no widget is watching it.

### Keeping a Provider Alive

```dart
@Riverpod(keepAlive: true)
Future<AppConfig> appConfig(AppConfigRef ref) async {
  return AppConfigService.load();
}
```

Use `keepAlive: true` sparingly — only for app-wide singletons (config, auth, user profile). Most providers should auto-dispose.

### `ref.onDispose` for Cleanup

```dart
@riverpod
Future<List<Article>> articleStream(ArticleStreamRef ref) async {
  final subscription = someStream.listen((_) => ref.invalidateSelf());
  ref.onDispose(subscription.cancel);  // Always cancel streams/timers
  return fetchInitialData();
}
```

### Avoiding Disposed Widget Reads

Never store a `WidgetRef` in a local variable and use it after the widget is disposed:

```dart
// BAD — ref may be invalid after await
class _MyState extends ConsumerState<MyWidget> {
  Future<void> _onTap() async {
    await someAsyncOperation();
    ref.read(someProvider.notifier).doSomething(); // ref may be stale!
  }
}

// GOOD — check mounted before using context/ref after await
Future<void> _onTap() async {
  await someAsyncOperation();
  if (!mounted) return;
  ref.read(someProvider.notifier).doSomething();
}
```

---

## 6. Family Providers — Parameterised Providers

Use `family` (via parameter in the annotation) for providers that differ by an ID or key:

```dart
@riverpod
Future<Article> article(ArticleRef ref, String id) async {
  return ref.watch(articleRepositoryProvider).getById(id);
}

// Usage
final article = ref.watch(articleProvider('article-123'));
```

Each unique ID creates an independent provider instance. All are auto-disposed individually when no longer watched.

Family parameters must be comparable (implement `==` and `hashCode`). Primitives (String, int) and `freezed` classes work. Avoid using `Map` or `List` as family parameters.

---

## 7. Common Pitfalls

| Pitfall | Problem | Fix |
|---|---|---|
| `ref.watch` inside a button `onPressed` | Creates subscription that leaks | Use `ref.read` inside callbacks |
| Missing `await` on `ref.refresh` | Doesn't wait for refresh to complete | `await ref.refresh(provider.future)` |
| Mutating state from a `FutureProvider` | FutureProviders are read-only | Convert to `AsyncNotifier` |
| Calling `state = ...` inside `build()` | Causes infinite rebuild loop | State mutations belong in methods |
| No `ref.onDispose` for streams/timers | Memory leak | Always cancel in `onDispose` |
| Passing `WidgetRef` to a service/repo | Tight coupling, hard to test | Pass the data, not the ref |
| `keepAlive: true` everywhere | Defeats purpose of auto-dispose | Only for true app-wide singletons |
