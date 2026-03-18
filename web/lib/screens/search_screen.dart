import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/search_provider.dart';
import '../providers/space_provider.dart';
import '../models/search_result.dart';
import '../models/freshness.dart';
import '../widgets/freshness_badge.dart';

class SearchScreen extends ConsumerStatefulWidget {
  final String? initialQuery;

  const SearchScreen({super.key, this.initialQuery});

  @override
  ConsumerState<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends ConsumerState<SearchScreen> {
  late final TextEditingController _searchCtrl;
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _searchCtrl = TextEditingController(text: widget.initialQuery ?? '');

    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(spaceListProvider.notifier).load();
      if (widget.initialQuery != null && widget.initialQuery!.isNotEmpty) {
        ref.read(searchProvider.notifier).setQuery(widget.initialQuery!);
        ref.read(searchProvider.notifier).search();
      }
    });
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchCtrl.dispose();
    super.dispose();
  }

  void _onSearchChanged(String query) {
    ref.read(searchProvider.notifier).setQuery(query);
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), () {
      if (query.trim().isNotEmpty) {
        ref.read(searchProvider.notifier).search();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final searchState = ref.watch(searchProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go('/'),
        ),
        title: const Row(
          children: [
            Icon(Icons.hub_outlined, color: Colors.white, size: 20),
            SizedBox(width: 8),
            Text('Knomantem',
                style: TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold)),
          ],
        ),
      ),
      body: Column(
        children: [
          // Search input
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border:
                  Border(bottom: BorderSide(color: Colors.grey[200]!)),
            ),
            child: TextField(
              controller: _searchCtrl,
              autofocus: true,
              decoration: InputDecoration(
                hintText: 'Search pages...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchCtrl.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchCtrl.clear();
                          ref.read(searchProvider.notifier).setQuery('');
                        },
                      )
                    : null,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                ),
                filled: true,
                fillColor: Colors.white,
              ),
              onChanged: _onSearchChanged,
            ),
          ),
          // Filter chips
          _FilterBar(),
          // Results
          Expanded(
            child: _SearchResults(
              searchState: searchState,
              onLoadMore: () =>
                  ref.read(searchProvider.notifier).loadMore(),
            ),
          ),
        ],
      ),
    );
  }
}

class _FilterBar extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final searchState = ref.watch(searchProvider);
    final filters = searchState.filters;
    final spaceState = ref.watch(spaceListProvider);

    return Container(
      height: 48,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: ListView(
        scrollDirection: Axis.horizontal,
        children: [
          // Space filter
          _FilterChip(
            label: filters.spaceId != null
                ? spaceState.spaces
                        .where((s) => s.id == filters.spaceId)
                        .map((s) => s.name)
                        .firstOrNull ??
                    'Space'
                : 'All Spaces',
            isActive: filters.spaceId != null,
            onPressed: () => _showSpaceFilter(context, ref, spaceState.spaces),
          ),
          const SizedBox(width: 8),
          // Freshness filter
          _FilterChip(
            label: filters.freshness != null
                ? filters.freshness!.capitalize()
                : 'Freshness',
            isActive: filters.freshness != null,
            onPressed: () => _showFreshnessFilter(context, ref),
          ),
          const SizedBox(width: 8),
          // Sort
          _FilterChip(
            label: 'Sort: ${filters.sort.capitalize()}',
            isActive: filters.sort != 'relevance',
            onPressed: () => _showSortFilter(context, ref),
          ),
          const SizedBox(width: 8),
          // Clear all
          if (filters.hasFilters)
            TextButton(
              onPressed: () {
                ref.read(searchProvider.notifier).clearFilters();
                ref.read(searchProvider.notifier).search();
              },
              child: const Text('Clear all'),
            ),
        ],
      ),
    );
  }

  void _showSpaceFilter(BuildContext context, WidgetRef ref, spaces) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const ListTile(title: Text('Filter by Space')),
          ListTile(
            title: const Text('All Spaces'),
            onTap: () {
              ref.read(searchProvider.notifier).setFilters(
                    ref
                        .read(searchProvider)
                        .filters
                        .copyWith(spaceId: null),
                  );
              ref.read(searchProvider.notifier).search();
              Navigator.pop(ctx);
            },
          ),
          ...spaces.map<Widget>((s) => ListTile(
                title: Text(s.name),
                onTap: () {
                  ref.read(searchProvider.notifier).setFilters(
                        ref
                            .read(searchProvider)
                            .filters
                            .copyWith(spaceId: s.id),
                      );
                  ref.read(searchProvider.notifier).search();
                  Navigator.pop(ctx);
                },
              )),
        ],
      ),
    );
  }

  void _showFreshnessFilter(BuildContext context, WidgetRef ref) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const ListTile(title: Text('Filter by Freshness')),
          ListTile(
            title: const Text('Any'),
            onTap: () {
              ref.read(searchProvider.notifier).setFilters(
                    ref.read(searchProvider).filters.copyWith(freshness: null),
                  );
              ref.read(searchProvider.notifier).search();
              Navigator.pop(ctx);
            },
          ),
          for (final status in ['fresh', 'aging', 'stale'])
            ListTile(
              leading: CircleAvatar(
                radius: 6,
                backgroundColor:
                    FreshnessStatusExtension.fromString(status).color,
              ),
              title: Text(status.capitalize()),
              onTap: () {
                ref.read(searchProvider.notifier).setFilters(
                      ref
                          .read(searchProvider)
                          .filters
                          .copyWith(freshness: status),
                    );
                ref.read(searchProvider.notifier).search();
                Navigator.pop(ctx);
              },
            ),
        ],
      ),
    );
  }

  void _showSortFilter(BuildContext context, WidgetRef ref) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const ListTile(title: Text('Sort by')),
          for (final sort in ['relevance', 'updated', 'freshness'])
            ListTile(
              title: Text(sort.capitalize()),
              onTap: () {
                ref.read(searchProvider.notifier).setFilters(
                      ref
                          .read(searchProvider)
                          .filters
                          .copyWith(sort: sort),
                    );
                ref.read(searchProvider.notifier).search();
                Navigator.pop(ctx);
              },
            ),
        ],
      ),
    );
  }
}

