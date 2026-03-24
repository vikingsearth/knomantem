import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/freshness.dart';
import '../providers/freshness_dashboard_provider.dart';
import '../widgets/freshness_badge.dart';

// ─── Theme Constants ──────────────────────────────────────────────────────────

const _bgColor = Color(0xFF141A12);
const _surfaceColor = Color(0xFF1E2619);
const _borderColor = Color(0xFF3A4A30);
const _mutedTextColor = Color(0xFF8A9A80);

const _freshColor = Color(0xFF5A8A48);
const _agingColor = Color(0xFFB8860B);
const _staleColor = Color(0xFF8B3A3A);

const _freshColorBright = Color(0xFF10B981);
const _agingColorBright = Color(0xFFF59E0B);
const _staleColorBright = Color(0xFFEF4444);

// ─── Screen ───────────────────────────────────────────────────────────────────

class FreshnessDashboardScreen extends ConsumerStatefulWidget {
  /// Optional initial filter applied when the screen opens (e.g. from a deep
  /// link like `/freshness?filter=stale`).
  final String? initialFilter;

  const FreshnessDashboardScreen({super.key, this.initialFilter});

  @override
  ConsumerState<FreshnessDashboardScreen> createState() =>
      _FreshnessDashboardScreenState();
}

class _FreshnessDashboardScreenState
    extends ConsumerState<FreshnessDashboardScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final notifier =
          ref.read(freshnessDashboardViewProvider.notifier);

      // Apply the initial filter from the URL query param if provided.
      if (widget.initialFilter != null) {
        final filterMap = {
          'fresh': FreshnessDashboardFilter.fresh,
          'aging': FreshnessDashboardFilter.aging,
          'stale': FreshnessDashboardFilter.stale,
        };
        final filter = filterMap[widget.initialFilter];
        if (filter != null) {
          notifier.setFilter(filter);
          return; // setFilter calls load() internally.
        }
      }

      notifier.load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(freshnessDashboardViewProvider);

    return Scaffold(
      backgroundColor: _bgColor,
      appBar: _buildAppBar(context, state),
      body: _buildBody(context, state),
    );
  }

  AppBar _buildAppBar(
      BuildContext context, FreshnessDashboardViewState state) {
    return AppBar(
      backgroundColor: _surfaceColor,
      foregroundColor: Colors.white,
      elevation: 0,
      leading: IconButton(
        icon: const Icon(Icons.arrow_back),
        onPressed: () => context.pop(),
        tooltip: 'Back',
      ),
      title: const Row(
        children: [
          Icon(Icons.monitor_heart_outlined, color: Colors.white, size: 22),
          SizedBox(width: 8),
          Text(
            'Knowledge Health',
            style: TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: 18,
            ),
          ),
        ],
      ),
      actions: [
        if (state.isLoading)
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
            child: SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: Colors.white,
              ),
            ),
          )
        else
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: () =>
                ref.read(freshnessDashboardViewProvider.notifier).load(),
          ),
        const SizedBox(width: 8),
      ],
    );
  }

  Widget _buildBody(
      BuildContext context, FreshnessDashboardViewState state) {
    return Column(
      children: [
        // Error banner (non-blocking — still shows data underneath).
        if (state.error != null) _ErrorBanner(message: state.error!),

        // Health overview stat cards.
        if (state.summary != null) ...[
          _HealthOverviewBar(summary: state.summary!),
          const Divider(height: 1, color: _borderColor),
        ],

        // Filter chips + sort dropdown toolbar.
        _FilterSortBar(state: state),
        const Divider(height: 1, color: _borderColor),

        // Page list or loading/empty states.
        Expanded(child: _PageList(state: state)),
      ],
    );
  }
}

// ─── Health Overview Bar ──────────────────────────────────────────────────────

class _HealthOverviewBar extends StatelessWidget {
  final FreshnessSummary summary;

  const _HealthOverviewBar({required this.summary});

