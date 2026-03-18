import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/page.dart';
import '../services/api_service.dart';

// ─── Page Tree ───────────────────────────────────────────────────────────────

class PageTreeState {
  final List<PageSummary> pages;
  final bool isLoading;
  final String? error;
  final Set<String> expandedIds;

  const PageTreeState({
    this.pages = const [],
    this.isLoading = false,
    this.error,
    this.expandedIds = const {},
  });

  PageTreeState copyWith({
    List<PageSummary>? pages,
    bool? isLoading,
    String? error,
    Set<String>? expandedIds,
  }) {
    return PageTreeState(
      pages: pages ?? this.pages,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      expandedIds: expandedIds ?? this.expandedIds,
    );
  }

  List<PageSummary> childrenOf(String? parentId) {
    return pages
        .where((p) => p.parentId == parentId)
        .toList()
      ..sort((a, b) => a.position.compareTo(b.position));
  }

  List<PageSummary> get roots => childrenOf(null);
}

class PageTreeNotifier extends StateNotifier<PageTreeState> {
  final ApiService _api;
  final String spaceId;

  PageTreeNotifier(this._api, this.spaceId) : super(const PageTreeState());

  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final pages = await _api.getPages(spaceId);
      state = state.copyWith(pages: pages, isLoading: false);
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  void toggleExpand(String id) {
    final expanded = Set<String>.from(state.expandedIds);
    if (expanded.contains(id)) {
      expanded.remove(id);
    } else {
      expanded.add(id);
    }
    state = state.copyWith(expandedIds: expanded);
  }

  Future<PageDetail?> createPage({
    required String title,
    String? parentId,
  }) async {
    try {
      final page = await _api.createPage(
        spaceId: spaceId,
        title: title,
        parentId: parentId,
      );
      await load();
      return page;
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
      return null;
    }
  }

  Future<void> deletePage(String id) async {
    try {
      await _api.deletePage(id);
      state = state.copyWith(
        pages: state.pages.where((p) => p.id != id).toList(),
      );
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    }
  }

  Future<void> movePage(String id,
      {required String? parentId, required int position}) async {
    try {
      await _api.movePage(id, parentId: parentId, position: position);
      await load();
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    }
  }
}

final pageTreeProvider = StateNotifierProvider.family<PageTreeNotifier,
    PageTreeState, String>((ref, spaceId) {
  return PageTreeNotifier(ref.watch(apiServiceProvider), spaceId);
});

// ─── Page Detail ─────────────────────────────────────────────────────────────

class PageDetailState {
  final PageDetail? page;
  final bool isLoading;
  final bool isSaving;
  final String? error;
  final DateTime? lastSaved;

  const PageDetailState({
    this.page,
    this.isLoading = false,
    this.isSaving = false,
    this.error,
    this.lastSaved,
  });

  PageDetailState copyWith({
    PageDetail? page,
    bool? isLoading,
    bool? isSaving,
    String? error,
    DateTime? lastSaved,
  }) {
    return PageDetailState(
      page: page ?? this.page,
      isLoading: isLoading ?? this.isLoading,
      isSaving: isSaving ?? this.isSaving,
      error: error,
      lastSaved: lastSaved ?? this.lastSaved,
    );
  }
}

class PageDetailNotifier extends StateNotifier<PageDetailState> {
  final ApiService _api;
  final String pageId;

  PageDetailNotifier(this._api, this.pageId) : super(const PageDetailState());

  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final page = await _api.getPage(pageId);
      state = state.copyWith(page: page, isLoading: false);
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> save({
    String? title,
    Map<String, dynamic>? content,
    String? changeSummary,
  }) async {
    state = state.copyWith(isSaving: true);
    try {
      final updated = await _api.updatePage(
        pageId,
        title: title,
        content: content,
        changeSummary: changeSummary,
      );
      state = state.copyWith(
        page: updated,
        isSaving: false,
        lastSaved: DateTime.now(),
      );
    } on ApiException catch (e) {
      state = state.copyWith(isSaving: false, error: e.message);
    }
  }

  Future<void> verify({String? notes}) async {
    try {
      final freshness = await _api.verifyPage(pageId, notes: notes);
      if (state.page != null) {
        state = state.copyWith(
          page: state.page!.copyWith(freshness: freshness),
        );
      }
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    }
  }

  void updateLocalTitle(String title) {
    if (state.page != null) {
      state = state.copyWith(page: state.page!.copyWith(title: title));
    }
  }

  void updateLocalContent(Map<String, dynamic> content) {
    if (state.page != null) {
      state = state.copyWith(page: state.page!.copyWith(content: content));
    }
  }
}

final pageDetailProvider = StateNotifierProvider.family<PageDetailNotifier,
    PageDetailState, String>((ref, pageId) {
  return PageDetailNotifier(ref.watch(apiServiceProvider), pageId);
});
