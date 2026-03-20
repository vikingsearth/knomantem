# Flutter Testing — Best Practices (2024/2025)

## Testing Stack for This Project

```yaml
dev_dependencies:
  flutter_test:
    sdk: flutter
  integration_test:
    sdk: flutter
  flutter_riverpod: ^2.6.1      # ProviderContainer for unit tests
  mockito: ^5.4.4               # Mock generation
  build_runner: ^2.4.9          # For mockito code gen
  network_image_mock: ^2.1.1    # Mock Image.network in widget tests
  alchemist: ^0.10.0            # Golden tests (alternative to matchesGoldenFile)
  # OR
  golden_toolkit: ^0.15.0       # Another golden test option
```

---

## 1. Test Types — When to Use Each

### Unit Tests

Test a single class or function in isolation. All dependencies are mocked. No Flutter framework, no widget tree.

**Use for:**
- Repository logic (caching rules, data transformation)
- `Notifier`/`AsyncNotifier` business logic in isolation
- Utility functions, validators, formatters
- Domain model methods

**Do NOT use for:**
- Anything that requires widget rendering
- Navigation logic

```dart
// test/features/knowledge_base/article_repository_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';

@GenerateMocks([ArticleService])
import 'article_repository_test.mocks.dart';

void main() {
  late MockArticleService mockService;
  late ArticleRepository repository;

  setUp(() {
    mockService = MockArticleService();
    repository = ArticleRepository(mockService);
  });

  group('ArticleRepository.getArticles', () {
    test('returns mapped domain models from service', () async {
      when(mockService.getArticles(page: 1)).thenAnswer(
        (_) async => [
          ArticleDto(id: '1', title: 'Test'),
        ],
      );

      final result = await repository.getArticles();

      expect(result, [isA<Article>()]);
      expect(result.first.id, '1');
    });

    test('caches results — service called only once on repeated calls', () async {
      when(mockService.getArticles(page: 1)).thenAnswer(
        (_) async => [ArticleDto(id: '1', title: 'Test')],
      );

      await repository.getArticles();
      await repository.getArticles();

      verify(mockService.getArticles(page: 1)).called(1); // Not 2
    });
  });
}
```

### Widget Tests

Test a single widget or small widget tree. Runs in a simulated Flutter environment — no real device needed. Faster than integration tests, provides meaningful UI-level assertions.

**Use for:**
- Screen rendering (correct text, icons, layout structure)
- User interaction flows within a screen (taps, input, scroll)
- State transitions visible in the UI (loading → data, error display)
- Form validation feedback

**Do NOT use for:**
- Cross-screen navigation (use integration tests)
- Platform-specific behaviour (camera, biometrics)

```dart
// test/features/knowledge_base/article_list_screen_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:network_image_mock/network_image_mock.dart';

void main() {
  group('ArticleListScreen', () {
    testWidgets('shows articles when loaded', (tester) async {
      await mockNetworkImagesFor(() async {
        final articles = [
          Article(id: '1', title: 'Riverpod Guide', content: '...', createdAt: DateTime.now()),
          Article(id: '2', title: 'Flutter Perf', content: '...', createdAt: DateTime.now()),
        ];

        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              // Override the provider with pre-loaded data — no network calls
              articleListProvider.overrideWith((_) async => articles),
            ],
            child: const MaterialApp(
              home: ArticleListScreen(),
            ),
          ),
        );

        // Wait for async provider to complete
        await tester.pumpAndSettle();

        expect(find.text('Riverpod Guide'), findsOneWidget);
        expect(find.text('Flutter Perf'), findsOneWidget);
      });
    });

    testWidgets('shows loading indicator while fetching', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            articleListProvider.overrideWith((_) async {
              await Future<void>.delayed(const Duration(seconds: 10));
              return [];
            }),
          ],
          child: const MaterialApp(home: ArticleListScreen()),
        ),
      );

      // pump() — single frame; pumpAndSettle() would wait forever for the delay
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('shows error and retry button on failure', (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            articleListProvider.overrideWith((_) async => throw Exception('Network error')),
          ],
          child: const MaterialApp(home: ArticleListScreen()),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text('Network error'), findsOneWidget);
      expect(find.text('Retry'), findsOneWidget);

      // Test retry interaction
      await tester.tap(find.text('Retry'));
      await tester.pump(); // Show loading state
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });
  });
}
```

