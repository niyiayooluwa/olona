// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_remote.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(authRemote)
const authRemoteProvider = AuthRemoteProvider._();

final class AuthRemoteProvider
    extends $FunctionalProvider<AuthRemote, AuthRemote, AuthRemote>
    with $Provider<AuthRemote> {
  const AuthRemoteProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authRemoteProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authRemoteHash();

  @$internal
  @override
  $ProviderElement<AuthRemote> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  AuthRemote create(Ref ref) {
    return authRemote(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AuthRemote value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AuthRemote>(value),
    );
  }
}

String _$authRemoteHash() => r'7bcb30ef6149e91ab5521bee836209e0941fb838';
