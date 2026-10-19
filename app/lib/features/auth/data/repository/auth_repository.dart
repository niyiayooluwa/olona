import 'package:dart_either/dart_either.dart';
import 'package:olona/core/errors/error_mapper.dart';
import 'package:olona/features/auth/data/models/register.dart';
import 'package:olona/features/auth/data/remote/auth_remote.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_repository.g.dart';

class AuthRepository {
  final AuthRemote _remote;

  AuthRepository(this._remote);

  Future<Either<Failure, RegisterResponse>> register(
    RegisterRequest request,
  ) async {
    // No tokens to save on signup (requires OTP verification first)
    return _remote.signup(request);
  }
}

@riverpod
AuthRepository authRepository(Ref ref) {
  return AuthRepository(
    ref.watch(authRemoteProvider),
    //ref.watch(secureStorageProvider),
  );
}
