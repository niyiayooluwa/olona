import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:flutter_spinkit/flutter_spinkit.dart';
import 'package:go_router/go_router.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/utils/validators.dart';
import 'package:olona/features/auth/ui/login/controllers/login_controller.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

class LoginForm extends HookConsumerWidget {
  const LoginForm({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final shad = ShadTheme.of(context);

    // 1. Local controllers (auto-disposed by hooks!)
    final emailController = useTextEditingController();
    final passwordController = useTextEditingController();
    final passwordIsObscured = useState<bool>(true);
    final formKey = useMemoized(() => GlobalKey<ShadFormState>());

    // 2. Watch backend state (loading, error, etc.)
    final registerState = ref.watch(loginControllerProvider);
    final isLoading = registerState.isLoading;

    void onSubmit() async {
      if (formKey.currentState?.saveAndValidate() ?? false) {
        await ref
            .read(loginControllerProvider.notifier)
            .login(
              email: emailController.text.trim(),
              password: passwordController.text,
            );
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
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
                //leading: const Icon(LucideIcons.mail, size: 16),
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
                //leading: const Icon(LucideIcons.lock),
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
                textInputAction: TextInputAction.done,
                enabled: !isLoading,
                validator: (v) {
                  if (v.isEmpty || v.length < 6) {
                    return 'Please input a valid password';
                  }
                  return null;
                },
              ),
              const SizedBox(height: UIConstants.sm),

              Align(
                alignment: AlignmentGeometry.centerRight,
                child: ShadButton.ghost(
                  size: ShadButtonSize.sm,
                  trailing: Icon(LucideIcons.arrowRight),
                  enabled: !isLoading,
                  onPressed: () => context.goNamed('forgot-password'),
                  child: const Text('Forgot Password'),
                ),
              ),
              const SizedBox(height: UIConstants.lg),

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
                    : const Text('Login'),
              ),
              SizedBox(height: UIConstants.xl),
            ],
          ),
        ),
      ],
    );
  }
}
