import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/user.dart';
import '../services/api_service.dart';

enum AuthStatus { unknown, authenticated, unauthenticated }

class AuthState {
  final AuthStatus status;
  final User? user;
  final String? error;

  const AuthState({
    required this.status,
    this.user,
    this.error,
  });

  const AuthState.unknown() : this(status: AuthStatus.unknown);
  const AuthState.unauthenticated({String? error})
      : this(status: AuthStatus.unauthenticated, error: error);
  const AuthState.authenticated(User user)
      : this(status: AuthStatus.authenticated, user: user);

  bool get isAuthenticated => status == AuthStatus.authenticated;
  bool get isLoading => status == AuthStatus.unknown;
}

class AuthNotifier extends StateNotifier<AuthState> {
  final ApiService _api;

  AuthNotifier(this._api) : super(const AuthState.unknown()) {
    _init();
  }

  Future<void> _init() async {
    final token = await _api.getAccessToken();
    if (token == null) {
      state = const AuthState.unauthenticated();
      return;
    }
    try {
      final user = await _api.getMe();
      state = AuthState.authenticated(user);
    } catch (_) {
      state = const AuthState.unauthenticated();
    }
  }

  Future<void> login(String email, String password) async {
    state = const AuthState.unknown();
    try {
      final result = await _api.login(email, password);
      await _api.saveTokens(result.accessToken, result.refreshToken);
      state = AuthState.authenticated(result.user);
    } on ApiException catch (e) {
      state = AuthState.unauthenticated(error: e.message);
    } catch (e) {
      state = AuthState.unauthenticated(error: e.toString());
    }
  }

  Future<void> register(
      String email, String displayName, String password) async {
    state = const AuthState.unknown();
    try {
      final result = await _api.register(email, displayName, password);
      await _api.saveTokens(result.accessToken, result.refreshToken);
      state = AuthState.authenticated(result.user);
    } on ApiException catch (e) {
      state = AuthState.unauthenticated(error: e.message);
    } catch (e) {
      state = AuthState.unauthenticated(error: e.toString());
    }
  }

  Future<void> logout() async {
    await _api.clearTokens();
    state = const AuthState.unauthenticated();
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref.watch(apiServiceProvider));
});
