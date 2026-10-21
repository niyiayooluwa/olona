import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/utils/validators.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class PasswordRequirementsView extends HookWidget {
  const PasswordRequirementsView({
    super.key,
    required this.controller,
  });

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final animController = useAnimationController(
      duration: const Duration(milliseconds: 280),
    );

    final curvedAnimation = useMemoized(
      () => CurvedAnimation(
        parent: animController,
        curve: Curves.easeInOutCubicEmphasized,
      ),
      [animController],
    );

    // Drive the animation forward when text exists, reverse when empty
    useEffect(() {
      void listener() {
        if (controller.text.isNotEmpty) {
          animController.forward();
        } else {
          animController.reverse();
        }
      }

      controller.addListener(listener);
      // Run once initially in case controller already has text
      if (controller.text.isNotEmpty) animController.value = 1.0;

      return () => controller.removeListener(listener);
    }, [controller, animController]);

    final shad = ShadTheme.of(context);

    return AnimatedBuilder(
      animation: Listenable.merge([animController, controller]),
      builder: (context, _) {
        if (animController.value == 0) return const SizedBox.shrink();

        final requirements = passwordRequirements(controller.text);

        return SizeTransition(
          sizeFactor: curvedAnimation,
          axisAlignment: -1.0,
          child: FadeTransition(
            opacity: curvedAnimation,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: requirements.entries.map((entry) {
                final isMet = entry.value;
                final label = entry.key;

                return Padding(
                  padding: const EdgeInsets.only(top: UIConstants.xs),
                  child: Row(
                    children: [
                      AnimatedContainer(
                        duration: const Duration(milliseconds: 200),
                        width: 16,
                        height: 16,
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          color: isMet
                              ? shad.colorScheme.primary.withValues(alpha: 0.15)
                              : shad.colorScheme.muted,
                        ),
                        child: Center(
                          child: Icon(
                            isMet ? LucideIcons.check : LucideIcons.circle,
                            size: 10,
                            color: isMet
                                ? shad.colorScheme.primary
                                : shad.colorScheme.mutedForeground,
                          ),
                        ),
                      ),
                      const SizedBox(width: UIConstants.sm),
                      Text(
                        label,
                        style: shad.textTheme.muted.copyWith(
                          fontSize: 12,
                          color: isMet
                              ? shad.colorScheme.foreground
                              : shad.colorScheme.mutedForeground,
                          fontWeight: isMet
                              ? FontWeight.w500
                              : FontWeight.normal,
                        ),
                      ),
                    ],
                  ),
                );
              }).toList(),
            ),
          ),
        );
      },
    );
  }
}
