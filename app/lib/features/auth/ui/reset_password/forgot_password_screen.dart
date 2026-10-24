import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:flutter_spinkit/flutter_spinkit.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/presentation/toast/toast_extensions.dart';
import 'package:olona/core/utils/validators.dart';
import 'package:olona/features/auth/shared/presentation/auth_layout.dart';
import 'package:olona/features/auth/ui/reset_password/controllers/reset_password_controller.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class ForgotPasswordScreen extends HookConsumerWidget {
  const ForgotPasswordScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.listen<AsyncValue<void>>(resetPasswordControllerProvider, (
      previous,
      next,
    ) {
      next.whenOrNull(
        error: (error, _) {
          context.showFailureToast(error);
        },
        data: (_) {
          final email = ref.read(resetPasswordEmailProvider);
          context.showSuccessToast(
            title: 'Code Sent',
            description: 'If this email has an account, an OTP has been sent.',
          );
          context.goNamed('reset-password', extra: email);
        },
      );
    });

    return Scaffold(
      body: AuthResponsiveLayout(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [_header(context), _form(context, ref), _footer(context)],
        ),
      ),
    );
  }

  Widget _header(BuildContext context) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
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
          'Forgot Password',
          style: shad.textTheme.h3,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.md),
        Text(
          "No worries! Enter your email address below, and we'll\nsend a code to reset your password",
          style: shad.textTheme.muted,

          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.lg),
      ],
    );
  }

  Widget _form(BuildContext context, WidgetRef ref) {
    final emailController = useTextEditingController();
    final formKey = useMemoized(() => GlobalKey<ShadFormState>());

    // Watch backend state (loading, error, etc.)
    final registerState = ref.watch(resetPasswordControllerProvider);
    final isLoading = registerState.isLoading;

    final shad = ShadTheme.of(context);

    void onSubmit() async {
      if (formKey.currentState?.saveAndValidate() ?? false) {
        await ref
            .read(resetPasswordControllerProvider.notifier)
            .forgotPassword(email: emailController.text.trim());
      }
    }

    return ShadForm(
      key: formKey,
      child: Column(
        children: [
          ShadInputFormField(
            id: 'email',
            placeholder: const Text('name@example.com'),
            controller: emailController,
            keyboardType: TextInputType.emailAddress,
            validator: validateEmail,
          ),
          const SizedBox(height: 32),

          ShadButton(
            width: double.infinity,
            enabled: !isLoading,
            onPressed: isLoading ? null : onSubmit,
            child: isLoading
                ? SizedBox(
                    child: SpinKitThreeBounce(
                      size: 16,
                      color: shad.colorScheme.accent,
                    ),
                  )
                : const Text('Reset Password'),
          ),
        ],
      ),
    );
  }

  Widget _footer(BuildContext context) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
        const SizedBox(height: UIConstants.xxl),
        GestureDetector(
          onTap: () {
            context.goNamed('login');
          },
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.chevron_left_rounded,
                size: 18,
                color: shad.colorScheme.mutedForeground,
              ),
              const SizedBox(width: 4),
              Text(
                'Back to sign-in',
                style: shad.textTheme.muted.copyWith(
                  fontSize: 14,
                  color: shad.colorScheme.mutedForeground,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: UIConstants.lg),
      ],
    );
  }
}