### Integration Tests

Run on a real device or emulator. Test complete user flows end-to-end, including navigation, platform plugins, and real network (or staged environment).

**Use for:**
- Critical user journeys (sign-in, create article, publish)
- Cross-screen navigation flows
- Performance profiling on real hardware
- Smoke tests before release

**Do NOT use for:** Every feature — integration tests are slow and expensive to maintain. Aim for a small number of high-value flows.

```dart
// integration_test/auth_flow_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:my_app/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('Authentication flow', () {
    testWidgets('user can log in and see home screen', (tester) async {
      app.main();
      await tester.pumpAndSettle();

      // Should start on login screen
      expect(find.byKey(const ValueKey('loginScreen')), findsOneWidget);

      await tester.enterText(find.byKey(const ValueKey('emailField')), 'user@example.com');
      await tester.enterText(find.byKey(const ValueKey('passwordField')), 'password123');
      await tester.tap(find.byKey(const ValueKey('loginButton')));

      await tester.pumpAndSettle(const Duration(seconds: 5));

      expect(find.byKey(const ValueKey('homeScreen')), findsOneWidget);
    });
  });
}
```

Run integration tests:
```bash
flutter test integration_test/auth_flow_test.dart
# Or for web:
flutter drive --driver=test_driver/integration_test.dart \
  --target=integration_test/auth_flow_test.dart -d chrome
```

---

## 2. Testing Riverpod Providers

### Unit Testing a Provider with `ProviderContainer`

`ProviderContainer` is the test-only equivalent of `ProviderScope` — use it in pure unit tests:

```dart
test('articleList provider returns articles from repository', () async {
  final container = ProviderContainer(
    overrides: [
      // Replace the real repository with a mock
      articleRepositoryProvider.overrideWithValue(MockArticleRepository()),
    ],
  );
  addTearDown(container.dispose);

  // Read the future provider
  final articles = await container.read(articleListProvider.future);

  expect(articles, isNotEmpty);
});
```

### Testing `AsyncNotifier` State Transitions

```dart
test('ArticleListNotifier.addArticle updates state', () async {
  final mockRepo = MockArticleRepository();
  final initialArticles = [Article(id: '1', title: 'Existing')];
  final newArticle = Article(id: '2', title: 'New Article');

  when(mockRepo.getArticles()).thenAnswer((_) async => initialArticles);
  when(mockRepo.createArticle(any)).thenAnswer((_) async => newArticle);

  final container = ProviderContainer(
    overrides: [articleRepositoryProvider.overrideWithValue(mockRepo)],
  );
  addTearDown(container.dispose);

  // Wait for initial build
  await container.read(articleListProvider.future);
  expect(container.read(articleListProvider).valueOrNull?.length, 1);

  // Trigger mutation
  await container.read(articleListProvider.notifier).addArticle(
    CreateArticleRequest(title: 'New Article', content: '...'),
  );

  expect(container.read(articleListProvider).valueOrNull?.length, 2);
  expect(container.read(articleListProvider).valueOrNull?.first.title, 'New Article');
});
```

### Overriding Providers in Widget Tests

Use `ProviderScope` with `overrides` in widget tests to replace real implementations:

```dart
ProviderScope(
  overrides: [
    // Override a provider with a value directly
    currentUserProvider.overrideWithValue(testUser),

    // Override with a custom implementation
    articleRepositoryProvider.overrideWith((ref) => MockArticleRepository()),

    // Override an async provider to return immediately
    articleListProvider.overrideWith((_) async => testArticles),
  ],
  child: const MaterialApp(home: ArticleListScreen()),
)
```

---

## 3. Testing with Dio Mocks

Inject `Dio` through the provider system so it can be replaced in tests:

```dart
// core/network/dio_client.dart
@riverpod
Dio dio(DioRef ref) => Dio(BaseOptions(baseUrl: 'https://api.example.com'));

// In service:
@riverpod
ArticleService articleService(ArticleServiceRef ref) =>
    ArticleService(ref.watch(dioProvider));
```

In tests, use mockito to generate a `MockDio`:

