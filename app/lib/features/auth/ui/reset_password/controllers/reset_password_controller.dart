import 'dart:async';

import 'package:olona/features/auth/data/models/forgot_password.dart';
import 'package:olona/features/auth/data/repository/auth_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'reset_password_controller.g.dart';

@riverpod
class ResetPasswordController extends _$ResetPasswordController {
  @override
  FutureOr<void> build() => null;

  Future<void> forgotPassword({required String email}) async {
    if (state.isLoading) return;

    state = const AsyncValue.loading();

    final repo = ref.read(authRepositoryProvider);
    final result = await repo.forgotPassword(
      ForgotPasswordRequest(email: email),
    );

    state = result.fold(
      ifLeft: (failure) => AsyncValue.error(failure, StackTrace.current),
      ifRight: (response) => const AsyncValue.data(null),
    );
  }

  Future<void> resetPassword({
    required String email,
    required String password,
    required String code,
  }) async {
    if (state.isLoading) return;

    state = const AsyncValue.loading();

    final repo = ref.read(authRepositoryProvider);
    final result = await repo.resetPassword(
      ResetPasswordRequest(email: email, code: code, newPassword: password),
    );

    state = result.fold(
      ifLeft: (failure) => AsyncValue.error(failure, StackTrace.current),
      ifRight: (response) => const AsyncValue.data(null),
    );
  }
}
