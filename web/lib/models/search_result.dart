import 'freshness.dart';
import 'tag.dart';

class SearchResult {
  final String pageId;
  final String title;
  final String excerpt;
  final Map<String, dynamic>? space;
  final FreshnessInfo? freshness;
  final List<Tag> tags;
  final double score;
  final DateTime updatedAt;

  const SearchResult({
    required this.pageId,
    required this.title,
    required this.excerpt,
    this.space,
    this.freshness,
    required this.tags,
    required this.score,
    required this.updatedAt,
  });

  factory SearchResult.fromJson(Map<String, dynamic> json) {
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
        pageId: json['page_id'] as String,
        freshnessScore: ((f['score'] ?? f['freshness_score'] ?? 0.0) as num).toDouble(),
        reviewIntervalDays: 30,
        status: FreshnessStatusExtension.fromString(
            f['status'] as String? ?? 'fresh'),
      );
    }

    return SearchResult(
      pageId: json['page_id'] as String,
      title: json['title'] as String,
      excerpt: json['excerpt'] as String? ?? '',
      space: json['space'] as Map<String, dynamic>?,
      freshness: freshness,
      tags: tags,
      score: (json['score'] as num? ?? 0.0).toDouble(),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class SearchFacets {
  final List<Map<String, dynamic>> spaces;
  final List<Map<String, dynamic>> tags;
  final List<Map<String, dynamic>> freshness;

  const SearchFacets({
    required this.spaces,
    required this.tags,
    required this.freshness,
  });

  factory SearchFacets.fromJson(Map<String, dynamic> json) {
    return SearchFacets(
      spaces: (json['spaces'] as List?)
              ?.map((e) => e as Map<String, dynamic>)
              .toList() ??
          [],
      tags: (json['tags'] as List?)
              ?.map((e) => e as Map<String, dynamic>)
              .toList() ??
          [],
      freshness: (json['freshness'] as List?)
              ?.map((e) => e as Map<String, dynamic>)
              .toList() ??
          [],
    );
  }
}

class SearchResponse {
  final List<SearchResult> results;
  final SearchFacets facets;
  final int queryTimeMs;
  final int total;
  final String? nextCursor;
  final bool hasMore;

  const SearchResponse({
    required this.results,
    required this.facets,
    required this.queryTimeMs,
    required this.total,
    this.nextCursor,
    required this.hasMore,
  });

  factory SearchResponse.fromJson(Map<String, dynamic> json) {
    final data = json['data'] as Map<String, dynamic>;
    final pagination = json['pagination'] as Map<String, dynamic>?;

    final results = (data['results'] as List? ?? [])
        .map((r) => SearchResult.fromJson(r as Map<String, dynamic>))
        .toList();

    final facets = data['facets'] != null
        ? SearchFacets.fromJson(data['facets'] as Map<String, dynamic>)
        : const SearchFacets(spaces: [], tags: [], freshness: []);

    return SearchResponse(
      results: results,
      facets: facets,
      queryTimeMs: data['query_time_ms'] as int? ?? 0,
      total: data['total'] as int? ?? 0,
      nextCursor: pagination?['next_cursor'] as String?,
      hasMore: pagination?['has_more'] as bool? ?? false,
    );
  }
}
