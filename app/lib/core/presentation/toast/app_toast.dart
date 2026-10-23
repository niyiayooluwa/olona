import 'package:flutter/material.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

/// The four semantic notification variants.
enum ToastType {
  success,
  info,
  error,
  warning,
}

/// Action item displayed in an [AppToast].
class ToastAction {
  const ToastAction({
    required this.label,
    required this.onPressed,
    this.isPrimary = true,
  });

  final String label;
  final VoidCallback onPressed;
  final bool isPrimary;
}

/// Reusable toast notification card matching the Olona UI specification:
/// - Distinct Light / Dark mode surfaces.
/// - Top-left semantic color bleed gradient fading into base background within 1/3 of the card.
/// - Top row: Leading icon, Title, and Muted 'X' dismiss button on the same horizontal level (space-between).
/// - Title, description, and action buttons share a single column.
/// - Optional side-by-side action buttons (one primary, one muted).
class AppToast extends StatelessWidget {
  const AppToast({
    super.key,
    required this.type,
    required this.title,
    this.description,
    this.actions,
    this.onDismiss,
    this.maxWidth = 420,
  });

  final ToastType type;
  final String title;
  final String? description;
  final List<ToastAction>? actions;
  final VoidCallback? onDismiss;
  final double maxWidth;

  Color _getSemanticColor(ShadColorScheme scheme) {
    switch (type) {
      case ToastType.success:
        return const Color(0xFF16A34A);
      case ToastType.info:
        return const Color(0xFF2563EB);
      case ToastType.error:
        return scheme.destructive;
      case ToastType.warning:
        return const Color(0xFFD97706);
    }
  }

  IconData _getIcon() {
    switch (type) {
      case ToastType.success:
        return Icons.check_circle_rounded;
      case ToastType.info:
        return Icons.info_rounded;
      case ToastType.error:
        return Icons.error_rounded;
      case ToastType.warning:
        return Icons.warning_rounded;
    }
  }

  @override
  Widget build(BuildContext context) {
    final shad = ShadTheme.of(context);
    final scheme = shad.colorScheme;
    final isDark = Theme.of(context).brightness == Brightness.dark;

    final semanticColor = _getSemanticColor(scheme);
    final baseCardColor = scheme.card;
    final borderColor = scheme.border;

    final hasDescription = description != null && description!.trim().isNotEmpty;
    final hasActions = actions != null && actions!.isNotEmpty;

    return ConstrainedBox(
      constraints: BoxConstraints(maxWidth: maxWidth),
      child: Container(
        decoration: BoxDecoration(
          color: baseCardColor,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: borderColor, width: 1.0),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: isDark ? 0.35 : 0.08),
              blurRadius: 16,
              offset: const Offset(0, 6),
            ),
            BoxShadow(
              color: Colors.black.withValues(alpha: isDark ? 0.20 : 0.04),
              blurRadius: 4,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        clipBehavior: Clip.antiAlias,
        child: Stack(
          children: [
            // Top-left semantic color bleed fading to base card background within 1/3 of the card
            Positioned.fill(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: const Alignment(0.4, 0.4),
                    stops: const [0.0, 0.33, 1.0],
                    colors: [
                      semanticColor.withValues(alpha: isDark ? 0.24 : 0.14),
                      semanticColor.withValues(alpha: isDark ? 0.06 : 0.03),
                      Colors.transparent,
                    ],
                  ),
                ),
              ),
            ),

            // Toast body
            Padding(
              padding: const EdgeInsets.symmetric(
                horizontal: UIConstants.md,
                vertical: UIConstants.md,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Top row: Icon, Title, and X button on the exact same vertical level
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      // Leading status icon
                      Icon(
                        _getIcon(),
                        size: 20,
                        color: semanticColor,
                      ),
                      const SizedBox(width: UIConstants.sm),

                      // Title in the same row
                      Expanded(
                        child: Text(
                          title,
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                            color: scheme.foreground,
                            height: 1.25,
                          ),
                        ),
                      ),

                      // Muted close button
                      if (onDismiss != null) ...[
                        const SizedBox(width: UIConstants.xs),
                        Semantics(
                          label: 'Dismiss notification',
                          button: true,
                          child: InkWell(
                            onTap: onDismiss,
                            borderRadius: BorderRadius.circular(4),
                            child: Padding(
                              padding: const EdgeInsets.all(2),
                              child: Icon(
                                Icons.close_rounded,
                                size: 16,
                                color: scheme.mutedForeground,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),

                  // Content column underneath the title row
                  if (hasDescription || hasActions)
                    Padding(
                      padding: const EdgeInsets.only(left: 28),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (hasDescription) ...[
                            const SizedBox(height: 4),
                            Text(
                              description!,
                              style: TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w400,
                                color: scheme.mutedForeground,
                                height: 1.4,
                              ),
                            ),
                          ],

                          // Feedback buttons side by side
                          if (hasActions) ...[
                            const SizedBox(height: UIConstants.md),
                            Row(
                              mainAxisSize: MainAxisSize.min,
                              children: actions!.take(2).map((action) {
                                if (action.isPrimary) {
                                  return Padding(
                                    padding: const EdgeInsets.only(right: 8),
                                    child: ShadButton(
                                      size: ShadButtonSize.sm,
                                      onPressed: action.onPressed,
                                      child: Text(action.label),
                                    ),
                                  );
                                } else {
                                  return ShadButton.ghost(
                                    size: ShadButtonSize.sm,
                                    onPressed: action.onPressed,
                                    child: Text(
                                      action.label,
                                      style: TextStyle(
                                        color: scheme.mutedForeground,
                                      ),
                                    ),
                                  );
                                }
                              }).toList(),
                            ),
                          ],
                        ],
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
