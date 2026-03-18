class Space {
  final String id;
  final String name;
  final String slug;
  final String? description;
  final String? icon;
  final String ownerId;
  final int? pageCount;
  final Map<String, dynamic>? settings;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Space({
    required this.id,
    required this.name,
    required this.slug,
    this.description,
    this.icon,
    required this.ownerId,
    this.pageCount,
    this.settings,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Space.fromJson(Map<String, dynamic> json) {
    return Space(
      id: json['id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      description: json['description'] as String?,
      icon: json['icon'] as String?,
      ownerId: json['owner_id'] as String,
      pageCount: json['page_count'] as int?,
      settings: json['settings'] as Map<String, dynamic>?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'slug': slug,
      'description': description,
      'icon': icon,
      'owner_id': ownerId,
      'page_count': pageCount,
      'settings': settings,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  Space copyWith({
    String? id,
    String? name,
    String? slug,
    String? description,
    String? icon,
    String? ownerId,
    int? pageCount,
    Map<String, dynamic>? settings,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return Space(
      id: id ?? this.id,
      name: name ?? this.name,
      slug: slug ?? this.slug,
      description: description ?? this.description,
      icon: icon ?? this.icon,
      ownerId: ownerId ?? this.ownerId,
      pageCount: pageCount ?? this.pageCount,
      settings: settings ?? this.settings,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }
}
