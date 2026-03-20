import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:graphview/graphview.dart' as gv;

import '../models/edge.dart';

// ---------------------------------------------------------------------------
// Colour palette (forest / knowledge graph theme)
// ---------------------------------------------------------------------------
const _kBg = Color(0xFF141A12);
const _kFresh = Color(0xFF5A8A48);
const _kAging = Color(0xFFB8860B);
const _kStale = Color(0xFF8B3A3A);
const _kUnknown = Color(0xFF3A4A30);
const _kRootRing = Color(0xFF9BBF7E);
const _kEdgeRef = Color(0xFF3A5030);
const _kEdgeDepends = Color(0xFF8B6914);
const _kEdgeSuper = Color(0xFF6B3A8B);
const _kEdgeRelated = Color(0xFF4A6040);

const _kNodeRadius = 28.0;
const _kRootRadius = 36.0;
const _kMaxRenderNodes = 100;

// ---------------------------------------------------------------------------
// Public widget — named GraphView so graph_screen.dart needs no changes.
// Internally we use the `graphview` package with a `gv.` prefix.
// ---------------------------------------------------------------------------
class GraphView extends StatefulWidget {
  final GraphData data;
  final String? rootId;
  final ValueChanged<String>? onNodeTap;

  const GraphView({
    super.key,
    required this.data,
    this.rootId,
    this.onNodeTap,
  });

  @override
  State<GraphView> createState() => _GraphViewState();
}

class _GraphViewState extends State<GraphView> {
  late gv.Graph _graph;
  late gv.Algorithm _algorithm;

  // Maps from our model IDs to graphview Node objects and back.
  final Map<String, gv.Node> _nodeMap = {};
  final Map<int, String> _keyToId = {};

  // Hovered node id for tooltip.
  String? _hoveredId;

  // Whether we are rendering a truncated view (>kMaxRenderNodes).
  bool _isTruncated = false;

  @override
  void initState() {
    super.initState();
    _buildGraph();
  }

  @override
  void didUpdateWidget(GraphView old) {
    super.didUpdateWidget(old);
    if (old.data != widget.data) {
      _buildGraph();
    }
  }

  void _buildGraph() {
    _nodeMap.clear();
    _keyToId.clear();

    final allNodes = widget.data.nodes;
    _isTruncated = allNodes.length > _kMaxRenderNodes;
    final renderNodes = _isTruncated
        ? _prioritisedNodes(allNodes, widget.rootId)
        : allNodes;
    final renderIds = {for (final n in renderNodes) n.id};

    final graph = gv.Graph()..isTree = false;

    for (final node in renderNodes) {
      final gvNode = gv.Node.Id(node.id);
      _nodeMap[node.id] = gvNode;
      _keyToId[node.id.hashCode] = node.id;
    }

    for (final edge in widget.data.edges) {
      if (!renderIds.contains(edge.sourceId) ||
          !renderIds.contains(edge.targetId)) {
        continue;
      }
      final src = _nodeMap[edge.sourceId];
      final tgt = _nodeMap[edge.targetId];
      if (src != null && tgt != null) {
        graph.addEdge(src, tgt,
            paint: _edgePaint(edge.edgeType));
      }
    }

    // Add any isolated nodes (no edges yet).
    for (final node in renderNodes) {
      final gvNode = _nodeMap[node.id]!;
      if (!graph.nodes.contains(gvNode)) {
        graph.addNode(gvNode);
      }
    }

    final algorithm = gv.FruchtermanReingoldAlgorithm(iterations: 1000);

    setState(() {
      _graph = graph;
      _algorithm = algorithm;
    });
  }

