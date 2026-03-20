# Flutter Architecture — Best Practices (2024/2025)

## Overview

This document covers clean architecture patterns for Flutter 3.x, including layer definitions, folder structures, widget type selection, and code generation. The guidance here is aligned with the official Flutter Architecture Guide (MVVM + Repository pattern) and community best practices for medium-to-large apps.

---

## 1. Recommended Architecture Layers

Flutter's official recommendation follows a two-layer model that maps cleanly to MVVM:

```
┌──────────────────────────────────────┐
│            UI Layer                  │
│  Views (Widgets) + ViewModels        │
│  (Riverpod Providers / Notifiers)    │
├──────────────────────────────────────┤
│           Data Layer                 │
│  Repositories + Services (API/DB)   │
└──────────────────────────────────────┘
```

### UI Layer

**Views** are pure widget trees. They should contain:
- Layout and visual composition
- Animation logic
- Simple conditional rendering (show/hide widgets based on state)
- Routing logic (calling `context.go()`)

Views must NOT contain business logic, data fetching, or direct API calls.

**ViewModels** are Riverpod `AsyncNotifier` or `Notifier` providers. They:
- Subscribe to repositories via `ref.watch`
- Transform raw data into UI-ready models
- Expose typed commands (methods) for user interactions
- Hold transient UI state (e.g., selected tab index, form validation state)

### Data Layer

**Repositories** are the single source of truth for a data domain. They:
- Aggregate data from one or more services
- Implement caching, retry logic, and error normalisation
- Expose clean domain models (not raw API DTOs)
- Are consumed by ViewModels via `ref.watch`

**Services** are thin wrappers around external systems:
- REST endpoints (via Dio)
- Local persistence (Hive, SQLite, SharedPreferences)
- Platform channels

Services hold no state and perform no caching — they just call the API and return the raw response.

### Optional: Domain Layer (Use-Cases)

Add a domain layer **only** when:
- A ViewModel must merge data from two or more repositories
- Shared business logic would otherwise be duplicated across multiple ViewModels

Avoid use-cases by default — they add ceremony without benefit in simple features.

---

## 2. Folder Structure — Feature-First (Recommended)

Feature-first organises code by product feature, making it easy to understand and delete entire features.

```
lib/
├── core/                        # App-wide utilities
│   ├── router/
│   │   └── app_router.dart      # go_router configuration
│   ├── theme/
│   │   └── app_theme.dart
│   ├── di/
│   │   └── providers.dart       # Shared infrastructure providers
│   ├── network/
│   │   └── dio_client.dart
│   └── errors/
│       └── app_exception.dart
│
├── features/
│   ├── auth/
│   │   ├── data/
│   │   │   ├── auth_service.dart        # Dio calls
│   │   │   └── auth_repository.dart
│   │   ├── domain/
│   │   │   └── user.dart               # Freezed model
│   │   └── presentation/
│   │       ├── login_screen.dart
│   │       ├── login_notifier.dart     # AsyncNotifier
│   │       └── widgets/
│   │           └── login_form.dart
│   │
│   ├── knowledge_base/
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │
│   └── settings/
│       ├── data/
│       ├── domain/
│       └── presentation/
│
└── main.dart
```

### Layer-First (Alternative — Avoid for Large Apps)

Layer-first organises by technical layer: `lib/data/`, `lib/domain/`, `lib/presentation/`. This works for small apps but creates painful cross-cutting when adding features to a large codebase. Feature-first is preferred.

### Dependency Rules

The golden rule: **dependencies only point inward / downward**.

```
presentation → domain ← data
                ↑
            core/di
```

- `presentation` imports from `domain` and `core`
- `data` imports from `domain` and `core`
- `domain` has zero Flutter/Riverpod imports (pure Dart)
- `core` has zero feature imports

Enforce this with `import_sorter` or `dart_custom_lint`.

---

## 3. StatelessWidget vs StatefulWidget vs ConsumerWidget

### Decision Matrix

| Widget Type | Use When |
|---|---|
| `StatelessWidget` | No local state, no provider access. Pure presentation. |
| `StatefulWidget` | Purely local, ephemeral UI state (animation controller, text focus, scroll controller). No business state. |
| `ConsumerWidget` | Reads from Riverpod providers. The standard widget for most screens. |
| `ConsumerStatefulWidget` | Needs both local state (e.g., `AnimationController`) AND provider access. |
| `HookConsumerWidget` | Using `flutter_hooks` for lifecycle management (optional pattern). |

### Guidelines

Prefer `ConsumerWidget` over `StatefulWidget` for state that should survive widget rebuilds, survive navigation, or be shared between widgets. If the state only matters to a single widget for the duration of its life on screen (e.g., whether a dropdown is open), `StatefulWidget` is correct.

