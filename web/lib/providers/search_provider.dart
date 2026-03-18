import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/search_result.dart';
import '../services/api_service.dart';

class SearchFilters {
  final String? spaceId;
  final List<String> tags;
  final String? freshness;
  final String? from;
  final String? to;
  final String sort;

  const SearchFilters({
    this.spaceId,
    this.tags = const [],
    this.freshness,
    this.from,
    this.to,
    this.sort = 'relevance',
  });

  SearchFilters copyWith({
    String? spaceId,
    List<String>? tags,
    String? freshness,
    String? from,
    String? to,
    String? sort,
  }) {
    return SearchFilters(
      spaceId: spaceId ?? this.spaceId,
      tags: tags ?? this.tags,
      freshness: freshness ?? this.freshness,
      from: from ?? this.from,
      to: to ?? this.to,
      sort: sort ?? this.sort,
    );
  }

  SearchFilters clearAll() => const SearchFilters();

  bool get hasFilters =>
      spaceId != null ||
      tags.isNotEmpty ||
      freshness != null ||
      from != null ||
      to != null;
}

class SearchState {
  final String query;
  final SearchFilters filters;
  final List<SearchResult> results;
  final SearchFacets? facets;
  final bool isLoading;
  final String? error;
  final int total;
  final int queryTimeMs;
  final String? nextCursor;
  final bool hasMore;

  const SearchState({
    this.query = '',
    this.filters = const SearchFilters(),
    this.results = const [],
    this.facets,
    this.isLoading = false,
    this.error,
    this.total = 0,
    this.queryTimeMs = 0,
    this.nextCursor,
    this.hasMore = false,
  });

  SearchState copyWith({
    String? query,
    SearchFilters? filters,
    List<SearchResult>? results,
    SearchFacets? facets,
    bool? isLoading,
    String? error,
    int? total,
    int? queryTimeMs,
    String? nextCursor,
    bool? hasMore,
  }) {
    return SearchState(
      query: query ?? this.query,
      filters: filters ?? this.filters,
      results: results ?? this.results,
      facets: facets ?? this.facets,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      total: total ?? this.total,
      queryTimeMs: queryTimeMs ?? this.queryTimeMs,
      nextCursor: nextCursor,
      hasMore: hasMore ?? this.hasMore,
    );
  }
}

class SearchNotifier extends StateNotifier<SearchState> {
  final ApiService _api;

  SearchNotifier(this._api) : super(const SearchState());

  void setQuery(String q) {
    state = state.copyWith(query: q);
  }

  void setFilters(SearchFilters filters) {
    state = state.copyWith(filters: filters);
  }

  void clearFilters() {
    state = state.copyWith(filters: const SearchFilters());
  }

  Future<void> search() async {
    if (state.query.trim().isEmpty) return;
    state = state.copyWith(isLoading: true, error: null, results: []);
    try {
      final response = await _api.search(
        q: state.query,
        spaceId: state.filters.spaceId,
        tags: state.filters.tags,
        freshness: state.filters.freshness,
        from: state.filters.from,
        to: state.filters.to,
        sort: state.filters.sort,
      );
      state = state.copyWith(
        isLoading: false,
        results: response.results,
        facets: response.facets,
        total: response.total,
        queryTimeMs: response.queryTimeMs,
        nextCursor: response.nextCursor,
        hasMore: response.hasMore,
      );
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> loadMore() async {
    if (!state.hasMore || state.isLoading || state.nextCursor == null) return;
    state = state.copyWith(isLoading: true);
    try {
      final response = await _api.search(
        q: state.query,
        spaceId: state.filters.spaceId,
        tags: state.filters.tags,
        freshness: state.filters.freshness,
        cursor: state.nextCursor,
      );
      state = state.copyWith(
        isLoading: false,
        results: [...state.results, ...response.results],
        nextCursor: response.nextCursor,
        hasMore: response.hasMore,
      );
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    }
  }
}

final searchProvider =
    StateNotifierProvider<SearchNotifier, SearchState>((ref) {
  return SearchNotifier(ref.watch(apiServiceProvider));
});
