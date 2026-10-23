import 'dart:async';
import 'package:olona/features/auth/data/models/login.dart';
import 'package:olona/features/auth/data/models/verify.dart';
import 'package:olona/features/auth/data/repository/auth_repository.dart';
import 'package:olona/features/auth/ui/register/controller/register_controller.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'login_controller.g.dart';

@riverpod
class LoginController extends _$LoginController {
  @override
  FutureOr<void> build() => null;

  Future<void> login({required String email, required String password}) async {
    if (state.isLoading) return;

    state = const AsyncValue.loading();
    // Save the attempted email so VerifyScreen knows which email to display
    ref.read(registeredEmailProvider.notifier).setEmail(email);

    final repo = ref.read(authRepositoryProvider);

    final result = await repo.login(
      LoginRequest(email: email, password: password),
    );

    state = result.fold(
      ifLeft: (failure) {
        if (failure.code == 'email_not_verified') {
          repo.resendOtp(ResendOtpRequest(email: email));
        }
        return AsyncValue.error(failure, StackTrace.current);
      },
      ifRight: (response) => const AsyncValue.data(null),
    );
  }
}
