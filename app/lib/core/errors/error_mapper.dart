import 'package:dio/dio.dart';
import 'package:olona/features/auth/data/models/error_response.dart';

/// Anything the app can fail with. `code` is the server's code when the
/// server answered, or a local one ('network', 'timeout', 'unexpected').
class Failure {
  const Failure({
    required this.code,
    required this.message,
    this.statusCode,
    this.details,
  });

  final String code;
  final String message;
  final int? statusCode;
  final List<FieldError>? details;

  @override
  String toString() => message;
}

Failure mapDioException(DioException e) {
  final response = e.response;

  // Server answered -> it sent the error envelope.
  if (response != null && response.data is Map) {
    try {
      final envelope = ApiErrorResponse.fromJson(
        (response.data as Map).cast<String, dynamic>(),
      );
      return Failure(
        code: envelope.error.code,
        message: envelope.error.message,
        statusCode: response.statusCode,
        details: envelope.error.details,
      );
    } catch (_) {
      // Non-2xx but not our shape (proxy/gateway). Don't explode.
      return Failure(
        code: 'unexpected',
        message: 'Something went wrong.',
        statusCode: response.statusCode,
      );
    }
  }

  // No response -> transport problem.
  return switch (e.type) {
    DioExceptionType.connectionTimeout ||
    DioExceptionType.receiveTimeout ||
    DioExceptionType.sendTimeout =>
      const Failure(code: 'timeout', message: 'The request timed out.'),
    DioExceptionType.connectionError =>
      const Failure(code: 'network', message: 'Please check your connection.'),
    DioExceptionType.cancel =>
      const Failure(code: 'cancelled', message: 'Request was cancelled.'),
    _ => const Failure(code: 'unexpected', message: 'Something went wrong.'),
  };
}

Failure mapException(Object error) =>
    const Failure(code: 'unexpected', message: 'Something went wrong.');