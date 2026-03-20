import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';
import '../providers/space_provider.dart';
import '../providers/freshness_provider.dart';
import '../models/freshness.dart';
import '../widgets/freshness_badge.dart';
import '../widgets/search_bar.dart' as kb;

class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(spaceListProvider.notifier).load();
      ref.read(freshnessDashboardProvider.notifier).load();
    });
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final spaceState = ref.watch(spaceListProvider);
    final freshnessState = ref.watch(freshnessDashboardProvider);
    final user = authState.user;

    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.hub_outlined, color: Colors.white),
            SizedBox(width: 8),
            Text('Knomantem',
                style: TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold)),
          ],
        ),
        actions: [
          kb.KnomantemSearchBar(
            onSearch: (q) => context.push('/search?q=${Uri.encodeComponent(q)}'),
          ),
          const SizedBox(width: 8),
          PopupMenuButton<String>(
            icon: CircleAvatar(
              backgroundColor: Colors.white24,
              child: Text(
                user?.displayName.isNotEmpty == true
                    ? user!.displayName[0].toUpperCase()
                    : 'U',
                style: const TextStyle(color: Colors.white),
              ),
            ),
            onSelected: (value) {
              if (value == 'logout') {
                ref.read(authProvider.notifier).logout();
              } else if (value == 'graph') {
                context.push('/graph');
              }
            },
            itemBuilder: (_) => [
              PopupMenuItem(
                value: 'profile',
                enabled: false,
                child: Text(user?.displayName ?? 'User'),
              ),
              const PopupMenuDivider(),
              const PopupMenuItem(
                value: 'graph',
                child: Row(
                  children: [
                    Icon(Icons.account_tree_outlined),
                    SizedBox(width: 8),
                    Text('Graph View'),
                  ],
                ),
              ),
              const PopupMenuDivider(),
              const PopupMenuItem(
                value: 'logout',
                child: Row(
                  children: [
                    Icon(Icons.logout),
                    SizedBox(width: 8),
                    Text('Sign Out'),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: Row(
        children: [
          // Left sidebar
          SizedBox(
            width: 240,
            child: _Sidebar(spaceState: spaceState),
          ),
          const VerticalDivider(width: 1),
          // Main content
          Expanded(
            child: _MainContent(freshnessState: freshnessState),
          ),
        ],
      ),
    );
  }
}

class _Sidebar extends ConsumerWidget {
  final SpaceListState spaceState;

  const _Sidebar({required this.spaceState});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      color: const Color(0xFFF8F7FF),
      child: ListView(
        padding: const EdgeInsets.symmetric(vertical: 8),
        children: [
          _SidebarSection(
            title: 'SPACES',
            action: IconButton(
              icon: const Icon(Icons.add, size: 18),
              onPressed: () => _showCreateSpaceDialog(context, ref),
              tooltip: 'New Space',
            ),
            children: [
              if (spaceState.isLoading)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
                )
              else
                ...spaceState.spaces.map((space) => ListTile(
                      dense: true,
                      leading: Text(
                        space.icon ?? '📁',
                        style: const TextStyle(fontSize: 16),
                      ),
                      title: Text(
                        space.name,
                        style: const TextStyle(fontSize: 14),
                        overflow: TextOverflow.ellipsis,
                      ),
                      trailing: space.pageCount != null
                          ? Text(
                              '${space.pageCount}',
                              style: TextStyle(
                                  fontSize: 12, color: Colors.grey[500]),
                            )
                          : null,
                      onTap: () {
                        ref
                            .read(selectedSpaceIdProvider.notifier)
                            .state = space.id;
                        context.push('/spaces/${space.id}');
                      },
                    )),
            ],
          ),
          _SidebarSection(
            title: 'TOOLS',
            children: [
              _KnowledgeHealthNavTile(ref: ref),
            ],
          ),
        ],
      ),
    );
  }

  void _showCreateSpaceDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Space'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: 'Space Name'),
          autofocus: true,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () async {
              if (nameCtrl.text.trim().isEmpty) return;
              Navigator.pop(ctx);
              await ref
                  .read(spaceListProvider.notifier)
                  .createSpace(name: nameCtrl.text.trim());
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }
}

class _SidebarSection extends StatelessWidget {
  final String title;
  final Widget? action;
  final List<Widget> children;

  const _SidebarSection({
    required this.title,
    this.action,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: Colors.grey[600],
                    letterSpacing: 0.5,
                  ),
                ),
              ),
              if (action != null) action!,
            ],
          ),
        ),
        ...children,
        const SizedBox(height: 8),
      ],
    );
  }
}

