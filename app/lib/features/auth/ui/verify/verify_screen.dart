import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:flutter_spinkit/flutter_spinkit.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/presentation/toast/toast_extensions.dart';
import 'package:olona/features/auth/shared/presentation/auth_layout.dart';
import 'package:olona/features/auth/ui/register/controller/register_controller.dart';
import 'package:olona/features/auth/ui/verify/controllers/verify_controller.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class VerifyScreen extends HookConsumerWidget {
  const VerifyScreen({super.key, this.email = ''});

  final String email;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Resolve email: passed via prop, extra, or registeredEmailProvider
    final registeredEmail = ref.watch(registeredEmailProvider);
    final targetEmail = email.isNotEmpty
        ? email
        : registeredEmail.isNotEmpty
        ? registeredEmail
        : 'your email address';

    // Resend cooldown timer hook state
    final cooldownSeconds = useState<int>(0);
    final isResending = useState<bool>(false);

    useEffect(() {
      Timer? timer;
      if (cooldownSeconds.value > 0) {
        timer = Timer.periodic(const Duration(seconds: 1), (t) {
          if (cooldownSeconds.value > 0) {
            cooldownSeconds.value -= 1;
          } else {
            t.cancel();
          }
        });
      }
      return () => timer?.cancel();
    }, [cooldownSeconds.value]);

    // LISTEN for verification state changes
    ref.listen<AsyncValue<void>>(verifyControllerProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          context.showFailureToast(error);
        },
        data: (_) {
          context.showSuccessToast(
            title: 'Account Verified',
            description: 'Your account has been verified successfully.',
          );
          // Navigate to setup or home route
          context.go('/set-username');
        },
      );
    });

    return Scaffold(
      body: AuthResponsiveLayout(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _header(context, targetEmail),
            _form(
              context,
              ref,
              targetEmail: targetEmail,
              cooldownSeconds: cooldownSeconds,
              isResending: isResending,
            ),
            _footer(context),
          ],
        ),
      ),
    );
  }

  Widget _header(BuildContext context, String targetEmail) {
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
          'Verify your email',
          style: shad.textTheme.h3,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.md),
        Text.rich(
          TextSpan(
            text: 'Enter the code sent to ',
            style: shad.textTheme.muted,
            children: [
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
        const SizedBox(height: UIConstants.lg),
      ],
    );
  }

  Widget _form(
    BuildContext context,
    WidgetRef ref, {
    required String targetEmail,
    required ValueNotifier<int> cooldownSeconds,
    required ValueNotifier<bool> isResending,
  }) {
    final shad = ShadTheme.of(context);
    final otpController = useTextEditingController();
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

    final verifyState = ref.watch(verifyControllerProvider);
    final isLoading = verifyState.isLoading;

    void submitVerification([String? code]) async {
      final finalCode = (code ?? otpController.text).trim();
      if (finalCode.length < 6) return;

      await ref
          .read(verifyControllerProvider.notifier)
          .verifyOtp(email: targetEmail, code: finalCode);
    }

    void handleResend() async {
      if (cooldownSeconds.value > 0 || isResending.value) return;

      isResending.value = true;
      try {
        await ref
            .read(verifyControllerProvider.notifier)
            .resendOtp(targetEmail);
        cooldownSeconds.value = 60; // 60s cooldown
        if (context.mounted) {
          context.showSuccessToast(
            title: 'Code Sent',
            description:
                'A new 6-digit verification code has been sent to your mail.',
          );
        }
      } catch (e) {
        if (context.mounted) {
          context.showFailureToast(e);
        }
      } finally {
        isResending.value = false;
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Center(
          child: ShadInputOTPFormField(
            id: 'otp',
            gap: 8,
            onChanged: (v) {
              otpController.text = v;
              if (v.replaceAll(RegExp(r'\s+'), '').length == 6) {
                submitVerification(v.replaceAll(RegExp(r'\s+'), ''));
              }
            },
            maxLength: 6,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            validator: (v) {
              if (v.replaceAll(RegExp(r'\s+'), '').length < 6) {
                return 'Enter the full 6-digit code';
              }
              return null;
            },
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
        if (isLoading) ...[
          const SizedBox(height: UIConstants.lg),
          Center(
            child: SpinKitThreeBounce(
              size: 16,
              color: shad.colorScheme.primary,
            ),
          ),
        ],
        const SizedBox(height: UIConstants.xl),

        // Resend countdown / action area matching WorkOS exactly
        Center(
          child: Column(
            children: [
              Text(
                "Didn't receive a code?",
                style: shad.textTheme.muted.copyWith(
                  fontSize: 14,
                  color: shad.colorScheme.mutedForeground,
                ),
              ),
              const SizedBox(height: 6),
              if (cooldownSeconds.value > 0)
                Text(
                  'You can request a new one in ${cooldownSeconds.value}',
                  style: shad.textTheme.muted.copyWith(
                    fontSize: 14,
                    color: shad.colorScheme.mutedForeground,
                  ),
                )
              else if (isResending.value)
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    SpinKitThreeBounce(
                      size: 10,
                      color: shad.colorScheme.primary,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      'Sending code...',
                      style: shad.textTheme.muted.copyWith(fontSize: 14),
                    ),
                  ],
                )
              else
                GestureDetector(
                  onTap: handleResend,
                  child: Text(
                    'Resend now',
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      color: shad.colorScheme.primary,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _footer(BuildContext context) {
    final shad = ShadTheme.of(context);

    return Column(
      children: [
        const SizedBox(height: UIConstants.xxl),
        GestureDetector(
          onTap: () {
            context.goNamed('register');
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
