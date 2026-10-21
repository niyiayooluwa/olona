import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/features/auth/shared/presentation/auth_layout.dart';
import 'package:olona/features/auth/ui/register/controller/register_controller.dart';
import 'package:olona/features/auth/ui/register/widgets/register_form.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class RegisterScreen extends HookConsumerWidget {
  const RegisterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // LISTEN for side effects (Toasts, Errors, Navigation)
    ref.listen<AsyncValue<void>>(registerControllerProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          // Show error toast using Shadcn UI or SnackBar
          ShadToaster.of(context).show(
            ShadToast.destructive(
              title: const Text('Registration Failed'),
              description: Text(error.toString()),
            ),
          );
        },
        data: (_) {
          // Success! Navigate forward
          context.go('/verify-email');
        },
      );
    });

    return const Scaffold(body: AuthResponsiveLayout(child: RegisterForm()));
  }
}
