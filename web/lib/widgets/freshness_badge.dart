import 'package:flutter/material.dart';
import '../models/freshness.dart';

class FreshnessBadge extends StatelessWidget {
  final FreshnessStatus status;
  final double? score;
  final bool showLabel;

  const FreshnessBadge({
    super.key,
    required this.status,
    this.score,
    this.showLabel = false,
  });

  @override
  Widget build(BuildContext context) {
    final color = status.color;

    if (showLabel) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
        decoration: BoxDecoration(
          color: color.withOpacity(0.12),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: color.withOpacity(0.4)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            CircleAvatar(radius: 4, backgroundColor: color),
            const SizedBox(width: 4),
            Text(
              score != null
                  ? '${score!.toStringAsFixed(0)}% ${status.label}'
                  : status.label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w500,
                color: color.withOpacity(0.8),
              ),
            ),
          ],
        ),
      );
    }

    return Tooltip(
      message: score != null
          ? '${status.label} – ${score!.toStringAsFixed(0)}%'
          : status.label,
      child: CircleAvatar(
        radius: 6,
        backgroundColor: color,
      ),
    );
  }
}
