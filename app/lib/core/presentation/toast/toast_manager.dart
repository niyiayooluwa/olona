import 'dart:async';
import 'package:flutter/material.dart';
import 'app_toast.dart';

/// Entry representing an active toast in the overlay.
class _ToastEntry {
  _ToastEntry({
    required this.id,
    required this.type,
    required this.title,
    this.description,
    this.actions,
    this.duration,
  });

  final String id;
  final ToastType type;
  final String title;
  final String? description;
  final List<ToastAction>? actions;
  final Duration? duration;
  Timer? timer;
}

/// Global manager for showing overlay toasts.
class ToastManager {
  ToastManager._();
  static final ToastManager instance = ToastManager._();

  OverlayEntry? _overlayEntry;
  final List<_ToastEntry> _toasts = [];
  final GlobalKey<AnimatedListState> _listKey = GlobalKey<AnimatedListState>();

  static const int maxVisibleToasts = 3;

  void show(
    BuildContext context, {
    required ToastType type,
    required String title,
    String? description,
    List<ToastAction>? actions,
    Duration? duration = const Duration(seconds: 4),
  }) {
    final overlayState = Overlay.of(context, rootOverlay: true);

    // If already at or above maxVisibleToasts, dismiss the oldest one (last item in list)
    if (_toasts.length >= maxVisibleToasts) {
      dismiss(_toasts.last.id);
    }

    final id = DateTime.now().microsecondsSinceEpoch.toString();
    final entry = _ToastEntry(
      id: id,
      type: type,
      title: title,
      description: description,
      actions: actions,
      duration: duration,
    );

    if (duration != null) {
      entry.timer = Timer(duration, () {
        dismiss(id);
      });
    }

    _toasts.insert(0, entry);

    if (_overlayEntry == null) {
      _overlayEntry = OverlayEntry(
        builder: (context) => _ToastOverlayHost(
          listKey: _listKey,
          toasts: _toasts,
          onDismiss: dismiss,
        ),
      );
      overlayState.insert(_overlayEntry!);
    } else {
      _listKey.currentState?.insertItem(
        0,
        duration: const Duration(milliseconds: 220),
      );
      _overlayEntry?.markNeedsBuild();
    }
  }

  void dismiss(String id) {
    final index = _toasts.indexWhere((e) => e.id == id);
    if (index == -1) return;

    final removed = _toasts.removeAt(index);
    removed.timer?.cancel();

    _listKey.currentState?.removeItem(
      index,
      (context, animation) => _buildAnimatedToast(removed, animation, () {}),
      duration: const Duration(milliseconds: 200),
    );

    if (_toasts.isEmpty) {
      Future.delayed(const Duration(milliseconds: 250), () {
        if (_toasts.isEmpty) {
          _overlayEntry?.remove();
          _overlayEntry = null;
        }
      });
    }
  }

  static Widget _buildAnimatedToast(
    _ToastEntry entry,
    Animation<double> animation,
    VoidCallback onDismiss,
  ) {
    final curvedAnimation = CurvedAnimation(
      parent: animation,
      curve: Curves.easeOutCubic,
      reverseCurve: Curves.easeInCubic,
    );

    return SlideTransition(
      position: Tween<Offset>(
        begin: const Offset(0.25, 0),
        end: Offset.zero,
      ).animate(curvedAnimation),
      child: FadeTransition(
        opacity: curvedAnimation,
        child: Padding(
          padding: const EdgeInsets.only(top: 10),
          child: AppToast(
            type: entry.type,
            title: entry.title,
            description: entry.description,
            actions: entry.actions,
            onDismiss: onDismiss,
          ),
        ),
      ),
    );
  }
}

class _ToastOverlayHost extends StatelessWidget {
  const _ToastOverlayHost({
    required this.listKey,
    required this.toasts,
    required this.onDismiss,
  });

  final GlobalKey<AnimatedListState> listKey;
  final List<_ToastEntry> toasts;
  final void Function(String id) onDismiss;

  @override
  Widget build(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    final isDesktop = mediaQuery.size.width >= 768;
    final bottomPadding = mediaQuery.padding.bottom + (isDesktop ? 24 : 16);

    return Positioned(
      bottom: bottomPadding,
      right: isDesktop ? 24 : 16,
      left: isDesktop ? null : 16,
      child: Material(
        type: MaterialType.transparency,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: isDesktop ? 400 : double.infinity,
            maxHeight: mediaQuery.size.height * 0.7,
          ),
          child: AnimatedList(
            key: listKey,
            shrinkWrap: true,
            reverse: true,
            physics: const NeverScrollableScrollPhysics(),
            initialItemCount: toasts.length,
            itemBuilder: (context, index, animation) {
              if (index >= toasts.length) return const SizedBox.shrink();
              final item = toasts[index];
              return ToastManager._buildAnimatedToast(
                item,
                animation,
                () => onDismiss(item.id),
              );
            },
          ),
        ),
      ),
    );
  }
}
