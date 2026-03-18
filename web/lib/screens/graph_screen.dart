import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/edge.dart';
import '../services/api_service.dart';
import '../widgets/graph_view.dart';

final _graphDataProvider =
    StateNotifierProvider.family<_GraphDataNotifier, _GraphDataState, String>(
        (ref, rootId) {
  return _GraphDataNotifier(ref.watch(apiServiceProvider), rootId);
});

class _GraphDataState {
  final GraphData? data;
  final bool isLoading;
  final String? error;
  final int depth;
  final String? edgeTypeFilter;

  const _GraphDataState({
    this.data,
    this.isLoading = false,
    this.error,
    this.depth = 2,
    this.edgeTypeFilter,
  });

  _GraphDataState copyWith({
    GraphData? data,
    bool? isLoading,
    String? error,
    int? depth,
    String? edgeTypeFilter,
  }) {
    return _GraphDataState(
      data: data ?? this.data,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      depth: depth ?? this.depth,
      edgeTypeFilter: edgeTypeFilter,
    );
  }
}

class _GraphDataNotifier extends StateNotifier<_GraphDataState> {
  final ApiService _api;
  final String rootId;

  _GraphDataNotifier(this._api, this.rootId) : super(const _GraphDataState()) {
    load();
  }

  Future<void> load() async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      final data = await _api.exploreGraph(
        rootId: rootId,
        depth: state.depth,
        edgeType: state.edgeTypeFilter,
      );
      state = state.copyWith(data: data, isLoading: false);
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> setDepth(int depth) async {
    state = state.copyWith(depth: depth);
    await load();
  }

  Future<void> setEdgeTypeFilter(String? edgeType) async {
    state = state.copyWith(edgeTypeFilter: edgeType);
    await load();
  }
}

class GraphScreen extends ConsumerStatefulWidget {
  final String? rootPageId;

  const GraphScreen({super.key, this.rootPageId});

  @override
  ConsumerState<GraphScreen> createState() => _GraphScreenState();
}

class _GraphScreenState extends ConsumerState<GraphScreen> {
  String? _effectiveRootId;

  @override
  void initState() {
    super.initState();
    _effectiveRootId = widget.rootPageId;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () {
            if (context.canPop()) {
              context.pop();
            } else {
              context.go('/');
            }
          },
        ),
        title: const Row(
          children: [
            Icon(Icons.account_tree_outlined, color: Colors.white, size: 20),
            SizedBox(width: 8),
            Text('Knowledge Graph',
                style: TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold)),
          ],
        ),
      ),
      body: _effectiveRootId == null
          ? _NoRootSelected(
              onRootSelected: (id) => setState(() => _effectiveRootId = id))
          : _GraphContent(rootId: _effectiveRootId!),
    );
  }
}

class _NoRootSelected extends StatefulWidget {
  final ValueChanged<String> onRootSelected;

  const _NoRootSelected({required this.onRootSelected});

  @override
  State<_NoRootSelected> createState() => _NoRootSelectedState();
}

class _NoRootSelectedState extends State<_NoRootSelected> {
  final _ctrl = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 400),
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.account_tree_outlined,
                    size: 48, color: Color(0xFF4F46E5)),
                const SizedBox(height: 16),
                const Text('Enter a Page ID to explore its graph',
                    textAlign: TextAlign.center),
                const SizedBox(height: 16),
                TextField(
                  controller: _ctrl,
                  decoration:
                      const InputDecoration(labelText: 'Page ID (UUID)'),
                ),
                const SizedBox(height: 16),
                ElevatedButton(
                  onPressed: () {
                    if (_ctrl.text.trim().isNotEmpty) {
                      widget.onRootSelected(_ctrl.text.trim());
                    }
                  },
                  child: const Text('Explore'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _GraphContent extends ConsumerWidget {
  final String rootId;

  const _GraphContent({required this.rootId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(_graphDataProvider(rootId));

    return Column(
      children: [
        // Filter bar
        _GraphFilterBar(
          rootId: rootId,
          state: state,
        ),
        // Graph canvas
        Expanded(
          child: state.isLoading
              ? const Center(child: CircularProgressIndicator())
              : state.error != null
                  ? _GraphError(error: state.error!)
                  : state.data != null
                      ? GraphView(
                          data: state.data!,
                          rootId: rootId,
                          onNodeTap: (nodeId) =>
                              context.push('/pages/$nodeId'),
                        )
                      : const Center(child: Text('No graph data')),
        ),
        // Status bar
        if (state.data != null)
          _GraphStatusBar(
            data: state.data!,
            rootId: rootId,
          ),
      ],
    );
  }
}

class _GraphFilterBar extends ConsumerWidget {
  final String rootId;
  final _GraphDataState state;

  const _GraphFilterBar({required this.rootId, required this.state});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: Colors.grey[200]!)),
      ),
      child: Row(
        children: [
          // Edge type filter
          const Text('Edges: ', style: TextStyle(fontSize: 13)),
          DropdownButton<String?>(
            value: state.edgeTypeFilter,
            hint: const Text('All types', style: TextStyle(fontSize: 13)),
            items: [
              const DropdownMenuItem(value: null, child: Text('All types')),
              ...['reference', 'related', 'depends_on', 'derived_from']
                  .map((t) => DropdownMenuItem(value: t, child: Text(t))),
            ],
            onChanged: (v) => ref
                .read(_graphDataProvider(rootId).notifier)
                .setEdgeTypeFilter(v),
            style: const TextStyle(fontSize: 13, color: Colors.black87),
            underline: const SizedBox(),
          ),
          const SizedBox(width: 24),
          // Depth slider
          const Text('Depth: ', style: TextStyle(fontSize: 13)),
          SizedBox(
            width: 140,
            child: Slider(
              value: state.depth.toDouble(),
              min: 1,
              max: 5,
              divisions: 4,
              label: state.depth.toString(),
              onChanged: (v) => ref
                  .read(_graphDataProvider(rootId).notifier)
                  .setDepth(v.round()),
            ),
          ),
          Text('${state.depth}', style: const TextStyle(fontSize: 13)),
        ],
      ),
    );
  }
}

class _GraphStatusBar extends ConsumerWidget {
  final GraphData data;
  final String rootId;

  const _GraphStatusBar({required this.data, required this.rootId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: Colors.grey[200]!)),
      ),
      child: Row(
        children: [
          Text(
            'Nodes: ${data.totalNodes}  |  Edges: ${data.totalEdges}',
            style: const TextStyle(fontSize: 12, color: Colors.grey),
          ),
          if (data.truncated) ...[
            const SizedBox(width: 8),
            const Chip(
              label: Text('Truncated', style: TextStyle(fontSize: 11)),
              backgroundColor: Color(0xFFFFF3CD),
              padding: EdgeInsets.zero,
              visualDensity: VisualDensity.compact,
            ),
          ],
        ],
      ),
    );
  }
}

class _GraphError extends StatelessWidget {
  final String error;

  const _GraphError({required this.error});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.error_outline, color: Colors.red, size: 48),
          const SizedBox(height: 8),
          Text(error, textAlign: TextAlign.center),
        ],
      ),
    );
  }
}

