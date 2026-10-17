import 'dart:async';

import 'package:dio/dio.dart';
import 'package:go_router/go_router.dart';
import 'package:olona/core/constants/constants.dart';
import 'package:olona/core/routing/router.dart';
import 'package:olona/core/storage/secure_storage.dart';

class AuthInterceptor extends Interceptor {
  final SecureStorage _storage;

  AuthInterceptor(this._storage);

  // A bare Dio instance used exclusively for the refresh call.
  // No interceptors attached — avoids circular dependency with the main Dio
  // instance.
  final _refreshDio = Dio(
    BaseOptions(
      baseUrl: ApiConstants.baseUrl,
      connectTimeout: ApiConstants.connectTimeout,
      receiveTimeout: ApiConstants.receiveTimeout,
      headers: {'Content-Type': 'application/json'},
    ),
  );

  // Lock to prevent multiple concurrent refresh calls when several requests
  // fail with 401 at the same time (e.g. app resumes after token expiry).
  bool _isRefreshing = false;
  final _pendingRequests = <Completer<String>>[];

  static const _bypassEndpoints = [
    '/auth/login',
    '/auth/signup',
    '/auth/oauth/google',
    '/auth/verify',
    '/auth/refresh',
    '/auth/resend-otp',
  ];

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    if (_bypassEndpoints.contains(options.path)) {
      return handler.next(options);
    }

    final token = await _storage.getAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }

    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response?.statusCode != 401) {
      return handler.next(err);
    }

    // If the 401 came from the refresh endpoint itself, the refresh token is dead.
    // Clear everything and send the user to login.
    if (err.requestOptions.path == '/auth/refresh') {
      await _clearAndRedirect();
      return handler.next(err);
    }

    // If a refresh is already in progress, queue this request until it resolves.
    if (_isRefreshing) {
      final completer = Completer<String>();
      _pendingRequests.add(completer);

      try {
        final newToken = await completer.future;
        final retryOptions = _buildRetry(err.requestOptions, newToken);
        final response = await _refreshDio.fetch(retryOptions);
        return handler.resolve(response);
      } catch (_) {
        return handler.next(err);
      }
    }

    // Start the refresh flow
    _isRefreshing = true;

    try {
      final refreshToken = await _storage.getRefreshToken();
      if (refreshToken == null) {
        await _clearAndRedirect();
        return handler.next(err);
      }

      final response = await _refreshDio.post(
        '/auth/refresh',
        data: {'refresh_token': refreshToken},
      );

      final newAccessToken = response.data['access_token'] as String;
      final newRefreshToken = response.data['refresh_token'] as String;

      // Save the fresh token pair
      await _storage.saveTokens(
        accessToken: newAccessToken,
        refreshToken: newRefreshToken,
      );

      // Unblock all queued requests with the new token
      for (final completer in _pendingRequests) {
        completer.complete(newAccessToken);
      }
      _pendingRequests.clear();
      _isRefreshing = false; // Release lock before retrying

      // Retry the original failed request with the new token
      final retryOptions = _buildRetry(err.requestOptions, newAccessToken);

      try {
        final retryResponse = await _refreshDio.fetch(retryOptions);
        return handler.resolve(retryResponse);
      } on DioException catch (retryErr) {
        return handler.next(retryErr);
      }
    } on DioException catch (refreshErr) {
      // Refresh failed — reject all queued requests
      for (final completer in _pendingRequests) {
        completer.completeError('refresh_failed');
      }
      _pendingRequests.clear();
      _isRefreshing = false;

      // Only log out if refresh token is genuinely revoked/expired (401 / 403)
      final status = refreshErr.response?.statusCode;
      if (status == 401 || status == 403) {
        await _clearAndRedirect();
      }
      return handler.next(err);
    } catch (_) {
      for (final completer in _pendingRequests) {
        completer.completeError('refresh_failed');
      }
      _pendingRequests.clear();
      _isRefreshing = false;
      return handler.next(err);
    }
  }

  /// Clones the original request options with a fresh Authorization header.
  RequestOptions _buildRetry(RequestOptions original, String newToken) {
    return original.copyWith(
      headers: {...original.headers, 'Authorization': 'Bearer $newToken'},
    );
  }

  /// Clears tokens from storage and navigates to the login screen.
  Future<void> _clearAndRedirect() async {
    await _storage.clearTokens();

    final context = navigatorKey.currentContext;
    if (context != null && context.mounted) {
      GoRouter.of(context).go('/login');
    }
  }
}