```dart
// Good: stateless, const-constructible
class CardTitle extends StatelessWidget {
  const CardTitle({super.key, required this.title});
  final String title;

  @override
  Widget build(BuildContext context) => Text(title);
}

// Good: ConsumerWidget for provider data
class ArticleListScreen extends ConsumerWidget {
  const ArticleListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final articlesAsync = ref.watch(articleListProvider);
    return articlesAsync.when(
      data: (articles) => ArticleList(articles: articles),
      loading: () => const CircularProgressIndicator(),
      error: (e, st) => ErrorView(error: e),
    );
  }
}

// Good: ConsumerStatefulWidget for local lifecycle + providers
class VideoPlayer extends ConsumerStatefulWidget {
  const VideoPlayer({super.key, required this.videoId});
  final String videoId;

  @override
  ConsumerState<VideoPlayer> createState() => _VideoPlayerState();
}

class _VideoPlayerState extends ConsumerState<VideoPlayer>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: const Duration(milliseconds: 300));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final playbackState = ref.watch(videoPlaybackProvider(widget.videoId));
    // ...
  }
}
```

---

## 4. Code Generation Patterns

### freezed (^2.5.x) — Immutable Data Classes

`freezed` generates immutable data classes with:
- `copyWith`
- Structural equality (`==` and `hashCode`)
- Pattern-matchable union types (sealed classes)
- JSON serialisation via `json_serializable`

**Setup in `pubspec.yaml`:**
```yaml
dependencies:
  freezed_annotation: ^2.4.4
  json_annotation: ^4.9.0

dev_dependencies:
  freezed: ^2.5.7
  json_serializable: ^6.8.0
  build_runner: ^2.4.9
```

**Domain model example:**
```dart
// features/knowledge_base/domain/article.dart
import 'package:freezed_annotation/freezed_annotation.dart';

part 'article.freezed.dart';
part 'article.g.dart';

@freezed
class Article with _$Article {
  const factory Article({
    required String id,
    required String title,
    required String content,
    required DateTime createdAt,
    @Default([]) List<String> tags,
    String? coverImageUrl,
  }) = _Article;

  factory Article.fromJson(Map<String, dynamic> json) =>
      _$ArticleFromJson(json);
}
```

**Union type (sealed class) for state:**
```dart
@freezed
class AuthState with _$AuthState {
  const factory AuthState.unauthenticated() = _Unauthenticated;
  const factory AuthState.authenticated({required User user}) = _Authenticated;
  const factory AuthState.loading() = _Loading;
}

// Usage with pattern matching:
final state = ref.watch(authStateProvider);
return switch (state) {
  AuthState.authenticated(:final user) => HomeScreen(user: user),
  AuthState.loading() => const SplashScreen(),
  AuthState.unauthenticated() => const LoginScreen(),
};
```

**Run code generation:**
```bash
dart run build_runner build --delete-conflicting-outputs
# Or watch mode during development:
dart run build_runner watch --delete-conflicting-outputs
```

### json_serializable Alone (Lighter Option)

For simple DTOs that don't need union types or rich value semantics, `json_serializable` without `freezed` is lighter:

```dart
@JsonSerializable()
class ArticleDto {
  const ArticleDto({required this.id, required this.title});

  final String id;
  final String title;

  factory ArticleDto.fromJson(Map<String, dynamic> json) =>
      _$ArticleDtoFromJson(json);

  Map<String, dynamic> toJson() => _$ArticleDtoToJson(this);
}
```

### What to Avoid

- Avoid hand-writing `==`, `hashCode`, and `copyWith` — always use `freezed` for domain models.
- Avoid using `Map<String, dynamic>` as a substitute for proper typed models in the domain layer.
- Do not put `@freezed` classes in the `presentation` layer; they belong in `domain`.
- Commit generated `.freezed.dart` and `.g.dart` files only if your CI cannot run `build_runner`. Otherwise add them to `.gitignore` and generate in CI.

---

## 5. Service Layer Example (Dio)

```dart
// core/network/dio_client.dart
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'dio_client.g.dart';

@riverpod
Dio dio(DioRef ref) {
  final dio = Dio(BaseOptions(
    baseUrl: 'https://api.example.com/v1',
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 30),
    headers: {'Content-Type': 'application/json'},
  ));
  // Add interceptors (auth, logging, retry)
  return dio;
}

// features/knowledge_base/data/article_service.dart
@riverpod
ArticleService articleService(ArticleServiceRef ref) {
  return ArticleService(ref.watch(dioProvider));
}

class ArticleService {
  const ArticleService(this._dio);
  final Dio _dio;

  Future<List<ArticleDto>> getArticles({int page = 1}) async {
    final response = await _dio.get('/articles', queryParameters: {'page': page});
    return (response.data as List)
        .map((e) => ArticleDto.fromJson(e as Map<String, dynamic>))
        .toList();
  }
}
```

---

## Key Takeaways

1. Use feature-first folder structure — features are product concepts, not technical layers.
2. Separate ViewModels (Riverpod Notifiers) from Views (Widgets) — they evolve at different rates.
3. Repositories own caching and error normalisation; Services own I/O only.
4. Use `freezed` for all domain models — immutability prevents an entire class of bugs.
5. Prefer `ConsumerWidget` over `StatefulWidget` unless state is purely ephemeral UI lifecycle state.
6. Run `build_runner watch` during development to keep generated code current.
