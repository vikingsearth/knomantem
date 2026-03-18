import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/page.dart';
import '../providers/page_provider.dart';
import 'freshness_badge.dart';

class PageTree extends ConsumerWidget {
  final PageTreeState treeState;
  final String spaceId;
  final String? selectedPageId;
  final ValueChanged<String> onPageSelected;
  final ValueChanged<String> onPageEdit;
  final ValueChanged<String?> onNewChild;

  const PageTree({
    super.key,
    required this.treeState,
    required this.spaceId,
    this.selectedPageId,
    required this.onPageSelected,
    required this.onPageEdit,
    required this.onNewChild,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (treeState.isLoading && treeState.pages.isEmpty) {
      return const Center(
          child: CircularProgressIndicator(strokeWidth: 2));
    }

    if (treeState.error != null && treeState.pages.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, color: Colors.red),
            const SizedBox(height: 8),
            Text(treeState.error!,
                style: const TextStyle(fontSize: 12, color: Colors.red)),
          ],
        ),
      );
    }

    if (treeState.pages.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.article_outlined, size: 40, color: Colors.grey[400]),
            const SizedBox(height: 8),
            const Text('No pages yet',
                style: TextStyle(color: Colors.grey, fontSize: 13)),
          ],
        ),
      );
    }

    return ListView(
      children: treeState.roots
          .map((page) => _PageTreeNode(
                page: page,
                treeState: treeState,
                spaceId: spaceId,
                selectedPageId: selectedPageId,
                depth: 0,
                onPageSelected: onPageSelected,
                onPageEdit: onPageEdit,
                onNewChild: onNewChild,
              ))
          .toList(),
    );
  }
}

class _PageTreeNode extends ConsumerWidget {
  final PageSummary page;
  final PageTreeState treeState;
  final String spaceId;
  final String? selectedPageId;
  final int depth;
  final ValueChanged<String> onPageSelected;
  final ValueChanged<String> onPageEdit;
  final ValueChanged<String?> onNewChild;

  const _PageTreeNode({
    required this.page,
    required this.treeState,
    required this.spaceId,
    required this.selectedPageId,
    required this.depth,
    required this.onPageSelected,
    required this.onPageEdit,
    required this.onNewChild,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isExpanded = treeState.expandedIds.contains(page.id);
    final isSelected = page.id == selectedPageId;
    final children = treeState.childrenOf(page.id);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Node row
        Material(
          color: isSelected
              ? Theme.of(context).colorScheme.primary.withOpacity(0.1)
              : Colors.transparent,
          child: InkWell(
            onTap: () => onPageSelected(page.id),
            onDoubleTap: () => onPageEdit(page.id),
            child: Padding(
              padding: EdgeInsets.only(
                left: 8.0 + depth * 16.0,
                right: 4,
                top: 4,
                bottom: 4,
              ),
              child: Row(
                children: [
                  // Expand/collapse toggle
                  SizedBox(
                    width: 20,
                    child: page.hasChildren || children.isNotEmpty
                        ? GestureDetector(
                            onTap: () => ref
                                .read(
                                    pageTreeProvider(spaceId).notifier)
                                .toggleExpand(page.id),
                            child: Icon(
                              isExpanded
                                  ? Icons.expand_more
                                  : Icons.chevron_right,
                              size: 18,
                              color: Colors.grey[600],
                            ),
                          )
                        : const SizedBox.shrink(),
                  ),
                  // Page icon
                  Text(
                    page.icon ?? '📄',
                    style: const TextStyle(fontSize: 14),
                  ),
                  const SizedBox(width: 6),
                  // Title
                  Expanded(
                    child: Text(
                      page.title,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: isSelected
                            ? FontWeight.w600
                            : FontWeight.normal,
                        overflow: TextOverflow.ellipsis,
                      ),
                      maxLines: 1,
                    ),
                  ),
                  // Freshness dot
                  FreshnessBadge(status: page.freshnessStatus),
                  // Context menu
                  _PageContextMenu(
                    page: page,
                    spaceId: spaceId,
                    onEdit: () => onPageEdit(page.id),
                    onNewChild: () => onNewChild(page.id),
                  ),
                ],
              ),
            ),
          ),
        ),
        // Children
        if (isExpanded && children.isNotEmpty)
          ...children.map((child) => _PageTreeNode(
                page: child,
                treeState: treeState,
                spaceId: spaceId,
                selectedPageId: selectedPageId,
                depth: depth + 1,
                onPageSelected: onPageSelected,
                onPageEdit: onPageEdit,
                onNewChild: onNewChild,
              )),
      ],
    );
  }
}

class _PageContextMenu extends ConsumerWidget {
  final PageSummary page;
  final String spaceId;
  final VoidCallback onEdit;
  final VoidCallback onNewChild;

  const _PageContextMenu({
    required this.page,
    required this.spaceId,
    required this.onEdit,
    required this.onNewChild,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return PopupMenuButton<String>(
      icon: Icon(Icons.more_horiz, size: 16, color: Colors.grey[400]),
      iconSize: 16,
      padding: EdgeInsets.zero,
      onSelected: (value) async {
        if (value == 'edit') {
          onEdit();
        } else if (value == 'new_child') {
          onNewChild();
        } else if (value == 'delete') {
          final confirm = await showDialog<bool>(
            context: context,
            builder: (ctx) => AlertDialog(
              title: const Text('Delete Page'),
              content: Text('Delete "${page.title}"? This cannot be undone.'),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(ctx, false),
                    child: const Text('Cancel')),
                ElevatedButton(
                  onPressed: () => Navigator.pop(ctx, true),
                  style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.red),
                  child: const Text('Delete'),
                ),
              ],
            ),
          );
          if (confirm == true) {
            await ref
                .read(pageTreeProvider(spaceId).notifier)
                .deletePage(page.id);
          }
        }
      },
      itemBuilder: (_) => [
        const PopupMenuItem(
          value: 'edit',
          child: Row(children: [
            Icon(Icons.edit_outlined, size: 16),
            SizedBox(width: 8),
            Text('Edit'),
          ]),
        ),
        const PopupMenuItem(
          value: 'new_child',
          child: Row(children: [
            Icon(Icons.add, size: 16),
            SizedBox(width: 8),
            Text('New child page'),
          ]),
        ),
        const PopupMenuDivider(),
        const PopupMenuItem(
          value: 'delete',
          child: Row(children: [
            Icon(Icons.delete_outline, size: 16, color: Colors.red),
            SizedBox(width: 8),
            Text('Delete', style: TextStyle(color: Colors.red)),
          ]),
        ),
      ],
    );
  }
}
