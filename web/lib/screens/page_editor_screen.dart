import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:go_router/go_router.dart';
import '../providers/page_provider.dart';
import '../providers/space_provider.dart';
import '../models/space.dart';
import '../models/page.dart';
import '../models/freshness.dart';
import '../models/tag.dart';
import '../widgets/freshness_badge.dart';
import '../widgets/search_bar.dart' as kb;
import '../services/api_service.dart';
import '../utils/quill_prosemirror.dart';

// ---------------------------------------------------------------------------
// Forest dark theme colours used throughout the editor.
// ---------------------------------------------------------------------------
const _kEditorBg = Color(0xFF161D14);
const _kEditorText = Color(0xFFD0E0C4);
const _kEditorSelection = Color(0xFF3A5030);
const _kToolbarBg = Color(0xFF1E2A1A);
const _kToolbarIcon = Color(0xFF9BBF7E);
const _kToolbarIconActive = Color(0xFF7AAB60);
const _kBorderColor = Color(0xFF2E3D28);

class PageEditorScreen extends ConsumerStatefulWidget {
  final String pageId;

  const PageEditorScreen({super.key, required this.pageId});

  @override
  ConsumerState<PageEditorScreen> createState() => _PageEditorScreenState();
}

class _PageEditorScreenState extends ConsumerState<PageEditorScreen> {
  late final TextEditingController _titleCtrl;
  late final QuillController _quillCtrl;
  final FocusNode _editorFocus = FocusNode();
  Timer? _autoSaveTimer;
  bool _sidebarOpen = true;
  bool _contentLoaded = false;

  @override
  void initState() {
    super.initState();
    _titleCtrl = TextEditingController();
    _quillCtrl = QuillController.basic();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(pageDetailProvider(widget.pageId).notifier).load();
    });
  }

  @override
  void dispose() {
    _autoSaveTimer?.cancel();
    _titleCtrl.dispose();
    _quillCtrl.dispose();
    _editorFocus.dispose();
    super.dispose();
  }

  void _onPageLoaded(PageDetail page) {
    if (_titleCtrl.text.isEmpty) {
      _titleCtrl.text = page.title;
    }
    if (!_contentLoaded) {
      _contentLoaded = true;
      final delta = page.content != null
          ? markdownToDeltaFromProseContent(page.content!)
          : Delta()
        ..insert('\n');
      _quillCtrl.document = Document.fromDelta(delta);
      // Listen for content changes to trigger auto-save.
      _quillCtrl.document.changes.listen((_) => _scheduleAutoSave());
    }
  }

  void _scheduleAutoSave() {
    _autoSaveTimer?.cancel();
    _autoSaveTimer = Timer(const Duration(seconds: 2), _save);
  }

  Future<void> _save() async {
    final title = _titleCtrl.text.trim();
    if (title.isEmpty) return;

    final delta = _quillCtrl.document.toDelta();
    final contentNodes = deltaToProseNodes(delta);

    await ref.read(pageDetailProvider(widget.pageId).notifier).save(
          title: title,
          content: {'type': 'doc', 'content': contentNodes},
        );
  }

  @override
  Widget build(BuildContext context) {
    final pageState = ref.watch(pageDetailProvider(widget.pageId));
    final page = pageState.page;

    // Sync controllers when page first loads.
    if (page != null && !_contentLoaded) {
      // Schedule after current build frame.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _onPageLoaded(page);
      });
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
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.8), fontSize: 12),
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
                quillCtrl: _quillCtrl,
                editorFocus: _editorFocus,
                sidebarOpen: _sidebarOpen,
                onSave: _save,
                onScheduleSave: _scheduleAutoSave,
                pageId: widget.pageId,
              ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Editor body — holds the title field and the Quill editor + toolbar.
// ---------------------------------------------------------------------------

class _EditorBody extends ConsumerWidget {
  final PageDetail? page;
  final PageDetailState pageState;
  final TextEditingController titleCtrl;
  final QuillController quillCtrl;
  final FocusNode editorFocus;
  final bool sidebarOpen;
  final VoidCallback onSave;
  final VoidCallback onScheduleSave;
  final String pageId;

