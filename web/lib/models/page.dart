import 'freshness.dart';
import 'tag.dart';

class PageSummary {
  final String id;
  final String spaceId;
  final String? parentId;
  final String title;
  final String slug;
  final String? icon;
  final int position;
  final int depth;
  final bool isTemplate;
  final bool hasChildren;
  final FreshnessStatus freshnessStatus;
  final DateTime createdAt;
  final DateTime updatedAt;

  const PageSummary({
    required this.id,
    required this.spaceId,
    this.parentId,
    required this.title,
    required this.slug,
    this.icon,
    required this.position,
    required this.depth,
    required this.isTemplate,
    required this.hasChildren,
    required this.freshnessStatus,
    required this.createdAt,
    required this.updatedAt,
  });

  factory PageSummary.fromJson(Map<String, dynamic> json) {
    return PageSummary(
      id: json['id'] as String,
      spaceId: json['space_id'] as String,
      parentId: json['parent_id'] as String?,
      title: json['title'] as String,
      slug: json['slug'] as String,
      icon: json['icon'] as String?,
      position: json['position'] as int? ?? 0,
      depth: json['depth'] as int? ?? 0,
      isTemplate: json['is_template'] as bool? ?? false,
      hasChildren: json['has_children'] as bool? ?? false,
      freshnessStatus: FreshnessStatusExtension.fromString(
          json['freshness_status'] as String? ?? 'fresh'),
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class PageDetail {
  final String id;
  final String spaceId;
  final String? parentId;
  final String title;
  final String slug;
  final Map<String, dynamic>? content;
  final String? icon;
  final String? coverImage;
  final int position;
  final int depth;
  final bool isTemplate;
  final String? createdBy;
  final String? updatedBy;
  final FreshnessInfo? freshness;
  final List<Tag> tags;
  final int? version;
  final DateTime createdAt;
  final DateTime updatedAt;

  const PageDetail({
    required this.id,
    required this.spaceId,
    this.parentId,
    required this.title,
    required this.slug,
    this.content,
    this.icon,
    this.coverImage,
    required this.position,
    required this.depth,
    required this.isTemplate,
    this.createdBy,
    this.updatedBy,
    this.freshness,
    required this.tags,
    this.version,
    required this.createdAt,
    required this.updatedAt,
  });

  factory PageDetail.fromJson(Map<String, dynamic> json) {
    List<Tag> tags = [];
    if (json['tags'] != null) {
      tags = (json['tags'] as List)
          .map((t) => Tag.fromJson(t as Map<String, dynamic>))
          .toList();
    }

    FreshnessInfo? freshness;
    if (json['freshness'] != null) {
      final f = json['freshness'] as Map<String, dynamic>;
      freshness = FreshnessInfo(
        pageId: json['id'] as String,
        freshnessScore: ((f['score'] ?? f['freshness_score'] ?? 0.0) as num).toDouble(),
        reviewIntervalDays: f['review_interval_days'] as int? ?? 30,
        status: FreshnessStatusExtension.fromString(
            f['status'] as String? ?? 'fresh'),
        lastVerifiedAt: f['last_verified_at'] != null
            ? DateTime.parse(f['last_verified_at'] as String)
            : null,
        nextReviewAt: f['next_review_at'] != null
            ? DateTime.parse(f['next_review_at'] as String)
            : null,
      );
    }

    return PageDetail(
      id: json['id'] as String,
      spaceId: json['space_id'] as String,
      parentId: json['parent_id'] as String?,
      title: json['title'] as String,
      slug: json['slug'] as String,
      content: json['content'] as Map<String, dynamic>?,
      icon: json['icon'] as String?,
      coverImage: json['cover_image'] as String?,
      position: json['position'] as int? ?? 0,
      depth: json['depth'] as int? ?? 0,
      isTemplate: json['is_template'] as bool? ?? false,
      createdBy: json['created_by'] as String?,
      updatedBy: json['updated_by'] as String?,
      freshness: freshness,
      tags: tags,
      version: json['version'] as int?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  PageDetail copyWith({
    String? id,
    String? spaceId,
    String? parentId,
    String? title,
    String? slug,
    Map<String, dynamic>? content,
    String? icon,
    String? coverImage,
    int? position,
    int? depth,
    bool? isTemplate,
    String? createdBy,
    String? updatedBy,
    FreshnessInfo? freshness,
    List<Tag>? tags,
    int? version,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return PageDetail(
      id: id ?? this.id,
      spaceId: spaceId ?? this.spaceId,
      parentId: parentId ?? this.parentId,
      title: title ?? this.title,
      slug: slug ?? this.slug,
      content: content ?? this.content,
      icon: icon ?? this.icon,
      coverImage: coverImage ?? this.coverImage,
      position: position ?? this.position,
      depth: depth ?? this.depth,
      isTemplate: isTemplate ?? this.isTemplate,
      createdBy: createdBy ?? this.createdBy,
      updatedBy: updatedBy ?? this.updatedBy,
      freshness: freshness ?? this.freshness,
      tags: tags ?? this.tags,
      version: version ?? this.version,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  /// Convert page content to a simple plain-text representation
  String get contentText {
    if (content == null) return '';
    return _extractText(content!);
  }

  String _extractText(Map<String, dynamic> node) {
    final buffer = StringBuffer();
    if (node['text'] != null) {
      buffer.write(node['text']);
    }
    final children = node['content'] as List?;
    if (children != null) {
      for (final child in children) {
        buffer.write(_extractText(child as Map<String, dynamic>));
      }
    }
    return buffer.toString();
  }
}
