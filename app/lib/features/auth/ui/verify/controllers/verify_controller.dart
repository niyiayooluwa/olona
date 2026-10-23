import 'dart:async';
import 'package:olona/features/auth/data/models/verify.dart';
import 'package:olona/features/auth/data/repository/auth_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'verify_controller.g.dart';

@riverpod
class VerifyController extends _$VerifyController {
  @override
  FutureOr<void> build() => null;

  Future<void> verifyOtp({required String email, required String code}) async {
    if (state.isLoading) return;

    state = const AsyncValue.loading();

    final repo = ref.read(authRepositoryProvider);

    final result = await repo.verifyOtp(
      VerifyRequest(code: code, email: email),
    );

    state = result.fold(
      ifLeft: (failure) => AsyncValue.error(failure, StackTrace.current),
      ifRight: (response) => const AsyncValue.data(null),
    );
  }

  Future<void> resendOtp(String email) async {
    final repo = ref.read(authRepositoryProvider);
    final result = await repo.resendOtp(ResendOtpRequest(email: email));
    result.fold(
      ifLeft: (failure) => throw failure,
      ifRight: (_) => null,
    );
  }
}
