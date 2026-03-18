class Tag {
  final String id;
  final String name;
  final String? color;
  final bool isAiGenerated;
  final int? pageCount;
  final double? confidenceScore;
  final DateTime? createdAt;

  const Tag({
    required this.id,
    required this.name,
    this.color,
    this.isAiGenerated = false,
    this.pageCount,
    this.confidenceScore,
    this.createdAt,
  });

  factory Tag.fromJson(Map<String, dynamic> json) {
    return Tag(
      id: json['id'] as String,
      name: json['name'] as String,
      color: json['color'] as String?,
      isAiGenerated: json['is_ai_generated'] as bool? ?? false,
      pageCount: json['page_count'] as int?,
      confidenceScore: json['confidence_score'] != null
          ? (json['confidence_score'] as num).toDouble()
          : null,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'color': color,
      'is_ai_generated': isAiGenerated,
      'page_count': pageCount,
      'confidence_score': confidenceScore,
      'created_at': createdAt?.toIso8601String(),
    };
  }
}
