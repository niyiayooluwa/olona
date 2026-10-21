import 'package:flutter/material.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/presentation/window_size.dart';

class AuthResponsiveLayout extends StatelessWidget {
  const AuthResponsiveLayout({super.key, required this.child, this.sidePanel});

  final Widget child;
  final Widget? sidePanel;
  @override
  Widget build(BuildContext context) {
    final isDesktop =
        windowSizeFor(MediaQuery.sizeOf(context).width) != WindowSize.compact;

    if (!isDesktop) {
      return SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(UIConstants.lg),
          child: child,
        ),
      );
    }

    return Row(
      children: [
        Expanded(flex: 4, child: _defaultPanel()),

        const SizedBox(width: UIConstants.lg),
        Expanded(
          flex: 6,
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: UIConstants.xl),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 400),
                child: child,
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _defaultPanel() {
    return Container(color: Colors.green);
  }
}
