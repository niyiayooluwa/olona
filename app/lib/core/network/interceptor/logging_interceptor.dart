import 'dart:developer';
import 'package:dio/dio.dart';

/// A Dio interceptor that logs network requests, responses, and errors.
/// This is useful for debugging network interactions using the dart:developer log
/// which avoids truncation issues common with standard print() statements.
class LoggingInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    log('REQUEST[${options.method}] => PATH: ${options.path}');
    log('Headers: ${options.headers}');
    if (options.data != null) {
      log('Request Body: ${options.data}');
    }
    super.onRequest(options, handler);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    log(
      'RESPONSE[${response.statusCode}] => PATH: ${response.requestOptions.path}',
    );
    if (response.data != null) {
      log('Response Data: ${response.data}');
    }
    super.onResponse(response, handler);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    log(
      'ERROR[${err.response?.statusCode}] => PATH: ${err.requestOptions.path}',
    );
    log('Error Message: ${err.message}');
    if (err.response != null) {
      log('Error Response Data: ${err.response?.data}');
    }
    super.onError(err, handler);
  }
}
