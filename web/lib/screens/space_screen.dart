import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/space_provider.dart';
import '../providers/page_provider.dart';
import '../models/space.dart';
import '../models/page.dart';
import '../widgets/page_tree.dart';
import '../widgets/freshness_badge.dart';
import '../widgets/search_bar.dart' as kb;

class SpaceScreen extends ConsumerStatefulWidget {
  final String spaceId;

  const SpaceScreen({super.key, required this.spaceId});

  @override
  ConsumerState<SpaceScreen> createState() => _SpaceScreenState();
}

class _SpaceScreenState extends ConsumerState<SpaceScreen> {
  String? _selectedPageId;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(spaceListProvider.notifier).load();
      ref.read(pageTreeProvider(widget.spaceId).notifier).load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final spaceState = ref.watch(spaceListProvider);
    final treeState = ref.watch(pageTreeProvider(widget.spaceId));

    Space? space;
    try {
      space = spaceState.spaces.firstWhere((s) => s.id == widget.spaceId);
    } catch (_) {}

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go('/'),
        ),
        title: Row(
          children: [
            const Icon(Icons.hub_outlined, color: Colors.white, size: 20),
            const SizedBox(width: 8),
            Text(
              space?.name ?? 'Space',
              style: const TextStyle(
                  color: Colors.white, fontWeight: FontWeight.bold),
            ),
          ],
        ),
        actions: [
          kb.KnomantemSearchBar(
            onSearch: (q) =>
                context.push('/search?q=${Uri.encodeComponent(q)}'),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: Row(
        children: [
          // Left panel: page tree
          SizedBox(
            width: 260,
            child: Column(
              children: [
                Padding(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          'PAGES',
                          style: TextStyle(
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                            color: Colors.grey[600],
                          ),
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.add, size: 18),
                        tooltip: 'New Page',
                        onPressed: () =>
                            _createPage(context, ref, parentId: null),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: PageTree(
                    treeState: treeState,
                    spaceId: widget.spaceId,
                    selectedPageId: _selectedPageId,
                    onPageSelected: (pageId) {
                      setState(() => _selectedPageId = pageId);
                    },
                    onPageEdit: (pageId) => context.push('/pages/$pageId'),
                    onNewChild: (parentId) =>
                        _createPage(context, ref, parentId: parentId),
                  ),
                ),
              ],
            ),
          ),
          const VerticalDivider(width: 1),
          // Right panel: page preview
          Expanded(
            child: _selectedPageId != null
                ? _PagePreview(
                    pageId: _selectedPageId!,
                    onEdit: () => context.push('/pages/$_selectedPageId'),
                  )
                : _EmptyState(space: space),
          ),
        ],
      ),
    );
  }

  Future<void> _createPage(BuildContext context, WidgetRef ref,
      {String? parentId}) async {
    final titleCtrl = TextEditingController();
    final result = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Page'),
        content: TextField(
          controller: titleCtrl,
          decoration: const InputDecoration(labelText: 'Page Title'),
          autofocus: true,
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          ElevatedButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: const Text('Create')),
        ],
      ),
    );

    if (result == true && titleCtrl.text.trim().isNotEmpty) {
      final page = await ref
          .read(pageTreeProvider(widget.spaceId).notifier)
          .createPage(title: titleCtrl.text.trim(), parentId: parentId);
      if (page != null && mounted) {
        context.push('/pages/${page.id}');
      }
    }
  }
}

class _PagePreview extends ConsumerStatefulWidget {
  final String pageId;
  final VoidCallback onEdit;

  const _PagePreview({required this.pageId, required this.onEdit});

  @override
  ConsumerState<_PagePreview> createState() => _PagePreviewState();
}

