import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/page_provider.dart';
import '../providers/space_provider.dart';
import '../models/space.dart';
import '../models/page.dart';
import '../models/freshness.dart';
import '../models/tag.dart';
import '../widgets/freshness_badge.dart';
import '../widgets/editor_toolbar.dart';
import '../widgets/search_bar.dart' as kb;
import '../services/api_service.dart';

class PageEditorScreen extends ConsumerStatefulWidget {
  final String pageId;

  const PageEditorScreen({super.key, required this.pageId});

  @override
  ConsumerState<PageEditorScreen> createState() => _PageEditorScreenState();
}

class _PageEditorScreenState extends ConsumerState<PageEditorScreen> {
  late final TextEditingController _titleCtrl;
  late final TextEditingController _contentCtrl;
  Timer? _autoSaveTimer;
  bool _sidebarOpen = true;
  bool _titleEditMode = false;

  @override
  void initState() {
    super.initState();
    _titleCtrl = TextEditingController();
    _contentCtrl = TextEditingController();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(pageDetailProvider(widget.pageId).notifier).load();
    });
  }

  @override
  void dispose() {
    _autoSaveTimer?.cancel();
    _titleCtrl.dispose();
    _contentCtrl.dispose();
    super.dispose();
  }

  void _onPageLoaded(PageDetail page) {
    if (_titleCtrl.text.isEmpty) {
      _titleCtrl.text = page.title;
    }
    if (_contentCtrl.text.isEmpty) {
      _contentCtrl.text = page.contentText;
    }
  }

  void _scheduleAutoSave() {
    _autoSaveTimer?.cancel();
    _autoSaveTimer = Timer(const Duration(seconds: 2), _save);
  }

  Future<void> _save() async {
    final title = _titleCtrl.text.trim();
    if (title.isEmpty) return;

    // Build simple content from plain text
    final lines = _contentCtrl.text.split('\n');
    final contentNodes = lines.map((line) {
      if (line.startsWith('# ')) {
        return {
          'type': 'heading',
          'attrs': {'level': 1},
          'content': [
            {'type': 'text', 'text': line.substring(2)}
          ],
        };
      } else if (line.startsWith('## ')) {
        return {
          'type': 'heading',
          'attrs': {'level': 2},
          'content': [
            {'type': 'text', 'text': line.substring(3)}
          ],
        };
      } else if (line.startsWith('### ')) {
        return {
          'type': 'heading',
          'attrs': {'level': 3},
          'content': [
            {'type': 'text', 'text': line.substring(4)}
          ],
        };
      } else {
        return {
          'type': 'paragraph',
          'content': [
            {'type': 'text', 'text': line}
          ],
        };
      }
    }).toList();

    await ref.read(pageDetailProvider(widget.pageId).notifier).save(
          title: title,
          content: {'type': 'doc', 'content': contentNodes},
        );
  }

  @override
  Widget build(BuildContext context) {
    final pageState = ref.watch(pageDetailProvider(widget.pageId));
    final page = pageState.page;

    // Sync controller when page first loads
    if (page != null && _titleCtrl.text.isEmpty) {
      _onPageLoaded(page);
    }

    return KeyboardListener(
      focusNode: FocusNode(),
      onKeyEvent: (event) {
        if (event is KeyDownEvent) {
          final ctrl = HardwareKeyboard.instance.isControlPressed;
          if (ctrl && event.logicalKey == LogicalKeyboardKey.keyS) {
            _save();
          }
        }
      },
      child: Scaffold(
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
          title: Row(
            children: [
              const Icon(Icons.hub_outlined, color: Colors.white, size: 20),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  page?.title ?? 'Loading...',
                  style: const TextStyle(
                      color: Colors.white, fontWeight: FontWeight.bold),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          actions: [
            if (pageState.isSaving)
              const Padding(
                padding: EdgeInsets.only(right: 8),
                child: Center(
                  child: SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white),
                  ),
                ),
              )
            else if (pageState.lastSaved != null)
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: Center(
                  child: Text(
                    'Saved',
                    style:
                        TextStyle(color: Colors.white.withOpacity(0.8), fontSize: 12),
                  ),
                ),
              ),
            IconButton(
              icon: Icon(
                  _sidebarOpen ? Icons.chevron_right : Icons.chevron_left),
              onPressed: () =>
                  setState(() => _sidebarOpen = !_sidebarOpen),
              tooltip: 'Toggle Sidebar',
              color: Colors.white,
            ),
            kb.KnomantemSearchBar(
              onSearch: (q) =>
                  context.push('/search?q=${Uri.encodeComponent(q)}'),
            ),
            const SizedBox(width: 8),
          ],
        ),
        body: pageState.isLoading && page == null
            ? const Center(child: CircularProgressIndicator())
            : _EditorBody(
                page: page,
                pageState: pageState,
                titleCtrl: _titleCtrl,
                contentCtrl: _contentCtrl,
                sidebarOpen: _sidebarOpen,
                onSave: _save,
                onScheduleSave: _scheduleAutoSave,
                pageId: widget.pageId,
              ),
      ),
    );
  }
}

