import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/presentation/toast/toast_extensions.dart';
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
          context.showFailureToast(error);
        },
        data: (_) {
          // Success! Navigate forward
          final email = ref.read(registeredEmailProvider);
          context.showSuccessToast(
            title: 'Account Created',
            description: 'Please check your email for the verification code.',
          );
          context.goNamed('verify', extra: email);
        },
      );
    });

    return Scaffold(
      body: AuthResponsiveLayout(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [_header(context), const RegisterForm(), _footer(context)],
        ),
      ),
    );
  }

  Widget _header(BuildContext context) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
        // Short logo mark paired directly with the header
        SvgPicture.asset(
          'assets/svgs/logo/black.svg',
          height: UIConstants.xxl,
          colorFilter: ColorFilter.mode(
            shad.colorScheme.foreground,
            BlendMode.srcIn,
          ),
        ),
        const SizedBox(height: UIConstants.lg),
        Text(
          'Create your account',
          style: shad.textTheme.h2,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.xs),
        Text(
          'Start managing your business in minutes.',
          style: shad.textTheme.muted,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.xl),
      ],
    );
  }

  Widget _footer(BuildContext context) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
        Text.rich(
          TextSpan(
            text: "Already have an account? ",
            style: shad.textTheme.muted,
            children: [
              TextSpan(
                text: 'Sign in here',
                style: TextStyle(
                  color: shad.colorScheme.primary,
                  decoration: TextDecoration.underline,
                ),
                recognizer: TapGestureRecognizer()
                  ..onTap = () {
                    context.replaceNamed('login');
                  },
              ),
            ],
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.xl),
      ],
    );
  }
}
