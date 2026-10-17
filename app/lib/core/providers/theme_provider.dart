import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:olona/core/storage/settings_service.dart';

part 'theme_provider.g.dart';

@Riverpod(keepAlive: true)
class ThemeModeNotifier extends _$ThemeModeNotifier {
  @override
  ThemeMode build() {
    final settingsAsync = ref.watch(settingsServiceProvider);

    return settingsAsync.maybeWhen(
      data: (settings) {
        final modeString = settings.themeMode;
        if (modeString == 'light') return ThemeMode.light;
        if (modeString == 'dark') return ThemeMode.dark;
        return ThemeMode.system;
      },
      orElse: () => ThemeMode.system,
    );
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    state = mode;
    final settings = await ref.read(settingsServiceProvider.future);
    await settings.setThemeMode(mode.name);
  }
}