class _EditorBody extends ConsumerWidget {
  final PageDetail? page;
  final PageDetailState pageState;
  final TextEditingController titleCtrl;
  final TextEditingController contentCtrl;
  final bool sidebarOpen;
  final VoidCallback onSave;
  final VoidCallback onScheduleSave;
  final String pageId;

  const _EditorBody({
    required this.page,
    required this.pageState,
    required this.titleCtrl,
    required this.contentCtrl,
    required this.sidebarOpen,
    required this.onSave,
    required this.onScheduleSave,
    required this.pageId,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Row(
      children: [
        // Editor area
        Expanded(
          child: Column(
            children: [
              // Breadcrumb
              if (page != null) _Breadcrumb(page: page!),
              // Toolbar
              EditorToolbar(
                onBold: () {
                  // placeholder: wrap selection with **
                },
                onItalic: () {},
                onCode: () {},
                onH1: () {
                  final pos = contentCtrl.selection.baseOffset;
                  if (pos >= 0) {
                    final text = contentCtrl.text;
                    final lineStart = text.lastIndexOf('\n', pos - 1) + 1;
                    contentCtrl.text = text.substring(0, lineStart) +
                        '# ' +
                        text.substring(lineStart);
                  }
                },
                onH2: () {},
                onSave: onSave,
              ),
              // Title field
              Padding(
                padding: const EdgeInsets.fromLTRB(24, 16, 24, 0),
                child: TextField(
                  controller: titleCtrl,
                  decoration: const InputDecoration(
                    border: InputBorder.none,
                    hintText: 'Page title',
                  ),
                  style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                  onChanged: (_) => onScheduleSave(),
                ),
              ),
              // Content editor (plain text placeholder)
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
                  child: TextField(
                    controller: contentCtrl,
                    decoration: const InputDecoration(
                      border: InputBorder.none,
                      hintText:
                          'Start writing... (Markdown-style: # H1, ## H2, etc.)',
                    ),
                    maxLines: null,
                    expands: true,
                    style: const TextStyle(
                        fontSize: 15, height: 1.6, fontFamily: 'monospace'),
                    onChanged: (_) => onScheduleSave(),
                    keyboardType: TextInputType.multiline,
                  ),
                ),
              ),
            ],
          ),
        ),
        // Metadata sidebar
        if (sidebarOpen && page != null)
          SizedBox(
            width: 280,
            child: _MetadataSidebar(page: page!, pageId: pageId),
          ),
      ],
    );
  }
}

class _Breadcrumb extends ConsumerWidget {
  final PageDetail page;

  const _Breadcrumb({required this.page});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final spaceState = ref.watch(spaceListProvider);
    Space? space;
    try {
      space = spaceState.spaces.firstWhere((s) => s.id == page.spaceId);
    } catch (_) {}

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: Colors.grey[200]!)),
      ),
      child: Row(
        children: [
          if (space != null) ...[
            InkWell(
              onTap: () => context.push('/spaces/${space!.id}'),
              child: Text(
                space.name,
                style: TextStyle(
                    color: Theme.of(context).colorScheme.primary, fontSize: 13),
              ),
            ),
            const Text(' > ', style: TextStyle(color: Colors.grey, fontSize: 13)),
          ],
          Text(page.title,
              style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 13)),
        ],
      ),
    );
  }
}

class _MetadataSidebar extends ConsumerStatefulWidget {
  final PageDetail page;
  final String pageId;

  const _MetadataSidebar({required this.page, required this.pageId});

  @override
  ConsumerState<_MetadataSidebar> createState() => _MetadataSidebarState();
}

class _MetadataSidebarState extends ConsumerState<_MetadataSidebar> {
  @override
  Widget build(BuildContext context) {
    final pageState = ref.watch(pageDetailProvider(widget.pageId));
    final page = pageState.page ?? widget.page;

    return Container(
      decoration: BoxDecoration(
        border: Border(left: BorderSide(color: Colors.grey[200]!)),
        color: const Color(0xFFFAFAFF),
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _MetaSection(
              title: 'Tags',
              child: _TagsPanel(page: page, pageId: widget.pageId),
            ),
            const Divider(height: 24),
            _MetaSection(
              title: 'Freshness',
              child: _FreshnessPanel(page: page, pageId: widget.pageId),
            ),
            const Divider(height: 24),
            _MetaSection(
              title: 'Version History',
              child: _VersionPanel(page: page),
            ),
            const Divider(height: 24),
            _MetaSection(
              title: 'Graph',
              child: _GraphPanel(pageId: widget.pageId),
            ),
          ],
        ),
      ),
    );
  }
}

