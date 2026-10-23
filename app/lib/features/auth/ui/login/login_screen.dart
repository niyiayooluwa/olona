import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/errors/error_mapper.dart';
import 'package:olona/core/presentation/toast/toast_extensions.dart';
import 'package:olona/features/auth/shared/presentation/auth_layout.dart';
import 'package:olona/features/auth/ui/login/controllers/login_controller.dart';
import 'package:olona/features/auth/ui/login/widgets/login_form.dart';
import 'package:olona/features/auth/ui/register/controller/register_controller.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class LoginScreen extends HookConsumerWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // LISTEN for side effects (Toasts, Errors, Navigation)
    ref.listen<AsyncValue<void>>(loginControllerProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          if (error is Failure && error.code == 'email_not_verified') {
            final email = ref.read(registeredEmailProvider);

            // Show informative warning toast
            context.showWarningToast(
              title: 'Email not verified',
              description:
                  'We sent a new 6-digit verification code to your email.',
            );
            // Redirect to /verify with their email pre-filled
            context.goNamed('verify', extra: email);
            return;
          }
          context.showFailureToast(error);
        },
        data: (_) {
          // Success! Navigate forward
          context.showSuccessToast(title: 'Login Successful');
          context.goNamed('home');
        },
      );
    });

    return Scaffold(
      body: AuthResponsiveLayout(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [_header(context), const LoginForm(), _footer(context)],
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
          'Welcome back',
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
            text: "Don't have an account? ",
            style: shad.textTheme.muted,
            children: [
              TextSpan(
                text: 'Create one here',
                style: TextStyle(
                  color: shad.colorScheme.primary,
                  decoration: TextDecoration.underline,
                ),
                recognizer: TapGestureRecognizer()
                  ..onTap = () {
                    context.replaceNamed('register');
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