  const _EditorBody({
    required this.page,
    required this.pageState,
    required this.titleCtrl,
    required this.quillCtrl,
    required this.editorFocus,
    required this.sidebarOpen,
    required this.onSave,
    required this.onScheduleSave,
    required this.pageId,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Row(
      children: [
        // Editor area.
        Expanded(
          child: Column(
            children: [
              // Breadcrumb.
              if (page != null) _Breadcrumb(page: page!),
              // Title field.
              Padding(
                padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
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
              // Rich text editor.
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
                  child: _QuillEditor(
                    controller: quillCtrl,
                    focusNode: editorFocus,
                  ),
                ),
              ),
            ],
          ),
        ),
        // Metadata sidebar.
        if (sidebarOpen && page != null)
          SizedBox(
            width: 280,
            child: _MetadataSidebar(page: page!, pageId: pageId),
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Quill editor widget — toolbar + editor area, forest dark themed.
// ---------------------------------------------------------------------------

class _QuillEditor extends StatelessWidget {
  final QuillController controller;
  final FocusNode focusNode;

  const _QuillEditor({
    required this.controller,
    required this.focusNode,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: _kEditorBg,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _kBorderColor),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          // Header bar.
          _EditorHeader(),
          // Toolbar.
          _buildToolbar(context),
          const Divider(height: 1, color: _kBorderColor),
          // Editor area.
          Expanded(
            child: _buildEditor(),
          ),
        ],
      ),
    );
  }

  Widget _buildToolbar(BuildContext context) {
    return Container(
      color: _kToolbarBg,
      child: QuillSimpleToolbar(
        controller: controller,
        configurations: QuillSimpleToolbarConfigurations(
          showFontFamily: false,
          showFontSize: false,
          showBackgroundColorButton: false,
          showColorButton: false,
          showAlignmentButtons: false,
          showIndent: false,
          showDirection: false,
          showSearchButton: false,
          showSubscript: false,
          showSuperscript: false,
          showUnderLineButton: false,
          showStrikeThrough: false,
          showQuote: false,
          showClipboardCut: false,
          showClipboardCopy: false,
          showClipboardPaste: false,
          showClearFormat: false,
          showSmallButton: false,
          showDividers: true,
          showBoldButton: true,
          showItalicButton: true,
          showInlineCode: true,
          showHeaderStyle: true,
          showListNumbers: true,
          showListBullets: true,
          showCodeBlock: true,
          showLink: true,
          showUndo: true,
          showRedo: true,
          toolbarIconAlignment: WrapAlignment.start,
          toolbarSectionSpacing: 4,
          iconTheme: QuillIconTheme(
            iconUnselectedColor: _kToolbarIcon,
            iconSelectedColor: _kToolbarIconActive,
            iconUnselectedFillColor: _kToolbarBg,
            iconSelectedFillColor: _kEditorSelection,
            borderRadius: 4,
          ),
          buttonOptions: QuillSimpleToolbarButtonOptions(
            base: QuillToolbarBaseButtonOptions(
              iconSize: 18,
              iconTheme: QuillIconTheme(
                iconUnselectedColor: _kToolbarIcon,
                iconSelectedColor: _kToolbarIconActive,
                iconUnselectedFillColor: _kToolbarBg,
                iconSelectedFillColor: _kEditorSelection,
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildEditor() {
    return QuillEditor(
      controller: controller,
      focusNode: focusNode,
      configurations: QuillEditorConfigurations(
        scrollController: ScrollController(),
        padding: const EdgeInsets.all(16),
        placeholder: 'Start writing…',
        expands: true,
        scrollable: true,
        autoFocus: false,
        isCheckBoxReadOnly: false,
        customStyles: DefaultStyles(
          color: _kEditorText,
          placeHolder: DefaultTextBlockStyle(
            TextStyle(
              color: _kEditorText.withOpacity(0.35),
              fontSize: 15,
              height: 1.6,
              fontFamily: 'monospace',
            ),
            HorizontalSpacing.zero,
            VerticalSpacing.zero,
            VerticalSpacing.zero,
            null,
          ),
          paragraph: DefaultTextBlockStyle(
            const TextStyle(
              color: _kEditorText,
              fontSize: 15,
              height: 1.6,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(0, 4),
            VerticalSpacing.zero,
            null,
          ),
          h1: DefaultTextBlockStyle(
            const TextStyle(
              color: _kEditorText,
              fontSize: 26,
              fontWeight: FontWeight.bold,
              height: 1.3,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(8, 4),
            VerticalSpacing.zero,
            null,
          ),
          h2: DefaultTextBlockStyle(
            const TextStyle(
              color: _kEditorText,
              fontSize: 21,
              fontWeight: FontWeight.bold,
              height: 1.35,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(6, 4),
            VerticalSpacing.zero,
            null,
          ),
          h3: DefaultTextBlockStyle(
            const TextStyle(
              color: _kEditorText,
              fontSize: 17,
              fontWeight: FontWeight.w600,
              height: 1.4,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(4, 4),
            VerticalSpacing.zero,
            null,
          ),
          bold: const TextStyle(
            color: _kEditorText,
            fontWeight: FontWeight.bold,
          ),
          italic: const TextStyle(
            color: _kEditorText,
            fontStyle: FontStyle.italic,
          ),
          small: const TextStyle(color: _kEditorText),
          inlineCode: InlineCodeStyle(
            style: const TextStyle(
              color: Color(0xFFA8D890),
              backgroundColor: Color(0xFF1F2D1A),
              fontFamily: 'monospace',
              fontSize: 13,
            ),
            header1: const TextStyle(
              color: Color(0xFFA8D890),
              backgroundColor: Color(0xFF1F2D1A),
              fontFamily: 'monospace',
              fontSize: 24,
              fontWeight: FontWeight.bold,
            ),
            header2: const TextStyle(
              color: Color(0xFFA8D890),
              backgroundColor: Color(0xFF1F2D1A),
              fontFamily: 'monospace',
              fontSize: 18,
            ),
            header3: const TextStyle(
              color: Color(0xFFA8D890),
              backgroundColor: Color(0xFF1F2D1A),
              fontFamily: 'monospace',
              fontSize: 15,
            ),
          ),
          code: DefaultTextBlockStyle(
            const TextStyle(
              color: Color(0xFFA8D890),
              backgroundColor: Color(0xFF1A2416),
              fontFamily: 'monospace',
              fontSize: 13,
              height: 1.5,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(8, 8),
            VerticalSpacing.zero,
            BoxDecoration(
              color: const Color(0xFF1A2416),
              borderRadius: BorderRadius.circular(4),
              border: Border.all(color: _kBorderColor),
            ),
          ),
          lists: DefaultListBlockStyle(
            const TextStyle(
              color: _kEditorText,
              fontSize: 15,
              height: 1.6,
            ),
            HorizontalSpacing.zero,
            const VerticalSpacing(0, 2),
            VerticalSpacing.zero,
            null,
            null,
          ),
          link: const TextStyle(
            color: Color(0xFF7EC8A0),
            decoration: TextDecoration.underline,
          ),
        ),
      ),
    );
  }
}

/// Thin header bar that sits above the toolbar — matches the forest dark style.
class _EditorHeader extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      height: 32,
      color: const Color(0xFF0F1710),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Row(
        children: [
          const Icon(Icons.terminal, size: 14, color: _kToolbarIcon),
          const SizedBox(width: 6),
          Text(
            'rich text',
            style: TextStyle(
              color: _kToolbarIcon.withOpacity(0.8),
              fontSize: 12,
              fontFamily: 'monospace',
            ),
          ),
          const Spacer(),
          // Decorative window dots.
          _dot(const Color(0xFFFF5F57)),
          const SizedBox(width: 4),
          _dot(const Color(0xFFFFBD2E)),
          const SizedBox(width: 4),
          _dot(const Color(0xFF28CA42)),
          const SizedBox(width: 2),
        ],
      ),
    );
  }

  Widget _dot(Color color) {
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

// ---------------------------------------------------------------------------
// Breadcrumb
// ---------------------------------------------------------------------------

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
                    color: Theme.of(context).colorScheme.primary,
                    fontSize: 13),
              ),
            ),
            const Text(' > ',
                style: TextStyle(color: Colors.grey, fontSize: 13)),
          ],
          Text(page.title,
              style: const TextStyle(
                  fontWeight: FontWeight.w500, fontSize: 13)),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Metadata sidebar (unchanged from original)
// ---------------------------------------------------------------------------

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
      label: Text(tag.name, style: const TextStyle(fontSize: 12)),
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
      return const Text('No freshness data',
          style: TextStyle(fontSize: 12));
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            FreshnessBadge(
                status: freshness.status,
                score: freshness.freshnessScore),
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
              padding: const EdgeInsets.symmetric(
                  horizontal: 12, vertical: 8),
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
              style:
                  const TextStyle(fontSize: 12, color: Colors.grey)),
          const Spacer(),
          Text(value,
              style: const TextStyle(
                  fontSize: 12, fontWeight: FontWeight.w500)),
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
          onPressed: () => context.push('/graph?root=$pageId'),
          icon: const Icon(Icons.account_tree_outlined, size: 14),
          label: const Text('Open Graph'),
          style: OutlinedButton.styleFrom(
            textStyle: const TextStyle(fontSize: 12),
            padding: const EdgeInsets.symmetric(
                horizontal: 12, vertical: 6),
          ),
        ),
      ],
    );
  }
}