  /// When we must truncate, prefer the root and its closest neighbours.
  List<GraphNode> _prioritisedNodes(List<GraphNode> all, String? rootId) {
    final sorted = List<GraphNode>.from(all)
      ..sort((a, b) {
        // Root first, then by depth, then by connection count descending.
        if (a.id == rootId) return -1;
        if (b.id == rootId) return 1;
        final depthCmp = a.depthFromRoot.compareTo(b.depthFromRoot);
        if (depthCmp != 0) return depthCmp;
        return b.connectionCount.compareTo(a.connectionCount);
      });
    return sorted.take(_kMaxRenderNodes).toList();
  }

  Paint _edgePaint(String type) {
    Color color;
    switch (type) {
      case 'depends_on':
        color = _kEdgeDepends;
        break;
      case 'supersedes':
        color = _kEdgeSuper;
        break;
      case 'related_to':
      case 'related':
        color = _kEdgeRelated;
        break;
      default:
        color = _kEdgeRef;
    }
    return Paint()
      ..color = color
      ..strokeWidth = 1.5
      ..style = PaintingStyle.stroke;
  }

  GraphNode? _nodeById(String id) {
    for (final n in widget.data.nodes) {
      if (n.id == id) return n;
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    if (widget.data.nodes.isEmpty) {
      return _EmptyState();
    }

    return Stack(
      children: [
        // Dark background fills the entire allocated space.
        Positioned.fill(
          child: ColoredBox(color: _kBg),
        ),

        // Truncation warning banner.
        if (_isTruncated)
          Positioned(
            top: 0,
            left: 0,
            right: 0,
            child: _TruncationBanner(
              rendered: _kMaxRenderNodes,
              total: widget.data.totalNodes,
            ),
          ),

        // Main graph canvas.
        Positioned.fill(
          child: RepaintBoundary(
            child: InteractiveViewer(
              constrained: false,
              boundaryMargin: const EdgeInsets.all(double.infinity),
              minScale: 0.05,
              maxScale: 4.0,
              child: gv.GraphView(
                graph: _graph,
                algorithm: _algorithm,
                paint: Paint()
                  ..color = _kEdgeRef
                  ..strokeWidth = 1.5
                  ..style = PaintingStyle.stroke,
                builder: (gv.Node gvNode) {
                  // Resolve our model node from the graphview node id.
                  final nodeId = gvNode.key?.value as String?;
                  if (nodeId == null) return const SizedBox(width: 56, height: 56);
                  final modelNode = _nodeById(nodeId);
                  if (modelNode == null) return const SizedBox(width: 56, height: 56);
                  final isRoot = nodeId == widget.rootId;
                  return _NodeWidget(
                    node: modelNode,
                    isRoot: isRoot,
                    isHovered: _hoveredId == nodeId,
                    onTap: () => widget.onNodeTap?.call(nodeId),
                    onHover: (entered) {
                      setState(() {
                        _hoveredId = entered ? nodeId : null;
                      });
                    },
                  );
                },
              ),
            ),
          ),
        ),

        // Legend — bottom-left.
        const Positioned(
          left: 16,
          bottom: 16,
          child: _GraphLegend(),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Individual node widget
// ---------------------------------------------------------------------------
class _NodeWidget extends StatelessWidget {
  final GraphNode node;
  final bool isRoot;
  final bool isHovered;
  final VoidCallback onTap;
  final ValueChanged<bool> onHover;

  const _NodeWidget({
    required this.node,
    required this.isRoot,
    required this.isHovered,
    required this.onTap,
    required this.onHover,
  });

  Color get _fillColor {
    switch (node.freshnessStatus) {
      case 'fresh':
        return _kFresh;
      case 'aging':
        return _kAging;
      case 'stale':
        return _kStale;
      default:
        return _kUnknown;
    }
  }

  @override
  Widget build(BuildContext context) {
    final radius = isRoot ? _kRootRadius : _kNodeRadius;
    final diameter = radius * 2;
    final fill = _fillColor;
    final borderColor = Color.lerp(fill, Colors.white, 0.25) ?? fill;

    // Truncate label.
    const maxChars = 20;
    final label = node.title.length > maxChars
        ? '${node.title.substring(0, maxChars)}…'
        : node.title;

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => onHover(true),
      onExit: (_) => onHover(false),
      child: GestureDetector(
        onTap: onTap,
        child: Tooltip(
          message: '${node.title}\n'
              'Status: ${node.freshnessStatus}\n'
              'Connections: ${node.connectionCount}',
          decoration: BoxDecoration(
            color: const Color(0xDD1E2A1A),
            borderRadius: BorderRadius.circular(6),
          ),
          textStyle: GoogleFonts.jetBrainsMono(
            fontSize: 11,
            color: Colors.white70,
          ),
          child: SizedBox(
            width: diameter + (isHovered ? 8 : 0),
            height: diameter + (isHovered ? 8 : 0),
            child: CustomPaint(
              painter: _NodePainter(
                radius: radius,
                fill: fill,
                border: borderColor,
                isRoot: isRoot,
                isHovered: isHovered,
                label: label,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Node CustomPainter
// ---------------------------------------------------------------------------
class _NodePainter extends CustomPainter {
  final double radius;
  final Color fill;
  final Color border;
  final bool isRoot;
  final bool isHovered;
  final String label;

  const _NodePainter({
    required this.radius,
    required this.fill,
    required this.border,
    required this.isRoot,
    required this.isHovered,
    required this.label,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);

    // Glow / shadow for root or hovered.
    if (isRoot || isHovered) {
      final glowColor =
          isRoot ? _kRootRing.withOpacity(0.35) : fill.withOpacity(0.25);
      canvas.drawCircle(
        center,
        radius + 6,
        Paint()
          ..color = glowColor
          ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 8),
      );
    }

    // Root ring (bright sage halo).
    if (isRoot) {
      canvas.drawCircle(
        center,
        radius + 4,
        Paint()
          ..color = _kRootRing
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2.5,
      );
    }

    // Fill circle — slightly transparent so edges behind are visible.
    canvas.drawCircle(
      center,
      radius,
      Paint()..color = fill.withOpacity(0.82),
    );

    // Border.
    canvas.drawCircle(
      center,
      radius,
      Paint()
        ..color = border
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.0,
    );

    // Label — JetBrains Mono, white, centred, at most 2 lines.
    final textStyle = GoogleFonts.jetBrainsMono(
      fontSize: 11,
      fontWeight: isRoot ? FontWeight.bold : FontWeight.normal,
      color: Colors.white,
    );
    final tp = TextPainter(
      text: TextSpan(text: label, style: textStyle),
      textDirection: TextDirection.ltr,
      textAlign: TextAlign.center,
      maxLines: 2,
      ellipsis: '…',
    )..layout(maxWidth: radius * 1.7);

    tp.paint(
      canvas,
      center - Offset(tp.width / 2, tp.height / 2),
    );
  }

  @override
  bool shouldRepaint(_NodePainter old) =>
      old.isRoot != isRoot ||
      old.isHovered != isHovered ||
      old.fill != fill ||
      old.label != label;
}

// ---------------------------------------------------------------------------
// Edge painter callback — graphview calls this for every edge.
// We override via the `paint` parameter on gv.GraphView, but for per-type
// styling we need to draw on top. Since graphview's built-in edge paint
// is a single Paint, we use a post-frame overlay CustomPainter for advanced
// dash patterns. For now, the per-type colors are passed via the edge paint
// set in _buildGraph(), which graphview honours per-edge.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Legend widget
// ---------------------------------------------------------------------------
class _GraphLegend extends StatelessWidget {
  const _GraphLegend();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xCC1A241A),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFF2A3A28), width: 1),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _LegendRow(color: _kRootRing, label: 'Root node', ring: true),
          const SizedBox(height: 6),
          _LegendRow(color: _kFresh, label: 'Fresh'),
          const SizedBox(height: 4),
          _LegendRow(color: _kAging, label: 'Aging'),
          const SizedBox(height: 4),
          _LegendRow(color: _kStale, label: 'Stale'),
          const SizedBox(height: 8),
          const _LegendDivider(),
          const SizedBox(height: 6),
          _EdgeLegendRow(color: _kEdgeRef, label: 'reference'),
          const SizedBox(height: 4),
          _EdgeLegendRow(color: _kEdgeDepends, label: 'depends_on', dashed: true),
          const SizedBox(height: 4),
          _EdgeLegendRow(color: _kEdgeSuper, label: 'supersedes', dotted: true),
          const SizedBox(height: 4),
          _EdgeLegendRow(color: _kEdgeRelated, label: 'related_to'),
        ],
      ),
    );
  }
}

class _LegendRow extends StatelessWidget {
  final Color color;
  final String label;
  final bool ring;

  const _LegendRow({required this.color, required this.label, this.ring = false});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 14,
          height: 14,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: ring ? Colors.transparent : color.withOpacity(0.85),
            border: Border.all(
              color: color,
              width: ring ? 2.5 : 1.5,
            ),
          ),
        ),
        const SizedBox(width: 8),
        Text(
          label,
          style: GoogleFonts.jetBrainsMono(
            fontSize: 10,
            color: Colors.white70,
          ),
        ),
      ],
    );
  }
}

