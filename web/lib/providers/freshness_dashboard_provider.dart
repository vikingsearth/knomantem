import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/freshness.dart';
import '../services/api_service.dart';

// ─── Filter / Sort Enums ──────────────────────────────────────────────────────

enum FreshnessDashboardFilter { all, fresh, aging, stale }

extension FreshnessDashboardFilterExtension on FreshnessDashboardFilter {
  String get label {
    switch (this) {
      case FreshnessDashboardFilter.all:
        return 'All';
      case FreshnessDashboardFilter.fresh:
        return 'Fresh';
      case FreshnessDashboardFilter.aging:
        return 'Aging';
      case FreshnessDashboardFilter.stale:
        return 'Stale';
    }
  }

  /// API-compatible status string, or null for "all".
  String? get apiStatus {
    switch (this) {
      case FreshnessDashboardFilter.all:
        return null;
      case FreshnessDashboardFilter.fresh:
        return 'fresh';
      case FreshnessDashboardFilter.aging:
        return 'aging';
      case FreshnessDashboardFilter.stale:
        return 'stale';
    }
  }
}

enum FreshnessDashboardSort { mostStale, leastStale, alphabetical }

extension FreshnessDashboardSortExtension on FreshnessDashboardSort {
  String get label {
    switch (this) {
      case FreshnessDashboardSort.mostStale:
        return 'Most stale first';
      case FreshnessDashboardSort.leastStale:
        return 'Least stale first';
      case FreshnessDashboardSort.alphabetical:
        return 'Alphabetical';
    }
  }

  /// API sort parameter value.
  String get apiSort {
    switch (this) {
      case FreshnessDashboardSort.mostStale:
        return 'score';
      case FreshnessDashboardSort.leastStale:
        return 'score_desc';
      case FreshnessDashboardSort.alphabetical:
        return 'title';
    }
  }
}

// ─── State ────────────────────────────────────────────────────────────────────

class FreshnessDashboardViewState {
  final FreshnessSummary? summary;
  final List<FreshnessDashboardItem> pages;
  final bool isLoading;
  final String? error;
  final FreshnessDashboardFilter filter;
  final FreshnessDashboardSort sort;

  /// Tracks which page IDs are currently being verified (shows spinner on row).
  final Set<String> verifyingIds;

  const FreshnessDashboardViewState({
    this.summary,
    this.pages = const [],
    this.isLoading = false,
    this.error,
    this.filter = FreshnessDashboardFilter.all,
    this.sort = FreshnessDashboardSort.mostStale,
    this.verifyingIds = const {},
  });

  FreshnessDashboardViewState copyWith({
    FreshnessSummary? summary,
    List<FreshnessDashboardItem>? pages,
    bool? isLoading,
    String? error,
    FreshnessDashboardFilter? filter,
    FreshnessDashboardSort? sort,
    Set<String>? verifyingIds,
    bool clearError = false,
  }) {
    return FreshnessDashboardViewState(
      summary: summary ?? this.summary,
      pages: pages ?? this.pages,
      isLoading: isLoading ?? this.isLoading,
      error: clearError ? null : (error ?? this.error),
      filter: filter ?? this.filter,
      sort: sort ?? this.sort,
      verifyingIds: verifyingIds ?? this.verifyingIds,
    );
  }
}

// ─── Notifier ────────────────────────────────────────────────────────────────

class FreshnessDashboardViewNotifier
    extends StateNotifier<FreshnessDashboardViewState> {
  final ApiService _api;

  FreshnessDashboardViewNotifier(this._api)
      : super(const FreshnessDashboardViewState());

  /// Load (or reload) data from the API using the current filter/sort.
  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final result = await _api.getFreshnessDashboard(
        status: state.filter.apiStatus,
        sort: state.sort.apiSort,
      );
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

  /// Change active status filter and reload.
  Future<void> setFilter(FreshnessDashboardFilter filter) async {
    if (state.filter == filter) return;
    state = state.copyWith(filter: filter);
    await load();
  }

  /// Change sort order and reload.
  Future<void> setSort(FreshnessDashboardSort sort) async {
    if (state.sort == sort) return;
    state = state.copyWith(sort: sort);
    await load();
  }

  /// Mark a single page as verified; refreshes the dashboard on success.
  Future<void> verifyPage(String pageId) async {
    final updated = Set<String>.from(state.verifyingIds)..add(pageId);
    state = state.copyWith(verifyingIds: updated);
    try {
      await _api.verifyPage(pageId);
      // Reload to get updated scores/summary from the server.
      await load();
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    } catch (e) {
      state = state.copyWith(error: e.toString());
    } finally {
      final done = Set<String>.from(state.verifyingIds)..remove(pageId);
      state = state.copyWith(verifyingIds: done);
    }
  }
}

// ─── Provider ────────────────────────────────────────────────────────────────

final freshnessDashboardViewProvider = StateNotifierProvider<
    FreshnessDashboardViewNotifier, FreshnessDashboardViewState>((ref) {
  return FreshnessDashboardViewNotifier(ref.watch(apiServiceProvider));
});