class _PagePreviewState extends ConsumerState<_PagePreview> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(pageDetailProvider(widget.pageId).notifier).load();
    });
  }

  @override
  void didUpdateWidget(_PagePreview old) {
    super.didUpdateWidget(old);
    if (old.pageId != widget.pageId) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        ref.read(pageDetailProvider(widget.pageId).notifier).load();
      });
    }
  }

  VoidCallback get onEdit => widget.onEdit;

  @override
  Widget build(BuildContext context) {
    final pageState = ref.watch(pageDetailProvider(widget.pageId));

    if (pageState.isLoading && pageState.page == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final page = pageState.page;
    if (page == null) {
      return const Center(child: Text('Page not found'));
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Page header
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: Colors.grey[200]!)),
          ),
          child: Row(
            children: [
              if (page.icon != null)
                Text(page.icon!, style: const TextStyle(fontSize: 20)),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  page.title,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
              ),
              if (page.freshness != null)
                FreshnessBadge(
                  status: page.freshness!.status,
                  score: page.freshness!.freshnessScore,
                ),
              const SizedBox(width: 12),
              ElevatedButton.icon(
                onPressed: onEdit,
                icon: const Icon(Icons.edit, size: 16),
                label: const Text('Edit'),
              ),
            ],
          ),
        ),
        // Page content preview
        Expanded(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: _PageContentRenderer(page: page),
          ),
        ),
      ],
    );
  }
}

class _PageContentRenderer extends StatelessWidget {
  final PageDetail page;

  const _PageContentRenderer({required this.page});

  @override
  Widget build(BuildContext context) {
    final content = page.content;
    if (content == null) {
      return const Text('No content yet. Click Edit to start writing.',
          style: TextStyle(color: Colors.grey));
    }

    final nodes = content['content'] as List? ?? [];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: nodes
          .map((n) => _renderNode(context, n as Map<String, dynamic>))
          .toList(),
    );
  }

  Widget _renderNode(BuildContext context, Map<String, dynamic> node) {
    final type = node['type'] as String? ?? '';
    final children = (node['content'] as List? ?? [])
        .map((c) => _renderNode(context, c as Map<String, dynamic>))
        .toList();

    switch (type) {
      case 'heading':
        final level = (node['attrs'] as Map?)['level'] as int? ?? 1;
        final textContent = _extractText(node);
        final style = [
          Theme.of(context).textTheme.headlineMedium,
          Theme.of(context).textTheme.headlineSmall,
          Theme.of(context).textTheme.titleLarge,
        ][level.clamp(1, 3) - 1];
        return Padding(
          padding: const EdgeInsets.only(bottom: 8, top: 16),
          child: Text(textContent, style: style),
        );
      case 'paragraph':
        final textContent = _extractText(node);
        return Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Text(textContent),
        );
      case 'bullet_list':
      case 'ordered_list':
        return Padding(
          padding: const EdgeInsets.only(bottom: 8, left: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: children,
          ),
        );
      case 'list_item':
        return Padding(
          padding: const EdgeInsets.only(bottom: 4),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('• '),
              Expanded(
                child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: children),
              ),
            ],
          ),
        );
      case 'code_block':
        final textContent = _extractText(node);
        return Container(
          margin: const EdgeInsets.only(bottom: 8),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.grey[100],
            borderRadius: BorderRadius.circular(4),
            border: Border.all(color: Colors.grey[300]!),
          ),
          child: Text(
            textContent,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
          ),
        );
      case 'text':
        final text = node['text'] as String? ?? '';
        final marks = (node['marks'] as List? ?? []);
        TextStyle style = const TextStyle();
        for (final mark in marks) {
          final markType = (mark as Map)['type'] as String? ?? '';
          if (markType == 'bold') {
            style = style.copyWith(fontWeight: FontWeight.bold);
          } else if (markType == 'italic') {
            style = style.copyWith(fontStyle: FontStyle.italic);
          } else if (markType == 'code') {
            style = style.copyWith(fontFamily: 'monospace');
          }
        }
        return Text(text, style: style);
      default:
        if (children.isNotEmpty) {
          return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: children);
        }
        return const SizedBox.shrink();
    }
  }

  String _extractText(Map<String, dynamic> node) {
    final buf = StringBuffer();
    if (node['text'] != null) buf.write(node['text']);
    final content = node['content'] as List?;
    if (content != null) {
      for (final c in content) {
        buf.write(_extractText(c as Map<String, dynamic>));
      }
    }
    return buf.toString();
  }
}

class _EmptyState extends StatelessWidget {
  final Space? space;

  const _EmptyState({this.space});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.article_outlined, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 16),
          Text(
            space != null
                ? 'Select a page from "${space!.name}"'
                : 'Select a page to view',
            style: TextStyle(color: Colors.grey[600], fontSize: 16),
          ),
        ],
      ),
    );
  }
}
