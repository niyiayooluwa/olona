import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:olona/features/auth/data/models/register.dart';

part 'verify.freezed.dart';
part 'verify.g.dart';

@freezed
abstract class VerifyRequest with _$VerifyRequest {
  const factory VerifyRequest({required String code, required String email}) =
      _VerifyRequest;

  factory VerifyRequest.fromJson(Map<String, dynamic> json) =>
      _$VerifyRequestFromJson(json);
}

@freezed
abstract class VerifyResponse with _$VerifyResponse {
  const factory VerifyResponse({required TokenPair data}) = _VerifyResponse;

  factory VerifyResponse.fromJson(Map<String, dynamic> json) =>
      _$VerifyResponseFromJson(json);
}

@freezed
abstract class ResendOtpRequest with _$ResendOtpRequest {
  const factory ResendOtpRequest({required String email}) = _ResendOtpRequest;

  factory ResendOtpRequest.fromJson(Map<String, dynamic> json) =>
      _$ResendOtpRequestFromJson(json);
}
