import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/utils/validators.dart';
import 'package:olona/features/auth/ui/register/controller/register_controller.dart';
import 'package:olona/features/auth/ui/register/widgets/password_requirements_view.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class RegisterForm extends HookConsumerWidget {
  const RegisterForm({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final shad = ShadTheme.of(context);

    // 1. Local controllers (auto-disposed by hooks!)
    final emailController = useTextEditingController();
    final passwordController = useTextEditingController();
    final passwordIsObscured = useState<bool>(true);
    final confirmPasswordController = useTextEditingController();
    final confirmIsObscured = useState<bool>(true);
    final formKey = useMemoized(() => GlobalKey<ShadFormState>());

    // 2. Watch backend state (loading, error, etc.)
    final registerState = ref.watch(registerControllerProvider);
    final isLoading = registerState.isLoading;

    void onSubmit() {
      if (formKey.currentState?.saveAndValidate() ?? false) {
        ref
            .read(registerControllerProvider.notifier)
            .register(
              email: emailController.text.trim(),
              password: passwordController.text,
            );
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          'Create your account',
          style: shad.textTheme.h1,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.sm),
        Text(
          'Start managing your business in minutes.',
          style: shad.textTheme.muted,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: UIConstants.xl),

        ShadForm(
          key: formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ShadInputFormField(
                id: 'email',
                label: const Text('Email'),
                controller: emailController,
                placeholder: Text(
                  'you@example.com',
                  style: shad.textTheme.muted,
                ),
                leading: const Icon(LucideIcons.mail, size: 16),
                keyboardType: TextInputType.emailAddress,
                textInputAction: TextInputAction.next,
                enabled: !isLoading,
                validator: validateEmail,
              ),
              const SizedBox(height: UIConstants.md),

              ShadInputFormField(
                id: 'password',
                label: const Text('Password'),
                controller: passwordController,
                placeholder: Text('••••••••••', style: shad.textTheme.muted),
                obscureText: passwordIsObscured.value,
                leading: const Icon(LucideIcons.lock),
                trailing: InkWell(
                  child: Icon(
                    passwordIsObscured.value
                        ? LucideIcons.eye
                        : LucideIcons.eyeOff,
                    size: UIConstants.md,
                  ),
                  onTap: () {
                    passwordIsObscured.value = !passwordIsObscured.value;
                  },
                ),
                textInputAction: TextInputAction.next,
                enabled: !isLoading,
                validator: validatePassword,
              ),
              const SizedBox(height: UIConstants.sm),
              PasswordRequirementsView(controller: passwordController),
              const SizedBox(height: UIConstants.md),

              ShadInputFormField(
                id: 'confirm_password',
                label: const Text('Confirm Password'),
                controller: confirmPasswordController,
                placeholder: Text('••••••••••', style: shad.textTheme.muted),
                obscureText: confirmIsObscured.value,
                leading: Icon(LucideIcons.lock),
                trailing: InkWell(
                  child: Icon(
                    confirmIsObscured.value
                        ? LucideIcons.eye
                        : LucideIcons.eyeOff,
                    size: UIConstants.md,
                  ),
                  onTap: () {
                    confirmIsObscured.value = !confirmIsObscured.value;
                  },
                ),
                textInputAction: TextInputAction.done,
                enabled: !isLoading,
                validator: (v) {
                  if (v.isEmpty) return 'Please confirm your password';
                  if (v != passwordController.text) {
                    return 'Passwords do not match';
                  }
                  return null;
                },
              ),
              const SizedBox(height: UIConstants.lg),

              ShadButton(
                width: double.infinity,
                enabled: !isLoading,
                onPressed: isLoading ? null : onSubmit,
                child: isLoading
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Create account'),
              ),
              SizedBox(height: UIConstants.xl),

              Text.rich(
                TextSpan(
                  text: "Already have an account? ",
                  style: shad.textTheme.muted,
                  children: [
                    TextSpan(
                      text: 'Sign in here',
                      style: TextStyle(color: shad.colorScheme.primary),
                      recognizer: TapGestureRecognizer()
                        ..onTap = () {
                          context.replace('/login');
                        },
                    ),
                  ],
                ),
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ],
    );
  }
}
