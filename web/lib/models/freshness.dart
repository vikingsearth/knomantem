import 'package:flutter/material.dart';

enum FreshnessStatus { fresh, aging, stale }

extension FreshnessStatusExtension on FreshnessStatus {
  String get label {
    switch (this) {
      case FreshnessStatus.fresh:
        return 'Fresh';
      case FreshnessStatus.aging:
        return 'Aging';
      case FreshnessStatus.stale:
        return 'Stale';
    }
  }

  Color get color {
    switch (this) {
      case FreshnessStatus.fresh:
        return const Color(0xFF10B981);
      case FreshnessStatus.aging:
        return const Color(0xFFF59E0B);
      case FreshnessStatus.stale:
        return const Color(0xFFEF4444);
    }
  }

  static FreshnessStatus fromString(String s) {
    switch (s) {
      case 'aging':
        return FreshnessStatus.aging;
      case 'stale':
        return FreshnessStatus.stale;
      default:
        return FreshnessStatus.fresh;
    }
  }
}

class FreshnessInfo {
  final String? id;
  final String pageId;
  final String? ownerId;
  final double freshnessScore;
  final int reviewIntervalDays;
  final DateTime? lastReviewedAt;
  final DateTime? nextReviewAt;
  final DateTime? lastVerifiedAt;
  final FreshnessStatus status;
  final double? decayRate;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const FreshnessInfo({
    this.id,
    required this.pageId,
    this.ownerId,
    required this.freshnessScore,
    required this.reviewIntervalDays,
    this.lastReviewedAt,
    this.nextReviewAt,
    this.lastVerifiedAt,
    required this.status,
    this.decayRate,
    this.createdAt,
    this.updatedAt,
  });

  factory FreshnessInfo.fromJson(Map<String, dynamic> json) {
    return FreshnessInfo(
      id: json['id'] as String?,
      pageId: (json['page_id'] ?? '') as String,
      ownerId: json['owner_id'] as String?,
      freshnessScore:
          ((json['freshness_score'] ?? json['score'] ?? 0.0) as num).toDouble(),
      reviewIntervalDays: json['review_interval_days'] as int? ?? 30,
      lastReviewedAt: json['last_reviewed_at'] != null
          ? DateTime.parse(json['last_reviewed_at'] as String)
          : null,
      nextReviewAt: json['next_review_at'] != null
          ? DateTime.parse(json['next_review_at'] as String)
          : null,
      lastVerifiedAt: json['last_verified_at'] != null
          ? DateTime.parse(json['last_verified_at'] as String)
          : null,
      status: FreshnessStatusExtension.fromString(
          json['status'] as String? ?? 'fresh'),
      decayRate: json['decay_rate'] != null
          ? (json['decay_rate'] as num).toDouble()
          : null,
      createdAt: json['created_at'] != null
          ? DateTime.parse(json['created_at'] as String)
          : null,
      updatedAt: json['updated_at'] != null
          ? DateTime.parse(json['updated_at'] as String)
          : null,
    );
  }
}

class FreshnessSummary {
  final int totalPages;
  final int fresh;
  final int aging;
  final int stale;
  final double averageScore;

  const FreshnessSummary({
    required this.totalPages,
    required this.fresh,
    required this.aging,
    required this.stale,
    required this.averageScore,
  });

  factory FreshnessSummary.fromJson(Map<String, dynamic> json) {
    return FreshnessSummary(
      totalPages: json['total_pages'] as int? ?? 0,
      fresh: json['fresh'] as int? ?? 0,
      aging: json['aging'] as int? ?? 0,
      stale: json['stale'] as int? ?? 0,
      averageScore: (json['average_score'] as num? ?? 0.0).toDouble(),
    );
  }
}

class FreshnessDashboardItem {
  final String pageId;
  final String title;
  final Map<String, dynamic>? space;
  final double freshnessScore;
  final FreshnessStatus status;
  final DateTime? lastVerifiedAt;
  final DateTime? nextReviewAt;
  final Map<String, dynamic>? owner;

  const FreshnessDashboardItem({
    required this.pageId,
    required this.title,
    this.space,
    required this.freshnessScore,
    required this.status,
    this.lastVerifiedAt,
    this.nextReviewAt,
    this.owner,
  });

  factory FreshnessDashboardItem.fromJson(Map<String, dynamic> json) {
    return FreshnessDashboardItem(
      pageId: json['page_id'] as String,
      title: json['title'] as String,
      space: json['space'] as Map<String, dynamic>?,
      freshnessScore: (json['freshness_score'] as num? ?? 0.0).toDouble(),
      status: FreshnessStatusExtension.fromString(
          json['status'] as String? ?? 'fresh'),
      lastVerifiedAt: json['last_verified_at'] != null
          ? DateTime.parse(json['last_verified_at'] as String)
          : null,
      nextReviewAt: json['next_review_at'] != null
          ? DateTime.parse(json['next_review_at'] as String)
          : null,
      owner: json['owner'] as Map<String, dynamic>?,
    );
  }
}
