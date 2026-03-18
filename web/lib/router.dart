import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'providers/auth_provider.dart';
import 'screens/login_screen.dart';
import 'screens/home_screen.dart';
import 'screens/space_screen.dart';
import 'screens/page_editor_screen.dart';
import 'screens/search_screen.dart';
import 'screens/graph_screen.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ref.watch(authProvider.notifier);
  final authState = ref.watch(authProvider);

  return GoRouter(
    initialLocation: '/',
    redirect: (context, state) {
      final isAuthenticated = authState.isAuthenticated;
      final isLoading = authState.isLoading;
      final isLoginRoute = state.matchedLocation == '/login';

      if (isLoading) return null;

      if (!isAuthenticated && !isLoginRoute) {
        return '/login';
      }

      if (isAuthenticated && isLoginRoute) {
        return '/';
      }

      return null;
    },
    refreshListenable: _AuthNotifierListenable(ref, authNotifier),
    routes: [
      GoRoute(
        path: '/login',
        name: 'login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/',
        name: 'home',
        builder: (context, state) => const HomeScreen(),
      ),
      GoRoute(
        path: '/spaces/:id',
        name: 'space',
        builder: (context, state) {
          final spaceId = state.pathParameters['id']!;
          return SpaceScreen(spaceId: spaceId);
        },
      ),
      GoRoute(
        path: '/pages/:id',
        name: 'page',
        builder: (context, state) {
          final pageId = state.pathParameters['id']!;
          return PageEditorScreen(pageId: pageId);
        },
      ),
      GoRoute(
        path: '/search',
        name: 'search',
        builder: (context, state) {
          final query = state.uri.queryParameters['q'];
          return SearchScreen(initialQuery: query);
        },
      ),
      GoRoute(
        path: '/graph',
        name: 'graph',
        builder: (context, state) {
          final rootId = state.uri.queryParameters['root'];
          return GraphScreen(rootPageId: rootId);
        },
      ),
    ],
  );
});

class _AuthNotifierListenable extends ChangeNotifier {
  final Ref _ref;

  // ignore: unused_element
  _AuthNotifierListenable(this._ref, AuthNotifier notifier) {
    _ref.listen(authProvider, (_, __) {
      notifyListeners();
    });
  }
}
