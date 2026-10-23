// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'verify_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(VerifyController)
const verifyControllerProvider = VerifyControllerProvider._();

final class VerifyControllerProvider
    extends $AsyncNotifierProvider<VerifyController, void> {
  const VerifyControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'verifyControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$verifyControllerHash();

  @$internal
  @override
  VerifyController create() => VerifyController();
}

String _$verifyControllerHash() => r'761fdbbd5cde719effd196926415826aa9d05626';

abstract class _$VerifyController extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  void runBuild() {
    build();
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleValue(ref, null);
  }
}
