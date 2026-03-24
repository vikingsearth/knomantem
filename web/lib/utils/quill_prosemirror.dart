// Adapter between Quill Delta and ProseMirror JSON content nodes.
//
// Supported constructs:
//   - Headings (level 1/2/3)
//   - Paragraphs
//   - Bold, italic, inline code marks
//   - Bullet lists, ordered lists
//   - Code blocks
//
// The strategy is pragmatic: handle the formats that actually appear in the
// seeded content. Edge cases that don't appear in practice are skipped cleanly.

import 'package:flutter_quill/flutter_quill.dart';

// ---------------------------------------------------------------------------
// Delta → Markdown
// ---------------------------------------------------------------------------

/// Convert a Quill [Delta] to a markdown string.
String deltaToMarkdown(Delta delta) {
  final buffer = StringBuffer();
  final ops = delta.toList();

  // We process ops line by line.  A "line" ends when we encounter a newline
  // character inside an insert string.  Line-level attributes live on the
  // newline op itself.
  List<Map<String, dynamic>> lineRuns = [];

  void flushLine(Map<String, dynamic>? lineAttrs) {
    final heading = lineAttrs?['header'] as int?;
    final listType = lineAttrs?['list'] as String?;
    final isCodeBlock = lineAttrs?['code-block'] == true;

    if (isCodeBlock) {
      // Accumulate code-block lines with a fence; caller handles multi-line.
      final text = lineRuns.map((r) => r['text'] as String).join();
      buffer.write(text);
      buffer.write('\n');
      lineRuns.clear();
      return;
    }

    final inlineText = _renderInlineRuns(lineRuns);

    if (heading != null) {
      buffer.write('${'#' * heading} $inlineText\n');
    } else if (listType == 'bullet') {
      buffer.write('- $inlineText\n');
    } else if (listType == 'ordered') {
      buffer.write('1. $inlineText\n');
    } else {
      buffer.write('$inlineText\n');
    }

    lineRuns.clear();
  }

  bool inCodeBlock = false;

  for (final op in ops) {
    final data = op.data;
    if (data is! String) continue;

    final attrs = op.attributes ?? {};
    final parts = data.split('\n');

    for (int i = 0; i < parts.length; i++) {
      final part = parts[i];
      final isLineEnd = i < parts.length - 1;

      if (part.isNotEmpty) {
        lineRuns.add({'text': part, 'attrs': attrs});
      }

      if (isLineEnd) {
        // The newline itself may carry line-level attributes on the *next* op
        // if this split came from inside a multi-line insert.  For ops that
        // end in \n the line attributes are on this op.
        final isCodeBlockLine = attrs['code-block'] == true;

        if (isCodeBlockLine) {
          if (!inCodeBlock) {
            buffer.write('```\n');
            inCodeBlock = true;
          }
          flushLine(attrs);
        } else {
          if (inCodeBlock) {
            buffer.write('```\n');
            inCodeBlock = false;
          }
          flushLine(attrs);
        }
      }
    }
  }

  if (inCodeBlock) {
    buffer.write('```\n');
  }

  // Flush any trailing content without a newline.
  if (lineRuns.isNotEmpty) {
    flushLine(null);
  }

  return buffer.toString();
}

String _renderInlineRuns(List<Map<String, dynamic>> runs) {
  final buf = StringBuffer();
  for (final run in runs) {
    final text = run['text'] as String;
    final attrs = run['attrs'] as Map<String, dynamic>? ?? {};
    String out = text;
    if (attrs['code'] == true) {
      out = '`$out`';
    } else {
      if (attrs['bold'] == true) out = '**$out**';
      if (attrs['italic'] == true) out = '_${out}_';
    }
    buf.write(out);
  }
  return buf.toString();
}

// ---------------------------------------------------------------------------
// Markdown → Delta
// ---------------------------------------------------------------------------

/// Convert a markdown string back to a Quill [Delta].
Delta markdownToDelta(String markdown) {
  final delta = Delta();
  final lines = markdown.split('\n');

  bool inCodeBlock = false;
  String codeLang = '';
  final codeLines = <String>[];

  for (int i = 0; i < lines.length; i++) {
    final line = lines[i];

    // Code fence open/close.
    if (line.startsWith('```')) {
      if (!inCodeBlock) {
        inCodeBlock = true;
        codeLang = line.substring(3).trim();
      } else {
        // Close fence — emit accumulated code lines.
        for (final cl in codeLines) {
          if (cl.isNotEmpty) delta.insert(cl);
          delta.insert('\n', {'code-block': true});
        }
        codeLines.clear();
        inCodeBlock = false;
        codeLang = '';
      }
      continue;
    }

    if (inCodeBlock) {
      codeLines.add(line);
      continue;
    }

    // Headings.
    if (line.startsWith('### ')) {
      _insertInline(delta, line.substring(4));
      delta.insert('\n', {'header': 3});
      continue;
    }
    if (line.startsWith('## ')) {
      _insertInline(delta, line.substring(3));
      delta.insert('\n', {'header': 2});
      continue;
    }
    if (line.startsWith('# ')) {
      _insertInline(delta, line.substring(2));
      delta.insert('\n', {'header': 1});
      continue;
    }

    // Bullet list.
    if (line.startsWith('- ') || line.startsWith('* ')) {
      _insertInline(delta, line.substring(2));
      delta.insert('\n', {'list': 'bullet'});
      continue;
    }

    // Ordered list (matches "1. ", "2. " etc).
    final orderedMatch = RegExp(r'^\d+\. (.*)').firstMatch(line);
    if (orderedMatch != null) {
      _insertInline(delta, orderedMatch.group(1)!);
      delta.insert('\n', {'list': 'ordered'});
      continue;
    }

    // Plain paragraph (skip blank lines at the end).
    if (line.isEmpty && i == lines.length - 1) continue;

    _insertInline(delta, line);
    delta.insert('\n');
  }

  return delta;
}

/// Parse a markdown inline string (bold/italic/code) and insert ops into [delta].
void _insertInline(Delta delta, String text) {
  // Simple state-machine parser for **bold**, _italic_, `code`.
  // We process left-to-right and emit one op per homogeneous run.
  int i = 0;
  final len = text.length;

  while (i < len) {
    // Inline code: `...`
    if (text[i] == '`') {
      final end = text.indexOf('`', i + 1);
      if (end != -1) {
        delta.insert(text.substring(i + 1, end), {'code': true});
        i = end + 1;
        continue;
      }
    }

    // Bold: **...**
    if (i + 1 < len && text[i] == '*' && text[i + 1] == '*') {
      final end = text.indexOf('**', i + 2);
      if (end != -1) {
        delta.insert(text.substring(i + 2, end), {'bold': true});
        i = end + 2;
        continue;
      }
    }

    // Italic: _..._
    if (text[i] == '_') {
      final end = text.indexOf('_', i + 1);
      if (end != -1) {
        delta.insert(text.substring(i + 1, end), {'italic': true});
        i = end + 1;
        continue;
      }
    }

    // Plain text: consume until next marker.
    int j = i + 1;
    while (j < len) {
      if (text[j] == '`') break;
      if (text[j] == '_') break;
      if (j + 1 < len && text[j] == '*' && text[j + 1] == '*') break;
      j++;
    }
    delta.insert(text.substring(i, j));
    i = j;
  }
}

// ---------------------------------------------------------------------------
// Delta → ProseMirror JSON nodes
// ---------------------------------------------------------------------------

/// Convert a Quill [Delta] to a list of ProseMirror content nodes.
List<Map<String, dynamic>> deltaToProseNodes(Delta delta) {
  final nodes = <Map<String, dynamic>>[];
  final ops = delta.toList();

  // Collect lines the same way deltaToMarkdown does.
  List<Map<String, dynamic>> lineRuns = [];

  // Code block accumulator: we group consecutive code-block lines into a
  // single ProseMirror code_block node.
  bool inCodeBlock = false;
  final codeLines = <String>[];

  void flushCodeBlock() {
    if (codeLines.isEmpty) return;
    nodes.add({
      'type': 'code_block',
      'content': [
        {'type': 'text', 'text': codeLines.join('\n')}
      ],
    });
    codeLines.clear();
    inCodeBlock = false;
  }

  void flushLine(Map<String, dynamic>? lineAttrs) {
    final heading = lineAttrs?['header'] as int?;
    final listType = lineAttrs?['list'] as String?;
    final isCodeBlock = lineAttrs?['code-block'] == true;

    if (isCodeBlock) {
      inCodeBlock = true;
      codeLines.add(lineRuns.map((r) => r['text'] as String).join());
      lineRuns.clear();
      return;
    }

    if (inCodeBlock) flushCodeBlock();

    final inlineNodes = _buildProseInlineNodes(lineRuns);
    lineRuns.clear();

    if (inlineNodes.isEmpty) {
      // Blank paragraph — still emit to preserve spacing.
      nodes.add({'type': 'paragraph', 'content': []});
      return;
    }

    if (heading != null) {
      nodes.add({
        'type': 'heading',
        'attrs': {'level': heading},
        'content': inlineNodes,
      });
    } else if (listType == 'bullet') {
      nodes.add({
        'type': 'bullet_list',
        'content': [
          {
            'type': 'list_item',
            'content': [
              {'type': 'paragraph', 'content': inlineNodes}
            ],
          }
        ],
      });
    } else if (listType == 'ordered') {
      nodes.add({
        'type': 'ordered_list',
        'content': [
          {
            'type': 'list_item',
            'content': [
              {'type': 'paragraph', 'content': inlineNodes}
            ],
          }
        ],
      });
    } else {
      nodes.add({'type': 'paragraph', 'content': inlineNodes});
    }
  }

  for (final op in ops) {
    final data = op.data;
    if (data is! String) continue;

    final attrs = op.attributes ?? {};
    final parts = data.split('\n');

    for (int i = 0; i < parts.length; i++) {
      final part = parts[i];
      final isLineEnd = i < parts.length - 1;

      if (part.isNotEmpty) {
        lineRuns.add({'text': part, 'attrs': attrs});
      }

      if (isLineEnd) {
        flushLine(attrs);
      }
    }
  }

  // Flush any code block that never got a closing newline.
  if (inCodeBlock) flushCodeBlock();

  // Flush any remaining inline runs.
  if (lineRuns.isNotEmpty) {
    flushLine(null);
  }

  return nodes;
}

List<Map<String, dynamic>> _buildProseInlineNodes(
    List<Map<String, dynamic>> runs) {
  final result = <Map<String, dynamic>>[];
  for (final run in runs) {
    final text = run['text'] as String;
    final attrs = run['attrs'] as Map<String, dynamic>? ?? {};

    final marks = <Map<String, dynamic>>[];
    if (attrs['bold'] == true) marks.add({'type': 'bold'});
    if (attrs['italic'] == true) marks.add({'type': 'italic'});
    if (attrs['code'] == true) marks.add({'type': 'code'});

    final node = <String, dynamic>{'type': 'text', 'text': text};
    if (marks.isNotEmpty) node['marks'] = marks;
    result.add(node);
  }
  return result;
}

// ---------------------------------------------------------------------------
// ProseMirror JSON → Delta
// ---------------------------------------------------------------------------

/// Load a ProseMirror doc JSON (the full [Map] with `type: "doc"`) into a
/// Quill [Delta] suitable for display in `QuillController`.
Delta markdownToDeltaFromProseContent(Map<String, dynamic> proseContent) {
  final delta = Delta();
  final topNodes = proseContent['content'] as List? ?? [];

  for (final node in topNodes) {
    _proseNodeToDelta(node as Map<String, dynamic>, delta);
  }

  // Quill always needs at least one trailing newline.
  if (delta.isEmpty) delta.insert('\n');

  return delta;
}

void _proseNodeToDelta(Map<String, dynamic> node, Delta delta) {
  final type = node['type'] as String? ?? '';
  final children = node['content'] as List? ?? [];

  switch (type) {
    case 'paragraph':
      if (children.isEmpty) {
        delta.insert('\n');
      } else {
        _inlineChildrenToDelta(children, delta);
        delta.insert('\n');
      }

    case 'heading':
      final level = (node['attrs'] as Map?)?['level'] as int? ?? 1;
      _inlineChildrenToDelta(children, delta);
      delta.insert('\n', {'header': level});

    case 'code_block':
      // Content is a single text node with the full code.
      final rawText = children.isEmpty
          ? ''
          : (children.first as Map)['text'] as String? ?? '';
      final lines = rawText.split('\n');
      for (final line in lines) {
        if (line.isNotEmpty) delta.insert(line);
        delta.insert('\n', {'code-block': true});
      }

    case 'bullet_list':
    case 'ordered_list':
      final listAttr = type == 'bullet_list' ? 'bullet' : 'ordered';
      for (final item in children) {
        _listItemToDelta(
            item as Map<String, dynamic>, listAttr, delta);
      }

    case 'list_item':
      // Shouldn't normally be called directly; handled inside list cases.
      for (final child in children) {
        _proseNodeToDelta(child as Map<String, dynamic>, delta);
      }

    case 'blockquote':
      for (final child in children) {
        _proseNodeToDelta(child as Map<String, dynamic>, delta);
      }

    default:
      // Unknown node type — recurse into children.
      for (final child in children) {
        _proseNodeToDelta(child as Map<String, dynamic>, delta);
      }
  }
}

void _listItemToDelta(
    Map<String, dynamic> item, String listAttr, Delta delta) {
  final children = item['content'] as List? ?? [];
  // A list_item usually wraps its content in a paragraph.
  for (final child in children) {
    final childMap = child as Map<String, dynamic>;
    final childType = childMap['type'] as String? ?? '';
    if (childType == 'paragraph') {
      final inlines = childMap['content'] as List? ?? [];
      _inlineChildrenToDelta(inlines, delta);
      delta.insert('\n', {'list': listAttr});
    } else {
      _proseNodeToDelta(childMap, delta);
    }
  }
  // If the list_item had no paragraph children, just emit a newline.
  if (children.isEmpty) {
    delta.insert('\n', {'list': listAttr});
  }
}

void _inlineChildrenToDelta(List children, Delta delta) {
  for (final child in children) {
    final node = child as Map<String, dynamic>;
    final type = node['type'] as String? ?? '';
    if (type != 'text') continue;

    final text = node['text'] as String? ?? '';
    final marks = node['marks'] as List? ?? [];

    final attrs = <String, dynamic>{};
    for (final mark in marks) {
      final markType = (mark as Map)['type'] as String? ?? '';
      switch (markType) {
        case 'bold':
          attrs['bold'] = true;
        case 'italic':
          attrs['italic'] = true;
        case 'code':
          attrs['code'] = true;
        case 'link':
          final href = (mark['attrs'] as Map?)?['href'] as String?;
          if (href != null) attrs['link'] = href;
      }
    }

    if (attrs.isEmpty) {
      delta.insert(text);
    } else {
      delta.insert(text, attrs);
    }
  }
}
