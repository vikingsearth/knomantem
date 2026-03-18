import 'package:flutter/material.dart';

class EditorToolbar extends StatelessWidget {
  final VoidCallback? onBold;
  final VoidCallback? onItalic;
  final VoidCallback? onCode;
  final VoidCallback? onH1;
  final VoidCallback? onH2;
  final VoidCallback? onBulletList;
  final VoidCallback? onNumberedList;
  final VoidCallback? onBlockQuote;
  final VoidCallback? onSave;

  const EditorToolbar({
    super.key,
    this.onBold,
    this.onItalic,
    this.onCode,
    this.onH1,
    this.onH2,
    this.onBulletList,
    this.onNumberedList,
    this.onBlockQuote,
    this.onSave,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 40,
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(color: Colors.grey[200]!),
          top: BorderSide(color: Colors.grey[200]!),
        ),
        color: const Color(0xFFFAFAFA),
      ),
      child: Row(
        children: [
          const SizedBox(width: 8),
          _ToolbarButton(
            icon: Icons.format_bold,
            tooltip: 'Bold (Ctrl+B)',
            onPressed: onBold,
          ),
          _ToolbarButton(
            icon: Icons.format_italic,
            tooltip: 'Italic (Ctrl+I)',
            onPressed: onItalic,
          ),
          _ToolbarButton(
            icon: Icons.code,
            tooltip: 'Inline Code',
            onPressed: onCode,
          ),
          const _ToolbarDivider(),
          _ToolbarButtonText(
            label: 'H1',
            tooltip: 'Heading 1',
            onPressed: onH1,
          ),
          _ToolbarButtonText(
            label: 'H2',
            tooltip: 'Heading 2',
            onPressed: onH2,
          ),
          const _ToolbarDivider(),
          _ToolbarButton(
            icon: Icons.format_list_bulleted,
            tooltip: 'Bullet List',
            onPressed: onBulletList,
          ),
          _ToolbarButton(
            icon: Icons.format_list_numbered,
            tooltip: 'Numbered List',
            onPressed: onNumberedList,
          ),
          _ToolbarButton(
            icon: Icons.format_quote,
            tooltip: 'Block Quote',
            onPressed: onBlockQuote,
          ),
          const Spacer(),
          if (onSave != null)
            TextButton.icon(
              onPressed: onSave,
              icon: const Icon(Icons.save_outlined, size: 16),
              label: const Text('Save', style: TextStyle(fontSize: 13)),
              style: TextButton.styleFrom(
                foregroundColor: Theme.of(context).colorScheme.primary,
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                minimumSize: Size.zero,
              ),
            ),
          const SizedBox(width: 8),
        ],
      ),
    );
  }
}

class _ToolbarButton extends StatelessWidget {
  final IconData icon;
  final String tooltip;
  final VoidCallback? onPressed;

  const _ToolbarButton({
    required this.icon,
    required this.tooltip,
    this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
          child: Icon(
            icon,
            size: 18,
            color: onPressed != null ? Colors.grey[700] : Colors.grey[400],
          ),
        ),
      ),
    );
  }
}

class _ToolbarButtonText extends StatelessWidget {
  final String label;
  final String tooltip;
  final VoidCallback? onPressed;

  const _ToolbarButtonText({
    required this.label,
    required this.tooltip,
    this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(4),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.bold,
              color: onPressed != null ? Colors.grey[700] : Colors.grey[400],
            ),
          ),
        ),
      ),
    );
  }
}

class _ToolbarDivider extends StatelessWidget {
  const _ToolbarDivider();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 1,
      height: 20,
      margin: const EdgeInsets.symmetric(horizontal: 4),
      color: Colors.grey[300],
    );
  }
}
