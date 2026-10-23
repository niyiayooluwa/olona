import 'package:dart_either/dart_either.dart';
import 'package:olona/core/errors/error_mapper.dart';
import 'package:olona/core/storage/secure_storage.dart';
import 'package:olona/features/auth/data/models/register.dart';
import 'package:olona/features/auth/data/models/verify.dart';
import 'package:olona/features/auth/data/remote/auth_remote.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_repository.g.dart';

class AuthRepository {
  final AuthRemote _remote;
  final SecureStorage _storage;

  AuthRepository(this._remote, this._storage);

  Future<Either<Failure, RegisterResponse>> register(
    RegisterRequest request,
  ) async {
    // No tokens to save on signup (requires OTP verification first)
    return _remote.signup(request);
  }

  Future<Either<Failure, VerifyResponse>> verifyOtp(
    VerifyRequest request,
  ) async {
    final result = await _remote.verifyOtp(request);

    return result.fold(
      ifLeft: (failure) => Left(failure),
      ifRight: (response) async {
        await _storage.saveTokens(
          accessToken: response.data.accessToken,
          refreshToken: response.data.refreshToken,
        );
        return Right(response);
      },
    );
  }
}

@riverpod
AuthRepository authRepository(Ref ref) {
  return AuthRepository(
    ref.watch(authRemoteProvider),
    ref.watch(secureStorageProvider),
    //ref.watch(secureStorageProvider),
  );
}