  @override
  Widget build(BuildContext context) {
    final total = summary.totalPages;
    final safeDivisor = total > 0 ? total : 1;

    return Container(
      color: _surfaceColor,
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
      child: Row(
        children: [
          Expanded(
            child: _StatCard(
              label: 'Total Pages',
              count: total,
              accentColor: Colors.white,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _StatCard(
              label: 'Fresh',
              count: summary.fresh,
              percentage: summary.fresh / safeDivisor,
              accentColor: _freshColorBright,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _StatCard(
              label: 'Aging',
              count: summary.aging,
              percentage: summary.aging / safeDivisor,
              accentColor: _agingColorBright,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _StatCard(
              label: 'Stale',
              count: summary.stale,
              percentage: summary.stale / safeDivisor,
              accentColor: _staleColorBright,
            ),
          ),
        ],
      ),
    );
  }
}

class _StatCard extends StatelessWidget {
  final String label;
  final int count;
  final double? percentage;
  final Color accentColor;

  const _StatCard({
    required this.label,
    required this.count,
    this.percentage,
    required this.accentColor,
  });

  @override
  Widget build(BuildContext context) {
    final pctText =
        percentage != null ? ' (${(percentage! * 100).toStringAsFixed(0)}%)' : '';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      decoration: BoxDecoration(
        color: _bgColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '$count$pctText',
            style: TextStyle(
              fontSize: 26,
              fontWeight: FontWeight.bold,
              color: accentColor,
              height: 1.1,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            label,
            style: const TextStyle(
              fontSize: 12,
              color: _mutedTextColor,
              letterSpacing: 0.3,
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Filter / Sort Bar ────────────────────────────────────────────────────────

class _FilterSortBar extends ConsumerWidget {
  final FreshnessDashboardViewState state;

  const _FilterSortBar({required this.state});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notifier = ref.read(freshnessDashboardViewProvider.notifier);

    return Container(
      color: _surfaceColor,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          // Filter chips.
          Wrap(
            spacing: 8,
            children: FreshnessDashboardFilter.values.map((f) {
              final isActive = state.filter == f;
              return FilterChip(
                label: Text(f.label),
                selected: isActive,
                onSelected: (_) => notifier.setFilter(f),
                backgroundColor: _bgColor,
                selectedColor: _freshColor.withOpacity(0.25),
                checkmarkColor: _freshColorBright,
                labelStyle: TextStyle(
                  color: isActive ? _freshColorBright : _mutedTextColor,
                  fontSize: 13,
                  fontWeight:
                      isActive ? FontWeight.w600 : FontWeight.normal,
                ),
                side: BorderSide(
                  color: isActive ? _freshColor : _borderColor,
                ),
                showCheckmark: false,
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              );
            }).toList(),
          ),

          const Spacer(),

          // Sort dropdown.
          DropdownButtonHideUnderline(
            child: DropdownButton<FreshnessDashboardSort>(
              value: state.sort,
              dropdownColor: _surfaceColor,
              icon: const Icon(Icons.sort, color: _mutedTextColor, size: 18),
              style: const TextStyle(color: Colors.white, fontSize: 13),
              onChanged: (v) {
                if (v != null) notifier.setSort(v);
              },
              items: FreshnessDashboardSort.values
                  .map((s) => DropdownMenuItem(
                        value: s,
                        child: Text(s.label),
                      ))
                  .toList(),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Page List ────────────────────────────────────────────────────────────────

class _PageList extends ConsumerWidget {
  final FreshnessDashboardViewState state;

  const _PageList({required this.state});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // First load: full-screen loading indicator.
    if (state.isLoading && state.pages.isEmpty && state.summary == null) {
      return const _SkeletonList();
    }

    if (state.pages.isEmpty && !state.isLoading) {
      return _EmptyState(filter: state.filter);
    }

    return ListView.separated(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      itemCount: state.pages.length,
      separatorBuilder: (_, __) =>
          const Divider(height: 1, color: _borderColor),
      itemBuilder: (context, index) {
        final item = state.pages[index];
        final isVerifying = state.verifyingIds.contains(item.pageId);
        return _PageRow(item: item, isVerifying: isVerifying);
      },
    );
  }
}

// ─── Page Row ─────────────────────────────────────────────────────────────────

class _PageRow extends ConsumerWidget {
  final FreshnessDashboardItem item;
  final bool isVerifying;

  const _PageRow({required this.item, required this.isVerifying});

  Color _scoreColor(double score) {
    if (score >= 70) return _freshColor;
    if (score >= 40) return _agingColor;
    return _staleColor;
  }

  String _formatDate(DateTime? dt) {
    if (dt == null) return '—';
    return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scorePercent = (item.freshnessScore * 100).clamp(0.0, 100.0);
    final scoreColor = _scoreColor(scorePercent);
    final spaceName =
        item.space?['name'] as String? ?? 'Unknown Space';
    final ownerName =
        item.owner?['display_name'] as String? ?? '—';

    return InkWell(
      onTap: () => context.push('/pages/${item.pageId}'),
      borderRadius: BorderRadius.circular(6),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            // Status dot badge.
            FreshnessBadge(status: item.status),
            const SizedBox(width: 12),

            // Title + space name.
            Expanded(
              flex: 3,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    item.title,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    spaceName,
                    style: const TextStyle(
                      color: _mutedTextColor,
                      fontSize: 12,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 12),

            // Freshness score.
            SizedBox(
              width: 56,
              child: Text(
                '${scorePercent.toStringAsFixed(0)}%',
                textAlign: TextAlign.right,
                style: TextStyle(
                  color: scoreColor,
                  fontWeight: FontWeight.bold,
                  fontSize: 14,
                ),
              ),
            ),
            const SizedBox(width: 12),

            // Next review date.
            SizedBox(
              width: 96,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    'Review by',
                    style: TextStyle(
                      color: _mutedTextColor,
                      fontSize: 10,
                      letterSpacing: 0.2,
                    ),
                  ),
                  Text(
                    _formatDate(item.nextReviewAt),
                    style: const TextStyle(
                      color: Colors.white70,
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 12),

            // Owner.
            SizedBox(
              width: 100,
              child: Text(
                ownerName,
                style: const TextStyle(
                  color: _mutedTextColor,
                  fontSize: 12,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            const SizedBox(width: 12),

            // Verify button.
            SizedBox(
              width: 76,
              child: isVerifying
                  ? const Center(
                      child: SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: _freshColorBright,
                        ),
                      ),
                    )
                  : OutlinedButton(
                      onPressed: () => ref
                          .read(freshnessDashboardViewProvider.notifier)
                          .verifyPage(item.pageId),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: _freshColorBright,
                        side: const BorderSide(color: _freshColor),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 10, vertical: 6),
                        minimumSize: Size.zero,
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        textStyle: const TextStyle(fontSize: 12),
                      ),
                      child: const Text('Verify'),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─── Empty State ──────────────────────────────────────────────────────────────

class _EmptyState extends StatelessWidget {
  final FreshnessDashboardFilter filter;

  const _EmptyState({required this.filter});

  @override
  Widget build(BuildContext context) {
    final isAllFilter = filter == FreshnessDashboardFilter.all;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(40),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              isAllFilter ? Icons.check_circle_outline : Icons.filter_list,
              size: 56,
              color: isAllFilter ? _freshColorBright : _mutedTextColor,
            ),
            const SizedBox(height: 16),
            Text(
              isAllFilter
                  ? 'All pages are fresh! Nothing needs attention.'
                  : 'No ${filter.label.toLowerCase()} pages found.',
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: Colors.white70,
                fontSize: 16,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─── Skeleton Loading List ────────────────────────────────────────────────────

class _SkeletonList extends StatelessWidget {
  const _SkeletonList();

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      itemCount: 8,
      separatorBuilder: (_, __) =>
          const Divider(height: 1, color: _borderColor),
      itemBuilder: (_, __) => const _SkeletonRow(),
    );
  }
}

class _SkeletonRow extends StatelessWidget {
  const _SkeletonRow();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 14),
      child: Row(
        children: [
          _SkeletonBox(width: 12, height: 12, radius: 6),
          const SizedBox(width: 12),
          Expanded(
            flex: 3,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _SkeletonBox(width: double.infinity, height: 13),
                const SizedBox(height: 5),
                _SkeletonBox(width: 80, height: 11),
              ],
            ),
          ),
          const SizedBox(width: 12),
          _SkeletonBox(width: 40, height: 14),
          const SizedBox(width: 12),
          _SkeletonBox(width: 90, height: 30),
          const SizedBox(width: 12),
          _SkeletonBox(width: 90, height: 30),
          const SizedBox(width: 12),
          _SkeletonBox(width: 64, height: 30, radius: 4),
        ],
      ),
    );
  }
}

class _SkeletonBox extends StatelessWidget {
  final double width;
  final double height;
  final double radius;

  const _SkeletonBox({
    required this.width,
    required this.height,
    this.radius = 3,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: _borderColor.withOpacity(0.6),
        borderRadius: BorderRadius.circular(radius),
      ),
    );
  }
}

// ─── Error Banner ─────────────────────────────────────────────────────────────

class _ErrorBanner extends StatelessWidget {
  final String message;

  const _ErrorBanner({required this.message});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
      color: _staleColor.withOpacity(0.15),
      child: Row(
        children: [
          const Icon(Icons.warning_amber_rounded,
              color: _staleColorBright, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(color: _staleColorBright, fontSize: 13),
            ),
          ),
        ],
      ),
    );
  }
}
