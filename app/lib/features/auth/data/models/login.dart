import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:olona/features/auth/data/models/register.dart';

part 'login.freezed.dart';
part 'login.g.dart';

@freezed
abstract class LoginRequest with _$LoginRequest {
  const factory LoginRequest({
    required String email,
    required String password,
  }) = _LoginRequest;

  factory LoginRequest.fromJson(Map<String, dynamic> json) =>
      _$LoginRequestFromJson(json);
}

@freezed
abstract class LoginResponse with _$LoginResponse {
  const factory LoginResponse({required TokenPair data}) = _LoginResponse;

  factory LoginResponse.fromJson(Map<String, dynamic> json) =>
      _$LoginResponseFromJson(json);
}