class _MainContent extends ConsumerWidget {
  final FreshnessDashboardState freshnessState;

  const _MainContent({required this.freshnessState});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final staleCount = freshnessState.pages
        .where((p) => p.status == FreshnessStatus.stale)
        .length;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _SectionTitle('Recent Pages'),
          const SizedBox(height: 12),
          _RecentPagesPlaceholder(),
          const SizedBox(height: 24),
          Row(
            children: [
              _SectionTitle('Freshness Alerts'),
              if (staleCount > 0) ...[
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 7, vertical: 2),
                  decoration: BoxDecoration(
                    color: const Color(0xFF8B3A3A),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    '$staleCount',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 11,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              ],
              const Spacer(),
              if (staleCount > 0)
                TextButton(
                  onPressed: () =>
                      context.push('/freshness?filter=stale'),
                  style: TextButton.styleFrom(
                    foregroundColor: const Color(0xFF5A8A48),
                    padding: EdgeInsets.zero,
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  child: const Text(
                    'View all →',
                    style: TextStyle(fontSize: 13),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          _FreshnessAlerts(freshnessState: freshnessState, ref: ref),
        ],
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  final String text;
  const _SectionTitle(this.text);

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
    );
  }
}

class _RecentPagesPlaceholder extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.grey[200]!),
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Center(
        child: Text(
          'Navigate to a space to see recent pages.',
          style: TextStyle(color: Colors.grey),
        ),
      ),
    );
  }
}

// ─── Knowledge Health Sidebar Tile ───────────────────────────────────────────

class _KnowledgeHealthNavTile extends StatelessWidget {
  final WidgetRef ref;

  const _KnowledgeHealthNavTile({required this.ref});

  @override
  Widget build(BuildContext context) {
    final freshnessState = ref.watch(freshnessDashboardProvider);
    final staleCount = freshnessState.pages
        .where((p) => p.status == FreshnessStatus.stale)
        .length;

    return ListTile(
      dense: true,
      leading: const Icon(Icons.monitor_heart_outlined, size: 18),
      title: const Text(
        'Knowledge Health',
        style: TextStyle(fontSize: 14),
        overflow: TextOverflow.ellipsis,
      ),
      trailing: staleCount > 0
          ? Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: const Color(0xFF8B3A3A),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(
                '$staleCount',
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 10,
                  fontWeight: FontWeight.bold,
                ),
              ),
            )
          : null,
      onTap: () => context.push('/freshness'),
    );
  }
}

// ─── Freshness Alerts Panel ───────────────────────────────────────────────────

class _FreshnessAlerts extends StatelessWidget {
  final FreshnessDashboardState freshnessState;
  final WidgetRef ref;

  const _FreshnessAlerts({
    required this.freshnessState,
    required this.ref,
  });

  @override
  Widget build(BuildContext context) {
    if (freshnessState.isLoading) {
      return const Center(child: CircularProgressIndicator(strokeWidth: 2));
    }

    final staleItems = freshnessState.pages
        .where((p) =>
            p.status == FreshnessStatus.stale ||
            p.status == FreshnessStatus.aging)
        .take(5)
        .toList();

    if (staleItems.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.green[50],
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: Colors.green[200]!),
        ),
        child: const Row(
          children: [
            Icon(Icons.check_circle, color: Colors.green),
            SizedBox(width: 8),
            Text('All pages are fresh!',
                style: TextStyle(color: Colors.green)),
          ],
        ),
      );
    }

    return Column(
      children: staleItems.map((item) {
        return Card(
          margin: const EdgeInsets.only(bottom: 8),
          child: ListTile(
            leading: FreshnessBadge(
              status: item.status,
              score: item.freshnessScore,
            ),
            title: Text(item.title),
            subtitle: Text(
              item.space?['name'] as String? ?? 'Unknown space',
              style: const TextStyle(fontSize: 12),
            ),
            trailing: ElevatedButton.icon(
              onPressed: () async {
                await ref
                    .read(freshnessDashboardProvider.notifier)
                    .verifyPage(item.pageId);
              },
              icon: const Icon(Icons.check, size: 16),
              label: const Text('Verify'),
              style: ElevatedButton.styleFrom(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                textStyle: const TextStyle(fontSize: 12),
              ),
            ),
            onTap: () => context.push('/pages/${item.pageId}'),
          ),
        );
      }).toList(),
    );
  }
}
