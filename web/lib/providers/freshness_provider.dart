import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/freshness.dart';
import '../services/api_service.dart';

class FreshnessDashboardState {
  final FreshnessSummary? summary;
  final List<FreshnessDashboardItem> pages;
  final bool isLoading;
  final String? error;

  const FreshnessDashboardState({
    this.summary,
    this.pages = const [],
    this.isLoading = false,
    this.error,
  });

  FreshnessDashboardState copyWith({
    FreshnessSummary? summary,
    List<FreshnessDashboardItem>? pages,
    bool? isLoading,
    String? error,
  }) {
    return FreshnessDashboardState(
      summary: summary ?? this.summary,
      pages: pages ?? this.pages,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

class FreshnessDashboardNotifier
    extends StateNotifier<FreshnessDashboardState> {
  final ApiService _api;

  FreshnessDashboardNotifier(this._api)
      : super(const FreshnessDashboardState());

  Future<void> load({String? status}) async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final result = await _api.getFreshnessDashboard(status: status);
      state = state.copyWith(
        summary: result.summary,
        pages: result.pages,
        isLoading: false,
      );
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> verifyPage(String pageId) async {
    try {
      await _api.verifyPage(pageId);
      // Reload dashboard after verification
      await load();
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    }
  }
}

final freshnessDashboardProvider =
    StateNotifierProvider<FreshnessDashboardNotifier, FreshnessDashboardState>(
        (ref) {
  return FreshnessDashboardNotifier(ref.watch(apiServiceProvider));
});