class _MetaSection extends StatelessWidget {
  final String title;
  final Widget child;

  const _MetaSection({required this.title, required this.child});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: Colors.grey,
            letterSpacing: 0.5,
          ),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}

class _TagsPanel extends ConsumerWidget {
  final PageDetail page;
  final String pageId;

  const _TagsPanel({required this.page, required this.pageId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        ...page.tags.map((tag) => _TagChip(tag: tag)),
        ActionChip(
          label: const Text('+ Add'),
          onPressed: () => _showAddTag(context, ref),
          visualDensity: VisualDensity.compact,
        ),
      ],
    );
  }

  void _showAddTag(BuildContext context, WidgetRef ref) {
    final ctrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add Tag'),
        content: TextField(
          controller: ctrl,
          decoration: const InputDecoration(labelText: 'Tag name'),
          autofocus: true,
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () async {
              if (ctrl.text.trim().isEmpty) return;
              Navigator.pop(ctx);
              final api = ref.read(apiServiceProvider);
              try {
                final tag = await api.createTag(ctrl.text.trim());
                await api.addTagsToPage(pageId, [
                  (tagId: tag.id, confidence: 1.0)
                ]);
                ref.read(pageDetailProvider(pageId).notifier).load();
              } catch (_) {}
            },
            child: const Text('Add'),
          ),
        ],
      ),
    );
  }
}

class _TagChip extends StatelessWidget {
  final Tag tag;

  const _TagChip({required this.tag});

  @override
  Widget build(BuildContext context) {
    final color = tag.color != null
        ? Color(int.parse(tag.color!.replaceFirst('#', '0xFF')))
        : Colors.blue;

    return Chip(
      label: Text(
        tag.name,
        style: const TextStyle(fontSize: 12),
      ),
      backgroundColor: color.withOpacity(0.15),
      side: BorderSide(color: color.withOpacity(0.4)),
      visualDensity: VisualDensity.compact,
      padding: EdgeInsets.zero,
    );
  }
}

class _FreshnessPanel extends ConsumerWidget {
  final PageDetail page;
  final String pageId;

  const _FreshnessPanel({required this.page, required this.pageId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final freshness = page.freshness;
    if (freshness == null) {
      return const Text('No freshness data', style: TextStyle(fontSize: 12));
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            FreshnessBadge(
                status: freshness.status, score: freshness.freshnessScore),
            const Spacer(),
            Text(
              '${freshness.freshnessScore.toStringAsFixed(0)}%',
              style: const TextStyle(fontWeight: FontWeight.bold),
            ),
          ],
        ),
        const SizedBox(height: 6),
        _MetaRow('Interval', '${freshness.reviewIntervalDays}d'),
        if (freshness.nextReviewAt != null)
          _MetaRow(
            'Next review',
            _formatDate(freshness.nextReviewAt!),
          ),
        const SizedBox(height: 8),
        SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: () async {
              await ref
                  .read(pageDetailProvider(pageId).notifier)
                  .verify();
            },
            icon: const Icon(Icons.verified_outlined, size: 16),
            label: const Text('Verify Now'),
            style: ElevatedButton.styleFrom(
              textStyle: const TextStyle(fontSize: 12),
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
          ),
        ),
      ],
    );
  }

  String _formatDate(DateTime dt) {
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';
  }
}

class _MetaRow extends StatelessWidget {
  final String label;
  final String value;

  const _MetaRow(this.label, this.value);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Row(
        children: [
          Text(label,
              style: const TextStyle(fontSize: 12, color: Colors.grey)),
          const Spacer(),
          Text(value,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }
}

class _VersionPanel extends StatelessWidget {
  final PageDetail page;

  const _VersionPanel({required this.page});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (page.version != null)
          Text('Current: v${page.version}',
              style: const TextStyle(fontSize: 12)),
        const SizedBox(height: 4),
        Text(
          'Updated: ${_formatRelative(page.updatedAt)}',
          style: const TextStyle(fontSize: 12, color: Colors.grey),
        ),
      ],
    );
  }

  String _formatRelative(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 0) return '${diff.inDays}d ago';
    if (diff.inHours > 0) return '${diff.inHours}h ago';
    if (diff.inMinutes > 0) return '${diff.inMinutes}m ago';
    return 'Just now';
  }
}

class _GraphPanel extends ConsumerWidget {
  final String pageId;

  const _GraphPanel({required this.pageId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('View page connections in the graph.',
            style: TextStyle(fontSize: 12, color: Colors.grey)),
        const SizedBox(height: 8),
        OutlinedButton.icon(
          onPressed: () =>
              context.push('/graph?root=$pageId'),
          icon: const Icon(Icons.account_tree_outlined, size: 14),
          label: const Text('Open Graph'),
          style: OutlinedButton.styleFrom(
            textStyle: const TextStyle(fontSize: 12),
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          ),
        ),
      ],
    );
  }
}
