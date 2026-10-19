import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:olona/core/presentation/window_size.dart';

class AppShell extends StatelessWidget {
  const AppShell({
    super.key,
    required this.child,
    this.location = '',
    this.isAuth = false,
  });

  final Widget child;

  /// Current router location, e.g. `/sales`. Drives the selected nav item.
  final String location;

  final bool isAuth;

  @override
  Widget build(BuildContext context) {
    if (isAuth) {
      return Scaffold(
        body: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 440),
                child: child,
              ),
            ),
          ),
        ),
      );
    }

    final size = windowSizeFor(MediaQuery.sizeOf(context).width);
    final selectedIndex = _indexOf(location);

    return Scaffold(
      body: Row(
        children: [
          if (size == WindowSize.large)
            _Sidebar(
              selectedIndex: selectedIndex,
              onSelected: (i) => _select(context, i),
            ),
          if (size == WindowSize.medium || size == WindowSize.expanded)
            _Rail(
              selectedIndex: selectedIndex,
              onSelected: (i) => _select(context, i),
            ),
          Expanded(child: SafeArea(child: child)),
        ],
      ),
      bottomNavigationBar: size == WindowSize.compact
          ? _BottomNav(
              selectedIndex: selectedIndex,
              onSelected: (i) => _select(context, i),
            )
          : null,
    );
  }

  void _select(BuildContext context, int index) {
    final path = _destinations[index].path;
    if (path != location) context.go(path);
  }

  int _indexOf(String location) {
    for (var i = 0; i < _destinations.length; i++) {
      final path = _destinations[i].path;
      final matches = path == '/'
          ? location == '/'
          : location == path || location.startsWith('$path/');
      if (matches) return i;
    }
    return 0;
  }
}

class _Destination {
  const _Destination(this.path, this.icon, this.label);

  final String path;
  final IconData icon;
  final String label;
}

const _destinations = [
  _Destination('/', Icons.space_dashboard_outlined, 'Dashboard'),
  _Destination('/sales', Icons.receipt_long_outlined, 'Sales'),
  _Destination('/inventory', Icons.inventory_2_outlined, 'Inventory'),
  _Destination('/analytics', Icons.query_stats_outlined, 'Analytics'),
];

class _BottomNav extends StatelessWidget {
  const _BottomNav({required this.selectedIndex, required this.onSelected});

  final int selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    return NavigationBar(
      selectedIndex: selectedIndex,
      onDestinationSelected: onSelected,
      destinations: [
        for (final destination in _destinations)
          NavigationDestination(
            icon: Icon(destination.icon),
            label: destination.label,
          ),
      ],
    );
  }
}

class _Rail extends StatelessWidget {
  const _Rail({required this.selectedIndex, required this.onSelected});

  final int selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    return NavigationRail(
      selectedIndex: selectedIndex,
      onDestinationSelected: onSelected,
      labelType: NavigationRailLabelType.all,
      destinations: [
        for (final destination in _destinations)
          NavigationRailDestination(
            icon: Icon(destination.icon),
            label: Text(destination.label),
          ),
      ],
    );
  }
}

class _Sidebar extends StatelessWidget {
  const _Sidebar({required this.selectedIndex, required this.onSelected});

  final int selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      width: 240,
      decoration: BoxDecoration(
        border: Border(right: BorderSide(color: colors.outlineVariant)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 24, 20, 28),
            child: Text(
              'Olona',
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
            ),
          ),
          for (var i = 0; i < _destinations.length; i++)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              child: ListTile(
                leading: Icon(_destinations[i].icon),
                title: Text(_destinations[i].label),
                selected: i == selectedIndex,
                selectedTileColor: colors.secondary,
                selectedColor: colors.primary,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8),
                ),
                onTap: () => onSelected(i),
              ),
            ),
        ],
      ),
    );
  }
}
