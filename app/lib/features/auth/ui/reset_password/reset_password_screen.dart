import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:flutter_spinkit/flutter_spinkit.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/presentation/toast/toast_extensions.dart';
import 'package:olona/features/auth/shared/presentation/auth_layout.dart';
import 'package:olona/features/auth/ui/register/widgets/password_requirements_view.dart';
import 'package:olona/features/auth/ui/reset_password/controllers/reset_password_controller.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class ResetPasswordScreen extends HookConsumerWidget {
  const ResetPasswordScreen({super.key, this.email = ''});

  final String email;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Resolve email from prop or global reset email provider
    final storedEmail = ref.watch(resetPasswordEmailProvider);
    final targetEmail = email.isNotEmpty
        ? email
        : storedEmail.isNotEmpty
            ? storedEmail
            : 'your email address';

    // Step state: 0 = Enter 6-digit code, 1 = Create new password
    final currentStep = useState<int>(0);

    // Form inputs
    final otpController = useTextEditingController();
    final passwordController = useTextEditingController();
    final confirmPasswordController = useTextEditingController();
    final passwordIsObscured = useState<bool>(true);
    final confirmPasswordIsObscured = useState<bool>(true);

    final formKey = useMemoized(() => GlobalKey<ShadFormState>());

    // LISTEN for reset password outcome
    ref.listen<AsyncValue<void>>(resetPasswordControllerProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          context.showFailureToast(error);
        },
        data: (_) {
          context.showSuccessToast(
            title: 'Password Reset',
            description: 'Your password has been updated. Please sign in with your new password.',
          );
          context.goNamed('login');
        },
      );
    });

    return Scaffold(
      body: AuthResponsiveLayout(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _header(context, targetEmail, currentStep.value),
            AnimatedSwitcher(
              duration: const Duration(milliseconds: 250),
              switchInCurve: Curves.easeOutCubic,
              switchOutCurve: Curves.easeInCubic,
              child: currentStep.value == 0
                  ? _buildStepOneOtp(
                      context,
                      targetEmail: targetEmail,
                      otpController: otpController,
                      onNext: () => currentStep.value = 1,
                    )
                  : _buildStepTwoPassword(
                      context,
                      ref,
                      targetEmail: targetEmail,
                      otpController: otpController,
                      passwordController: passwordController,
                      confirmPasswordController: confirmPasswordController,
                      passwordIsObscured: passwordIsObscured,
                      confirmPasswordIsObscured: confirmPasswordIsObscured,
                      formKey: formKey,
                      onBack: () => currentStep.value = 0,
                    ),
            ),
            _footer(context, currentStep),
          ],
        ),
      ),
    );
  }

  Widget _header(BuildContext context, String targetEmail, int step) {
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
          step == 0 ? 'Enter verification code' : 'Create new password',
          style: shad.textTheme.h3,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.md),
        Text.rich(
          TextSpan(
            text: step == 0
                ? 'Enter the 6-digit code sent to\n'
                : 'Choose a strong password to secure your account\n',
            style: shad.textTheme.muted,
            children: [
              if (step == 0)
                TextSpan(
                  text: targetEmail,
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: shad.colorScheme.foreground,
                  ),
                ),
            ],
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.xl),
      ],
    );
  }

  // STEP 1: Enter 6-digit OTP code
  Widget _buildStepOneOtp(
    BuildContext context, {
    required String targetEmail,
    required TextEditingController otpController,
    required VoidCallback onNext,
  }) {
    final shad = ShadTheme.of(context);
    final slotRadius = BorderRadius.circular(10);
    final slotStyle = TextStyle(
      fontSize: 24,
      fontWeight: FontWeight.w700,
      color: shad.colorScheme.foreground,
    );

    ShadInputOTPSlot buildSlot() {
      return ShadInputOTPSlot(
        width: 48,
        height: 60,
        padding: const EdgeInsets.symmetric(vertical: 8),
        style: slotStyle,
        singleRadius: slotRadius,
        firstRadius: slotRadius,
        lastRadius: slotRadius,
        middleRadius: slotRadius,
        decoration: ShadDecoration(
          border: ShadBorder.all(
            color: shad.colorScheme.border,
            width: 1.2,
            radius: slotRadius,
          ),
          focusedBorder: ShadBorder.all(
            color: shad.colorScheme.ring,
            width: 1.8,
            radius: slotRadius,
          ),
        ),
      );
    }

    void handleContinue() {
      final code = otpController.text.replaceAll(RegExp(r'\s+'), '');
      if (code.length < 6) {
        context.showWarningToast(
          title: 'Incomplete Code',
          description: 'Please enter all 6 digits of the verification code.',
        );
        return;
      }
      onNext();
    }

    return Column(
      key: const ValueKey('step_1_otp'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Center(
          child: ShadInputOTPFormField(
            id: 'otp',
            gap: 8,
            onChanged: (v) {
              otpController.text = v;
              if (v.replaceAll(RegExp(r'\s+'), '').length == 6) {
                handleContinue();
              }
            },
            maxLength: 6,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            children: [
              ShadInputOTPGroup(children: [buildSlot()]),
              ShadInputOTPGroup(children: [buildSlot()]),
              ShadInputOTPGroup(children: [buildSlot()]),
              ShadInputOTPGroup(children: [buildSlot()]),
              ShadInputOTPGroup(children: [buildSlot()]),
              ShadInputOTPGroup(children: [buildSlot()]),
            ],
          ),
        ),
        const SizedBox(height: UIConstants.xxl),

        ShadButton(
          width: double.infinity,
          onPressed: handleContinue,
          child: const Text('Continue'),
        ),
      ],
    );
  }

  // STEP 2: New password + Confirm password
  Widget _buildStepTwoPassword(
    BuildContext context,
    WidgetRef ref, {
    required String targetEmail,
    required TextEditingController otpController,
    required TextEditingController passwordController,
    required TextEditingController confirmPasswordController,
    required ValueNotifier<bool> passwordIsObscured,
    required ValueNotifier<bool> confirmPasswordIsObscured,
    required GlobalKey<ShadFormState> formKey,
    required VoidCallback onBack,
  }) {
    final shad = ShadTheme.of(context);
    final resetState = ref.watch(resetPasswordControllerProvider);
    final isLoading = resetState.isLoading;

    void onSubmit() async {
      if (formKey.currentState?.saveAndValidate() ?? false) {
        final code = otpController.text.replaceAll(RegExp(r'\s+'), '');
        await ref.read(resetPasswordControllerProvider.notifier).resetPassword(
              email: targetEmail,
              code: code,
              password: passwordController.text,
            );
      }
    }

    return ShadForm(
      key: const ValueKey('step_2_password'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ShadForm(
            key: formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                // 1. New Password
                ShadInputFormField(
                  id: 'new_password',
                  label: const Text('New Password'),
                  controller: passwordController,
                  obscureText: passwordIsObscured.value,
                  placeholder: Text(
                    '••••••••',
                    style: shad.textTheme.muted,
                  ),
                  enabled: !isLoading,
                  trailing: ShadIconButton.ghost(
                    width: 24,
                    height: 24,
                    padding: EdgeInsets.zero,
                    icon: Icon(
                      passwordIsObscured.value
                          ? LucideIcons.eyeOff
                          : LucideIcons.eye,
                      size: 16,
                    ),
                    onPressed: () =>
                        passwordIsObscured.value = !passwordIsObscured.value,
                  ),
                  validator: (value) {
                    if (value.isEmpty) return 'Password is required';
                    if (value.length < 8) {
                      return 'Password must be at least 8 characters';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: UIConstants.sm),

                // Animated requirements checklist
                PasswordRequirementsView(controller: passwordController),
                const SizedBox(height: UIConstants.md),

                // 2. Confirm Password
                ShadInputFormField(
                  id: 'confirm_password',
                  label: const Text('Confirm Password'),
                  controller: confirmPasswordController,
                  obscureText: confirmPasswordIsObscured.value,
                  placeholder: Text(
                    '••••••••',
                    style: shad.textTheme.muted,
                  ),
                  enabled: !isLoading,
                  trailing: ShadIconButton.ghost(
                    width: 24,
                    height: 24,
                    padding: EdgeInsets.zero,
                    icon: Icon(
                      confirmPasswordIsObscured.value
                          ? LucideIcons.eyeOff
                          : LucideIcons.eye,
                      size: 16,
                    ),
                    onPressed: () => confirmPasswordIsObscured.value =
                        !confirmPasswordIsObscured.value,
                  ),
                  validator: (value) {
                    if (value.isEmpty) return 'Please confirm your password';
                    if (value != passwordController.text) {
                      return 'Passwords do not match';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: UIConstants.lg),

                // Submit button
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
                      : const Text('Set New Password'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _footer(BuildContext context, ValueNotifier<int> currentStep) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
        const SizedBox(height: UIConstants.xxl),
        GestureDetector(
          onTap: () {
            if (currentStep.value > 0) {
              currentStep.value = 0;
            } else {
              context.goNamed('login');
            }
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
                currentStep.value > 0 ? 'Back to code' : 'Back to sign-in',
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