class _FilterChip extends StatelessWidget {
  final String label;
  final bool isActive;
  final VoidCallback onPressed;

  const _FilterChip({
    required this.label,
    required this.isActive,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: FilterChip(
        label: Text(label, style: const TextStyle(fontSize: 13)),
        selected: isActive,
        onSelected: (_) => onPressed(),
        visualDensity: VisualDensity.compact,
      ),
    );
  }
}

class _SearchResults extends StatelessWidget {
  final SearchState searchState;
  final VoidCallback onLoadMore;

  const _SearchResults({
    required this.searchState,
    required this.onLoadMore,
  });

  @override
  Widget build(BuildContext context) {
    if (searchState.query.isEmpty) {
      return const Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.search, size: 64, color: Colors.grey),
            SizedBox(height: 16),
            Text('Type to search pages...',
                style: TextStyle(color: Colors.grey, fontSize: 16)),
          ],
        ),
      );
    }

    if (searchState.isLoading && searchState.results.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (searchState.error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, color: Colors.red, size: 48),
            const SizedBox(height: 8),
            Text(searchState.error!),
          ],
        ),
      );
    }

    if (searchState.results.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.search_off, size: 64, color: Colors.grey),
            const SizedBox(height: 16),
            Text(
              'No results for "${searchState.query}"',
              style: const TextStyle(color: Colors.grey, fontSize: 16),
            ),
          ],
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: searchState.results.length + 1,
      itemBuilder: (context, index) {
        if (index < searchState.results.length) {
          return _SearchResultCard(result: searchState.results[index]);
        }
        // Load more footer
        if (searchState.hasMore) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Center(
              child: TextButton(
                onPressed: searchState.isLoading ? null : onLoadMore,
                child: searchState.isLoading
                    ? const SizedBox(
                        width: 24,
                        height: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Load more...'),
              ),
            ),
          );
        }
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Center(
            child: Text(
              '${searchState.total} result${searchState.total == 1 ? '' : 's'} '
              '(${searchState.queryTimeMs}ms)',
              style: const TextStyle(color: Colors.grey, fontSize: 12),
            ),
          ),
        );
      },
    );
  }
}

class _SearchResultCard extends StatelessWidget {
  final SearchResult result;

  const _SearchResultCard({required this.result});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () => context.push('/pages/${result.pageId}'),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Title row
              Row(
                children: [
                  const Icon(Icons.article_outlined, size: 18, color: Colors.grey),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      result.title,
                      style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        fontSize: 15,
                      ),
                    ),
                  ),
                  if (result.freshness != null)
                    FreshnessBadge(
                      status: result.freshness!.status,
                      score: result.freshness!.freshnessScore,
                    ),
                ],
              ),
              if (result.space != null) ...[
                const SizedBox(height: 4),
                InkWell(
                  onTap: () => context
                      .push('/spaces/${result.space!['id']}'),
                  child: Text(
                    result.space!['name'] as String? ?? '',
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                ),
              ],
              if (result.excerpt.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  _stripHtml(result.excerpt),
                  style: const TextStyle(fontSize: 13, height: 1.4),
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              if (result.tags.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  children: result.tags.map((tag) {
                    final color = tag.color != null
                        ? Color(int.parse(
                            tag.color!.replaceFirst('#', '0xFF')))
                        : Colors.blue;
                    return Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: color.withOpacity(0.12),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        '# ${tag.name}',
                        style: TextStyle(
                            fontSize: 11,
                            color: color.withOpacity(0.8)),
                      ),
                    );
                  }).toList(),
                ),
              ],
              const SizedBox(height: 8),
              Text(
                _formatRelative(result.updatedAt),
                style: const TextStyle(fontSize: 12, color: Colors.grey),
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _stripHtml(String html) {
    return html
        .replaceAll(RegExp(r'<mark>|</mark>'), '')
        .replaceAll(RegExp(r'<[^>]+>'), '');
  }

  String _formatRelative(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 0) return 'Updated ${diff.inDays}d ago';
    if (diff.inHours > 0) return 'Updated ${diff.inHours}h ago';
    if (diff.inMinutes > 0) return 'Updated ${diff.inMinutes}m ago';
    return 'Updated just now';
  }
}

extension StringCapitalize on String {
  String capitalize() =>
      isEmpty ? this : '${this[0].toUpperCase()}${substring(1)}';
}
