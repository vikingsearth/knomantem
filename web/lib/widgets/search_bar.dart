import 'package:flutter/material.dart';

/// A compact search bar widget for the app bar.
class KnomantemSearchBar extends StatefulWidget {
  final ValueChanged<String> onSearch;
  final String? initialQuery;

  const KnomantemSearchBar({
    super.key,
    required this.onSearch,
    this.initialQuery,
  });

  @override
  State<KnomantemSearchBar> createState() => _KnomantemSearchBarState();
}

class _KnomantemSearchBarState extends State<KnomantemSearchBar> {
  bool _expanded = false;
  final _ctrl = TextEditingController();
  final _focus = FocusNode();

  @override
  void initState() {
    super.initState();
    if (widget.initialQuery != null) {
      _ctrl.text = widget.initialQuery!;
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    _focus.dispose();
    super.dispose();
  }

  void _submit() {
    if (_ctrl.text.trim().isNotEmpty) {
      widget.onSearch(_ctrl.text.trim());
    }
    setState(() => _expanded = false);
  }

  @override
  Widget build(BuildContext context) {
    if (_expanded) {
      return SizedBox(
        width: 240,
        child: TextField(
          controller: _ctrl,
          focusNode: _focus,
          autofocus: true,
          style: const TextStyle(color: Colors.white, fontSize: 14),
          cursorColor: Colors.white,
          decoration: InputDecoration(
            hintText: 'Search...',
            hintStyle: TextStyle(color: Colors.white.withOpacity(0.6)),
            prefixIcon: const Icon(Icons.search, color: Colors.white, size: 20),
            suffixIcon: IconButton(
              icon: const Icon(Icons.close, color: Colors.white, size: 18),
              onPressed: () => setState(() => _expanded = false),
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(20),
              borderSide: BorderSide(color: Colors.white.withOpacity(0.4)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(20),
              borderSide: BorderSide(color: Colors.white.withOpacity(0.4)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(20),
              borderSide: const BorderSide(color: Colors.white),
            ),
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            isDense: true,
          ),
          onSubmitted: (_) => _submit(),
        ),
      );
    }

    return IconButton(
      icon: const Icon(Icons.search, color: Colors.white),
      onPressed: () => setState(() => _expanded = true),
      tooltip: 'Search',
    );
  }
}
