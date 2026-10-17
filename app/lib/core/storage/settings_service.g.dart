// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'settings_service.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(settingsService)
const settingsServiceProvider = SettingsServiceProvider._();

final class SettingsServiceProvider
    extends
        $FunctionalProvider<
          AsyncValue<SettingsService>,
          SettingsService,
          FutureOr<SettingsService>
        >
    with $FutureModifier<SettingsService>, $FutureProvider<SettingsService> {
  const SettingsServiceProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'settingsServiceProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$settingsServiceHash();

  @$internal
  @override
  $FutureProviderElement<SettingsService> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<SettingsService> create(Ref ref) {
    return settingsService(ref);
  }
}

String _$settingsServiceHash() => r'df20ada6b4c41caadb70cd61c99dd6b5af85b9b2';
