import 'dart:math' as math;
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import '../models/edge.dart';
import '../models/freshness.dart';

/// A force-directed graph visualization using CustomPainter.
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

class _GraphViewState extends State<GraphView>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late List<_NodeState> _nodes;
  late List<GraphEdge> _edges;

  // Viewport transform
  double _scale = 1.0;
  Offset _pan = Offset.zero;

  // Dragging state
  String? _draggingNodeId;
  Offset? _dragStart;
  Offset? _dragNodeStart;

  // Tooltip
  String? _hoveredNodeId;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 30),
    );
    _initGraph();
    _controller.addListener(_tickPhysics);
    _controller.forward();
  }

  @override
  void didUpdateWidget(GraphView old) {
    super.didUpdateWidget(old);
    if (old.data != widget.data) {
      _initGraph();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _initGraph() {
    final rng = math.Random(42);
    _nodes = widget.data.nodes.map((n) {
      final isRoot = n.id == widget.rootId;
      return _NodeState(
        node: n,
        x: rng.nextDouble() * 600 - 300,
        y: rng.nextDouble() * 400 - 200,
        vx: 0,
        vy: 0,
        isRoot: isRoot,
      );
    }).toList();
    _edges = widget.data.edges;

    // Center root
    if (widget.rootId != null) {
      final rootIdx = _nodes.indexWhere((n) => n.node.id == widget.rootId);
      if (rootIdx >= 0) {
        _nodes[rootIdx] = _nodes[rootIdx].copyWith(x: 0, y: 0);
      }
    }
  }

  void _tickPhysics() {
    if (_draggingNodeId != null) return;

    const repulsion = 8000.0;
    const springK = 0.08;
    const restLen = 120.0;
    const damping = 0.85;

    final n = _nodes.length;
    List<double> fx = List.filled(n, 0);
    List<double> fy = List.filled(n, 0);

    // Repulsion between all node pairs
    for (int i = 0; i < n; i++) {
      for (int j = i + 1; j < n; j++) {
        final dx = _nodes[i].x - _nodes[j].x;
        final dy = _nodes[i].y - _nodes[j].y;
        final dist = math.sqrt(dx * dx + dy * dy).clamp(1.0, 1000.0);
        final force = repulsion / (dist * dist);
        final fx_ = force * dx / dist;
        final fy_ = force * dy / dist;
        fx[i] += fx_;
        fy[i] += fy_;
        fx[j] -= fx_;
        fy[j] -= fy_;
      }
    }

    // Spring forces for edges
    for (final edge in _edges) {
      final si = _nodes.indexWhere((n) => n.node.id == edge.sourceId);
      final ti = _nodes.indexWhere((n) => n.node.id == edge.targetId);
      if (si < 0 || ti < 0) continue;
      final dx = _nodes[ti].x - _nodes[si].x;
      final dy = _nodes[ti].y - _nodes[si].y;
      final dist = math.sqrt(dx * dx + dy * dy).clamp(1.0, 1000.0);
      final force = springK * (dist - restLen);
      final fx_ = force * dx / dist;
      final fy_ = force * dy / dist;
      fx[si] += fx_;
      fy[si] += fy_;
      fx[ti] -= fx_;
      fy[ti] -= fy_;
    }

    // Gravity toward center
    for (int i = 0; i < n; i++) {
      fx[i] -= _nodes[i].x * 0.01;
      fy[i] -= _nodes[i].y * 0.01;
    }

    // Integrate
    setState(() {
      for (int i = 0; i < n; i++) {
        final node = _nodes[i];
        final newVx = (node.vx + fx[i]) * damping;
        final newVy = (node.vy + fy[i]) * damping;
        _nodes[i] = node.copyWith(
          x: node.x + newVx,
          y: node.y + newVy,
          vx: newVx,
          vy: newVy,
        );
      }
    });
  }

  Offset _worldToScreen(Size size, double x, double y) {
    return Offset(
      size.width / 2 + (x + _pan.dx) * _scale,
      size.height / 2 + (y + _pan.dy) * _scale,
    );
  }

  Offset _screenToWorld(Size size, Offset screen) {
    return Offset(
      (screen.dx - size.width / 2) / _scale - _pan.dx,
      (screen.dy - size.height / 2) / _scale - _pan.dy,
    );
  }

  String? _nodeAt(Size size, Offset screen) {
    for (final node in _nodes) {
      final pos = _worldToScreen(size, node.x, node.y);
      final r = _nodeRadius(node.node) * _scale;
      final d = (screen - pos).distance;
      if (d <= r) return node.node.id;
    }
    return null;
  }

  double _nodeRadius(GraphNode node) {
    final base = 20.0 +
        (node.connectionCount.clamp(0, 20) / 20.0) * 20.0;
    return node.id == widget.rootId ? base + 8 : base;
  }

  void _fitToScreen(Size size) {
    if (_nodes.isEmpty) return;
    double minX = _nodes.first.x,
        maxX = _nodes.first.x,
        minY = _nodes.first.y,
        maxY = _nodes.first.y;
    for (final n in _nodes) {
      if (n.x < minX) minX = n.x;
      if (n.x > maxX) maxX = n.x;
      if (n.y < minY) minY = n.y;
      if (n.y > maxY) maxY = n.y;
    }
    final cx = (minX + maxX) / 2;
    final cy = (minY + maxY) / 2;
    final w = (maxX - minX + 100).clamp(1.0, double.infinity);
    final h = (maxY - minY + 100).clamp(1.0, double.infinity);
    final s =
        math.min(size.width / w, size.height / h).clamp(0.2, 2.0);
    setState(() {
      _scale = s;
      _pan = Offset(-cx, -cy);
    });
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, constraints) {
      final size = Size(constraints.maxWidth, constraints.maxHeight);
      return Stack(
        children: [
          // Graph canvas
          Listener(
            onPointerSignal: (event) {
              if (event is PointerScrollEvent) {
                setState(() {
                  _scale =
                      (_scale * (event.scrollDelta.dy > 0 ? 0.9 : 1.1))
                          .clamp(0.1, 5.0);
                });
              }
            },
            child: GestureDetector(
              onPanStart: (details) {
                final nodeId = _nodeAt(size, details.localPosition);
                if (nodeId != null) {
                  setState(() => _draggingNodeId = nodeId);
                  _dragStart = details.localPosition;
                  final ni = _nodes.indexWhere((n) => n.node.id == nodeId);
                  _dragNodeStart = Offset(_nodes[ni].x, _nodes[ni].y);
                }
              },
              onPanUpdate: (details) {
                if (_draggingNodeId != null) {
                  // Drag node
                  final ni = _nodes
                      .indexWhere((n) => n.node.id == _draggingNodeId);
                  if (ni >= 0) {
                    final world = _screenToWorld(size, details.localPosition);
                    setState(() {
                      _nodes[ni] = _nodes[ni]
                          .copyWith(x: world.dx, y: world.dy, vx: 0, vy: 0);
                    });
                  }
                } else {
                  // Pan viewport
                  setState(() {
                    _pan += details.delta / _scale;
                  });
                }
              },
              onPanEnd: (_) {
                setState(() => _draggingNodeId = null);
              },
              onTapUp: (details) {
                final nodeId = _nodeAt(size, details.localPosition);
                if (nodeId != null) {
                  widget.onNodeTap?.call(nodeId);
                }
              },
              child: MouseRegion(
                onHover: (event) {
                  final nodeId = _nodeAt(size, event.localPosition);
                  if (nodeId != _hoveredNodeId) {
                    setState(() => _hoveredNodeId = nodeId);
                  }
                },
                onExit: (_) => setState(() => _hoveredNodeId = null),
                child: CustomPaint(
                  size: size,
                  painter: _GraphPainter(
                    nodes: _nodes,
                    edges: _edges,
                    rootId: widget.rootId,
                    scale: _scale,
                    pan: _pan,
                    hoveredNodeId: _hoveredNodeId,
                    nodeRadius: _nodeRadius,
                  ),
                ),
              ),
            ),
          ),
          // Fit button
          Positioned(
            right: 16,
            bottom: 16,
            child: FloatingActionButton.small(
              onPressed: () => _fitToScreen(size),
              tooltip: 'Fit to screen',
              child: const Icon(Icons.fit_screen),
            ),
          ),
          // Tooltip overlay
          if (_hoveredNodeId != null)
            _buildTooltip(size, _hoveredNodeId!),
        ],
      );
    });
  }

  Widget _buildTooltip(Size size, String nodeId) {
    final ni = _nodes.indexWhere((n) => n.node.id == nodeId);
    if (ni < 0) return const SizedBox.shrink();
    final node = _nodes[ni];
    final pos = _worldToScreen(size, node.x, node.y);
    return Positioned(
      left: (pos.dx + 20).clamp(0, size.width - 200),
      top: (pos.dy - 60).clamp(0, size.height - 80),
      child: IgnorePointer(
        child: Container(
          constraints: const BoxConstraints(maxWidth: 200),
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: Colors.black87,
            borderRadius: BorderRadius.circular(6),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                node.node.title,
                style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                    fontSize: 12),
              ),
              const SizedBox(height: 2),
              Text(
                'Freshness: ${node.node.freshnessStatus}',
                style:
                    const TextStyle(color: Colors.white70, fontSize: 11),
              ),
              Text(
                'Connections: ${node.node.connectionCount}',
                style:
                    const TextStyle(color: Colors.white70, fontSize: 11),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _GraphPainter extends CustomPainter {
  final List<_NodeState> nodes;
  final List<GraphEdge> edges;
  final String? rootId;
  final double scale;
  final Offset pan;
  final String? hoveredNodeId;
  final double Function(GraphNode) nodeRadius;

  _GraphPainter({
    required this.nodes,
    required this.edges,
    this.rootId,
    required this.scale,
    required this.pan,
    this.hoveredNodeId,
    required this.nodeRadius,
  });

  Offset _toScreen(Size size, double x, double y) {
    return Offset(
      size.width / 2 + (x + pan.dx) * scale,
      size.height / 2 + (y + pan.dy) * scale,
    );
  }

  Color _freshnessColor(String status) {
    switch (status) {
      case 'aging':
        return const Color(0xFFF59E0B);
      case 'stale':
        return const Color(0xFFEF4444);
      default:
        return const Color(0xFF10B981);
    }
  }

  @override
  void paint(Canvas canvas, Size size) {
    // Draw background
    canvas.drawRect(
      Rect.fromLTWH(0, 0, size.width, size.height),
      Paint()..color = Colors.white,
    );

    final edgePaint = Paint()
      ..color = const Color(0xFF9CA3AF)
      ..strokeWidth = 1.5 * scale
      ..style = PaintingStyle.stroke;

    final dependsPaint = Paint()
      ..color = const Color(0xFF9CA3AF)
      ..strokeWidth = 2.5 * scale
      ..style = PaintingStyle.stroke;

    // Build node position map
    final posMap = <String, Offset>{};
    for (final n in nodes) {
      posMap[n.node.id] = _toScreen(size, n.x, n.y);
    }

    // Draw edges
    for (final edge in edges) {
      final src = posMap[edge.sourceId];
      final tgt = posMap[edge.targetId];
      if (src == null || tgt == null) continue;

      final paint =
          edge.edgeType == 'depends_on' ? dependsPaint : edgePaint;

      // Draw curved line
      final path = Path();
      final ctrl = Offset(
        (src.dx + tgt.dx) / 2 + (tgt.dy - src.dy) * 0.2,
        (src.dy + tgt.dy) / 2 - (tgt.dx - src.dx) * 0.2,
      );
      path.moveTo(src.dx, src.dy);
      path.quadraticBezierTo(ctrl.dx, ctrl.dy, tgt.dx, tgt.dy);
      canvas.drawPath(path, paint);

      // Edge label
      final labelPos = Offset(
        (src.dx + tgt.dx) / 2,
        (src.dy + tgt.dy) / 2,
      );
      final tp = TextPainter(
        text: TextSpan(
          text: edge.edgeType,
          style: TextStyle(
              fontSize: (10 * scale).clamp(8, 12),
              color: const Color(0xFF9CA3AF)),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      tp.paint(canvas,
          labelPos - Offset(tp.width / 2, tp.height / 2));
    }

    // Draw nodes
    for (final n in nodes) {
      final pos = posMap[n.node.id];
      if (pos == null) continue;

      final r = nodeRadius(n.node) * scale;
      final isRoot = n.node.id == rootId;
      final isHovered = n.node.id == hoveredNodeId;
      final fillColor = _freshnessColor(n.node.freshnessStatus);

      // Shadow
      if (isHovered || isRoot) {
        canvas.drawCircle(
          pos + const Offset(2, 2),
          r + 2,
          Paint()..color = Colors.black.withOpacity(0.15),
        );
      }

      // Fill
      canvas.drawCircle(
        pos,
        r,
        Paint()..color = fillColor.withOpacity(0.2),
      );

      // Border
      canvas.drawCircle(
        pos,
        r,
        Paint()
          ..color = fillColor
          ..style = PaintingStyle.stroke
          ..strokeWidth = (isRoot ? 3.0 : 1.5) * scale,
      );

      // Title text
      final maxChars = 18;
      final title = n.node.title.length > maxChars
          ? '${n.node.title.substring(0, maxChars)}...'
          : n.node.title;

      final tp = TextPainter(
        text: TextSpan(
          text: title,
          style: TextStyle(
            fontSize: (11 * scale).clamp(9, 14),
            fontWeight: isRoot ? FontWeight.bold : FontWeight.normal,
            color: Colors.black87,
          ),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
        maxLines: 2,
      )..layout(maxWidth: r * 2);

      tp.paint(canvas, pos - Offset(tp.width / 2, tp.height / 2));
    }
  }

  @override
  bool shouldRepaint(_GraphPainter old) =>
      old.nodes != nodes ||
      old.scale != scale ||
      old.pan != pan ||
      old.hoveredNodeId != hoveredNodeId;
}

class _NodeState {
  final GraphNode node;
  final double x;
  final double y;
  final double vx;
  final double vy;
  final bool isRoot;

  const _NodeState({
    required this.node,
    required this.x,
    required this.y,
    required this.vx,
    required this.vy,
    required this.isRoot,
  });

  _NodeState copyWith({
    GraphNode? node,
    double? x,
    double? y,
    double? vx,
    double? vy,
    bool? isRoot,
  }) {
    return _NodeState(
      node: node ?? this.node,
      x: x ?? this.x,
      y: y ?? this.y,
      vx: vx ?? this.vx,
      vy: vy ?? this.vy,
      isRoot: isRoot ?? this.isRoot,
    );
  }
}
