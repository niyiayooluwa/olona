import 'package:dio/dio.dart';
import 'package:dart_either/dart_either.dart';
import 'package:olona/core/errors/error_mapper.dart';
import 'package:olona/core/network/dio_client.dart';
import 'package:olona/features/auth/data/models/register.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_remote.g.dart';

class AuthRemote {
  final Dio _dio;

  AuthRemote(this._dio);

  Future<Either<Failure, RegisterResponse>> signup(
    RegisterRequest request,
  ) async {
    try {
      final response = await _dio.post('/auth/signup', data: request.toJson());
      return Right(RegisterResponse.fromJson((response.data)));
    } on DioException catch (e) {
      return Left(mapDioException(e));
    } catch (e) {
      return Left(mapException(e));
    }
  }
}

@riverpod
AuthRemote authRemote(Ref ref) {
  return AuthRemote(ref.watch(dioProvider));
}
