import 'package:freezed_annotation/freezed_annotation.dart';

part 'error_response.freezed.dart';
part 'error_response.g.dart';

@freezed
abstract class ApiErrorResponse with _$ApiErrorResponse {
  const factory ApiErrorResponse({required ApiErrorBody error}) =
      _ApiErrorResponse;
  factory ApiErrorResponse.fromJson(Map<String, dynamic> json) =>
      _$ApiErrorResponseFromJson(json);
}

@freezed
abstract class ApiErrorBody with _$ApiErrorBody {
  const factory ApiErrorBody({
    required String code,
    required String message,
    List<FieldError>? details, // omitted unless validation failed
  }) = _ApiErrorBody;
  factory ApiErrorBody.fromJson(Map<String, dynamic> json) =>
      _$ApiErrorBodyFromJson(json);
}

@freezed
abstract class FieldError with _$FieldError {
  const factory FieldError({required String field, required String code}) =
      _FieldError;
  factory FieldError.fromJson(Map<String, dynamic> json) =>
      _$FieldErrorFromJson(json);
}
