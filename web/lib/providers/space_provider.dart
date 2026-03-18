import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/space.dart';
import '../services/api_service.dart';

class SpaceListState {
  final List<Space> spaces;
  final bool isLoading;
  final String? error;

  const SpaceListState({
    this.spaces = const [],
    this.isLoading = false,
    this.error,
  });

  SpaceListState copyWith({
    List<Space>? spaces,
    bool? isLoading,
    String? error,
  }) {
    return SpaceListState(
      spaces: spaces ?? this.spaces,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

class SpaceListNotifier extends StateNotifier<SpaceListState> {
  final ApiService _api;

  SpaceListNotifier(this._api) : super(const SpaceListState());

  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final spaces = await _api.getSpaces();
      state = state.copyWith(spaces: spaces, isLoading: false);
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<Space?> createSpace({
    required String name,
    String? description,
    String? icon,
  }) async {
    try {
      final space = await _api.createSpace(
        name: name,
        description: description,
        icon: icon,
      );
      state = state.copyWith(spaces: [...state.spaces, space]);
      return space;
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
      return null;
    }
  }

  Future<void> deleteSpace(String id) async {
    try {
      await _api.deleteSpace(id);
      state = state.copyWith(
        spaces: state.spaces.where((s) => s.id != id).toList(),
      );
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message);
    }
  }
}

final spaceListProvider =
    StateNotifierProvider<SpaceListNotifier, SpaceListState>((ref) {
  return SpaceListNotifier(ref.watch(apiServiceProvider));
});

final selectedSpaceIdProvider = StateProvider<String?>((ref) => null);

final selectedSpaceProvider = Provider<Space?>((ref) {
  final id = ref.watch(selectedSpaceIdProvider);
  if (id == null) return null;
  final spaces = ref.watch(spaceListProvider).spaces;
  try {
    return spaces.firstWhere((s) => s.id == id);
  } catch (_) {
    return null;
  }
});
