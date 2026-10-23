import 'package:olona/features/auth/data/models/register.dart';
import 'package:olona/features/auth/data/repository/auth_repository.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'register_controller.g.dart';

@Riverpod(keepAlive: true)
class RegisteredEmail extends _$RegisteredEmail {
  @override
  String build() => '';

  void setEmail(String email) => state = email;
}

@riverpod
class RegisterController extends _$RegisterController {
  @override
  FutureOr<void> build() => null;

  Future<void> register({
    required String email,
    required String password,
  }) async {
    if (state.isLoading) return;

    state = const AsyncValue.loading();

    final repo = ref.read(authRepositoryProvider);

    final result = await repo.register(
      RegisterRequest(email: email, password: password),
    );

    state = result.fold(
      ifLeft: (failure) => AsyncValue.error(failure, StackTrace.current),
      // We do not fetch user here since signup doesn't log them in automatically
      // (usually requires OTP verification first based on swagger)
      ifRight: (response) {
        ref.read(registeredEmailProvider.notifier).setEmail(email);
        return const AsyncValue.data(null);
      },
    );
  }
}
