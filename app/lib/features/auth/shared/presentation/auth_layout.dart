import 'package:flutter/material.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class AuthResponsiveLayout extends StatelessWidget {
  const AuthResponsiveLayout({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final scheme = ShadTheme.of(context).colorScheme;

    return SafeArea(
      child: LayoutBuilder(
        builder: (context, constraints) {
          return SingleChildScrollView(
            child: ConstrainedBox(
              constraints: BoxConstraints(
                minHeight: constraints.maxHeight,
              ),
              child: IntrinsicHeight(
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: UIConstants.lg,
                    vertical: UIConstants.lg,
                  ),
                  child: Column(
                    children: [
                      const Spacer(),

                      // Form sitting directly on background (Constrained to 400px)
                      Center(
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 400),
                          child: child,
                        ),
                      ),

                      const Spacer(),

                      // Global Footer pinned cleanly to the bottom
                      _buildFooter(scheme),
                    ],
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _buildFooter(ShadColorScheme scheme) {
    return Padding(
      padding: const EdgeInsets.only(top: UIConstants.lg),
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              InkWell(
                onTap: () {},
                child: Text(
                  'Privacy Policy',
                  style: TextStyle(
                    color: scheme.mutedForeground,
                    fontSize: 12,
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 8),
                child: Text(
                  '•',
                  style: TextStyle(
                    color: scheme.mutedForeground,
                    fontSize: 12,
                  ),
                ),
              ),
              InkWell(
                onTap: () {},
                child: Text(
                  'Terms of Service',
                  style: TextStyle(
                    color: scheme.mutedForeground,
                    fontSize: 12,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            '© 2026 Olona Inc. All rights reserved.',
            style: TextStyle(
              color: scheme.mutedForeground,
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}
