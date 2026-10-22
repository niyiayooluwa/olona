import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:olona/core/routing/router.dart';
import 'package:olona/core/theme/app_theme.dart';
import 'package:shadcn_ui/shadcn_ui.dart';

// Not yet enabled — re-add these alongside the initialization block below
// when Hive and Firebase are switched on. Leaving them imported while unused
// fails `flutter analyze --fatal-infos` in CI.
// import 'package:firebase_core/firebase_core.dart';
// import 'package:google_fonts/google_fonts.dart';

//===================================
// STORAGE HELPERS
//===================================
/*Future<void> _safeOpenBox<T>(String boxName) async {
  try {
    if (!Hive.isBoxOpen(boxName)) {
      await Hive.openBox<T>(boxName);
    }
  } catch (_) {
    try {
      await Hive.deleteBoxFromDisk(boxName);
      await Hive.openBox<T>(boxName);
    } catch (_) {
      // Degrade gracefully if local storage fails
    }
  }
}*/

//===================================
// APP ENTRY POINT
//===================================
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  /*
  // Initialize Hive
  await Hive.initFlutter();
  
  GoogleFonts.config.allowRuntimeFetching = false;

  await Future.wait([
    _safeOpenBox<String>('app_cache'),
    _safeOpenBox<String>('user_cache'), 
    // () async {
    //   try {
    //     await Firebase.initializeApp(
    //       options: DefaultFirebaseOptions.currentPlatform,
    //     );
    //   } catch (_) {}
    // }(),
  ]);*/

  final container = ProviderContainer();
  runApp(
    UncontrolledProviderScope(container: container, child: const MainApp()),
  );
}

//===================================
// ROOT WIDGET
//===================================
class MainApp extends ConsumerWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);

    return ShadApp.router(
      title: 'Olona',
      debugShowCheckedModeBanner: true,

      //===================================
      // THEME INJECTION
      //===================================
      theme: AppTheme.light(),
      //darkTheme: AppTheme.dark(),

      //===================================
      // ROUTING
      //===================================
      routerConfig: router,

      //===================================
      // SYSTEM UI CONFIG
      //===================================
      builder: (context, child) {
        final brightness = Theme.of(context).brightness;
        SystemChrome.setSystemUIOverlayStyle(
          brightness == Brightness.light
              ? SystemUiOverlayStyle.dark
              : SystemUiOverlayStyle.light,
        );
        return child ??
            const Scaffold(body: Center(child: Text("Router not configured")));
      },
    );
  }
}
