import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'secure_storage.g.dart';

// The secure storage class. Very secure.The design is very human...lol
class SecureStorage {
  // Use a constant instance to avoid recreation overhead
  final _storage = const FlutterSecureStorage();

  // refresh token and access token are JWT tokens that are received and should be
  // stored upon successful authentiaction
  static const refreshTokenKey = 'refresh_token';
  static const accessTokenKey = 'access_token';
  static const fcmTokenKey = 'fcm_token';
  String? _refreshTokenCache; // In-memory cache (instance variable)
  String? _accessTokenCache; // In-memory cache (instance variable)

  // Saves the tokens to secure storage service instance upon call
  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    _accessTokenCache = accessToken;
    _refreshTokenCache = refreshToken;
    await _storage.write(key: refreshTokenKey, value: refreshToken);
    await _storage.write(key: accessTokenKey, value: accessToken);
  }

  // returns the access token to caller
  Future<String?> getAccessToken() async {
    _accessTokenCache ??= await _storage.read(key: accessTokenKey);
    return _accessTokenCache;
  }

  // returns the access toke to the caller
  Future<String?> getRefreshToken() async {
    _refreshTokenCache ??= await _storage.read(key: refreshTokenKey);
    return _refreshTokenCache;
  }

  // Clears token. Done when the refresh token expires or when the user logs out
  Future<void> clearTokens() async {
    _refreshTokenCache = null;
    _accessTokenCache = null;
    await _storage.delete(key: accessTokenKey);
    await _storage.delete(key: refreshTokenKey);
    await _storage.delete(key: fcmTokenKey);
  }
}

@Riverpod(keepAlive: true)
SecureStorage secureStorage(Ref ref) => SecureStorage();
