class GraphNode {
  final String id;
  final String title;
  final String? spaceId;
  final String freshnessStatus;
  final int connectionCount;
  final int depthFromRoot;

  const GraphNode({
    required this.id,
    required this.title,
    this.spaceId,
    required this.freshnessStatus,
    required this.connectionCount,
    required this.depthFromRoot,
  });

  factory GraphNode.fromJson(Map<String, dynamic> json) {
    return GraphNode(
      id: json['id'] as String,
      title: json['title'] as String,
      spaceId: json['space_id'] as String?,
      freshnessStatus: json['freshness_status'] as String? ?? 'fresh',
      connectionCount: json['connection_count'] as int? ?? 0,
      depthFromRoot: json['depth_from_root'] as int? ?? 0,
    );
  }
}

class GraphEdge {
  final String? id;
  final String sourceId;
  final String targetId;
  final String edgeType;
  final Map<String, dynamic>? metadata;
  final DateTime? createdAt;

  const GraphEdge({
    this.id,
    required this.sourceId,
    required this.targetId,
    required this.edgeType,
    this.metadata,
    this.createdAt,
  });

  factory GraphEdge.fromJson(Map<String, dynamic> json) {
    return GraphEdge(
      id: json['id'] as String?,
      sourceId: (json['source_id'] ?? json['source_page_id'] ?? '') as String,
      targetId: (json['target_id'] ?? json['target_page_id'] ?? '') as String,
      edgeType: json['edge_type'] as String? ?? 'reference',
      metadata: json['metadata'] as Map<String, dynamic>?,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'] as String)
          : null,
    );
  }

  factory GraphEdge.fromNeighborJson(Map<String, dynamic> json) {
    final source = json['source'] as Map<String, dynamic>?;
    final target = json['target'] as Map<String, dynamic>?;
    return GraphEdge(
      id: json['id'] as String?,
      sourceId: source?['id'] as String? ?? '',
      targetId: target?['id'] as String? ?? '',
      edgeType: json['edge_type'] as String? ?? 'reference',
      metadata: json['metadata'] as Map<String, dynamic>?,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'] as String)
          : null,
    );
  }
}

class GraphData {
  final List<GraphNode> nodes;
  final List<GraphEdge> edges;
  final int totalNodes;
  final int totalEdges;
  final bool truncated;

  const GraphData({
    required this.nodes,
    required this.edges,
    required this.totalNodes,
    required this.totalEdges,
    required this.truncated,
  });

  factory GraphData.fromJson(Map<String, dynamic> json) {
    final data = json['data'] as Map<String, dynamic>;
    return GraphData(
      nodes: (data['nodes'] as List? ?? [])
          .map((n) => GraphNode.fromJson(n as Map<String, dynamic>))
          .toList(),
      edges: (data['edges'] as List? ?? [])
          .map((e) => GraphEdge.fromJson(e as Map<String, dynamic>))
          .toList(),
      totalNodes: data['total_nodes'] as int? ?? 0,
      totalEdges: data['total_edges'] as int? ?? 0,
      truncated: data['truncated'] as bool? ?? false,
    );
  }
}