class _EdgeLegendRow extends StatelessWidget {
  final Color color;
  final String label;
  final bool dashed;
  final bool dotted;

  const _EdgeLegendRow({
    required this.color,
    required this.label,
    this.dashed = false,
    this.dotted = false,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: 22,
          height: 14,
          child: CustomPaint(
            painter: _DashLinePainter(
              color: color,
              dashed: dashed,
              dotted: dotted,
            ),
          ),
        ),
        const SizedBox(width: 6),
        Text(
          label,
          style: GoogleFonts.jetBrainsMono(
            fontSize: 10,
            color: Colors.white70,
          ),
        ),
      ],
    );
  }
}

class _DashLinePainter extends CustomPainter {
  final Color color;
  final bool dashed;
  final bool dotted;

  const _DashLinePainter({
    required this.color,
    required this.dashed,
    required this.dotted,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 1.5
      ..style = PaintingStyle.stroke;

    final y = size.height / 2;

    if (!dashed && !dotted) {
      canvas.drawLine(Offset(0, y), Offset(size.width, y), paint);
      return;
    }

    final dashLen = dotted ? 1.5 : 4.0;
    final gapLen = dotted ? 3.0 : 3.0;
    double x = 0;
    while (x < size.width) {
      canvas.drawLine(
        Offset(x, y),
        Offset(math.min(x + dashLen, size.width), y),
        paint,
      );
      x += dashLen + gapLen;
    }
  }

  @override
  bool shouldRepaint(_DashLinePainter old) => false;
}

class _LegendDivider extends StatelessWidget {
  const _LegendDivider();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 1,
      width: 120,
      color: const Color(0xFF2A3A28),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------
class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: _kBg,
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Text(
            'No connections yet.\nAdd links between pages to build the graph.',
            textAlign: TextAlign.center,
            style: GoogleFonts.jetBrainsMono(
              fontSize: 14,
              color: const Color(0xFF5A7A50),
              height: 1.6,
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Truncation warning banner
// ---------------------------------------------------------------------------
class _TruncationBanner extends StatelessWidget {
  final int rendered;
  final int total;

  const _TruncationBanner({required this.rendered, required this.total});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      color: const Color(0xCC3A3000),
      child: Row(
        children: [
          const Icon(Icons.warning_amber_rounded,
              size: 14, color: Color(0xFFD4A017)),
          const SizedBox(width: 8),
          Text(
            'Graph has $total nodes. Showing the closest $rendered for performance.',
            style: GoogleFonts.jetBrainsMono(
              fontSize: 11,
              color: const Color(0xFFD4A017),
            ),
          ),
        ],
      ),
    );
  }
}