```dart
@GenerateMocks([Dio])
import 'article_service_test.mocks.dart';

test('ArticleService.getArticles parses response correctly', () async {
  final mockDio = MockDio();

  when(mockDio.get('/articles', queryParameters: anyNamed('queryParameters')))
      .thenAnswer((_) async => Response(
            data: [
              {'id': '1', 'title': 'Test Article', 'content': '...', 'created_at': '2025-01-01T00:00:00Z'}
            ],
            statusCode: 200,
            requestOptions: RequestOptions(path: '/articles'),
          ));

  final service = ArticleService(mockDio);
  final articles = await service.getArticles(page: 1);

  expect(articles.first.title, 'Test Article');
});
```

Alternatively, use Dio's built-in `HttpClientAdapter` to intercept requests without mockito:

```dart
// A simple in-memory adapter for tests
class MockAdapter implements HttpClientAdapter {
  final Map<String, Response<dynamic>> responses;
  MockAdapter(this.responses);

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final body = jsonEncode(responses[options.path]?.data);
    return ResponseBody.fromString(body, 200);
  }

  @override
  void close({bool force = false}) {}
}
```

---

## 4. Golden Tests for UI Regression

Golden tests render a widget to an image and compare it against a stored baseline file. They catch unintended visual changes.

### Using `matchesGoldenFile` (built-in)

```dart
testWidgets('ArticleCard matches golden', (tester) async {
  await mockNetworkImagesFor(() async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: ArticleCard(
            article: Article(id: '1', title: 'Test', content: '...'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await expectLater(
      find.byType(ArticleCard),
      matchesGoldenFile('goldens/article_card.png'),
    );
  });
});
```

Generate/update goldens:
```bash
flutter test --update-goldens
```

### Gotchas with Golden Tests

- Golden files are platform-specific — a golden generated on macOS will not match one generated on Linux. Run goldens in CI on a single consistent OS (typically Linux).
- Fonts render differently across platforms. Load test fonts explicitly or use a consistent font in tests.
- Network images fail in golden tests — always wrap with `mockNetworkImagesFor()` from `network_image_mock`.
- Golden tests are brittle in CI if OS-level rendering changes (e.g., after Flutter version upgrades). Budget time to regenerate baselines on Flutter upgrades.

### Recommended: Use `alchemist` for Multi-Scenario Goldens

`alchemist` (^0.10.x) makes it easy to test multiple widget states in a single golden file:

```dart
goldenTest(
  'ArticleCard renders correctly',
  fileName: 'article_card',
  builder: () => GoldenTestGroup(
    children: [
      GoldenTestScenario(
        name: 'default',
        child: ArticleCard(article: testArticle),
      ),
      GoldenTestScenario(
        name: 'with long title',
        child: ArticleCard(article: testArticle.copyWith(title: 'A Very Long Article Title That Wraps')),
      ),
      GoldenTestScenario(
        name: 'loading',
        child: const ArticleCardSkeleton(),
      ),
    ],
  ),
);
```

---

## 5. Test File Organisation

```
test/
├── features/
│   ├── auth/
│   │   ├── auth_notifier_test.dart        # Unit: Notifier logic
│   │   ├── auth_repository_test.dart      # Unit: Repository
│   │   └── login_screen_test.dart         # Widget: UI
│   └── knowledge_base/
│       ├── article_repository_test.dart
│       ├── article_list_screen_test.dart
│       └── article_card_test.dart
├── core/
│   └── network/
│       └── dio_client_test.dart
└── goldens/
    ├── article_card.png
    └── login_screen.png

integration_test/
├── auth_flow_test.dart
├── article_create_flow_test.dart
└── test_driver/
    └── integration_test.dart
```

---

## 6. CI Test Strategy

```bash
# Unit + Widget tests (fast — run on every PR)
flutter test test/

# Golden tests (run on every PR; fail if images differ)
flutter test test/ --update-goldens  # Only on baseline update PRs

# Integration tests (run on main branch merge or release branch)
flutter test integration_test/
```

Add `flutter_test_config.dart` at the test root to configure global test behaviour (timeout, font loading):

```dart
// test/flutter_test_config.dart
import 'dart:async';
import 'package:flutter_test/flutter_test.dart';

Future<void> testExecutable(FutureOr<void> Function() testMain) async {
  TestWidgetsFlutterBinding.ensureInitialized();
  // Load test fonts to get consistent text rendering in goldens
  await loadAppFonts(); // Your font loading helper
  await testMain();
}
```
