import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:olona/core/storage/secure_storage.dart';
import 'package:olona/features/auth/ui/login/login_screen.dart';
import 'package:olona/features/auth/ui/register/register_screen.dart';
import 'package:olona/features/auth/ui/reset_password/forgot_password_screen.dart';
import 'package:olona/features/auth/ui/reset_password/reset_password_screen.dart';
import 'package:olona/features/auth/ui/verify/verify_screen.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'router.g.dart';

// I guess this is pretty clear from the signature...
final GlobalKey<NavigatorState> navigatorKey = GlobalKey<NavigatorState>();

/// Lets you browse the authenticated shell before login is implemented.
/// `kDebugMode` is false in release builds, so this can never ship enabled.
const bool _previewAppShell = true;

// 💪 BOOM!!!! Riverpod just generated all the boilerplate
// so you dont have to. You're welcome 💅
@riverpod
GoRouter router(Ref ref) {
  return GoRouter(
    navigatorKey: navigatorKey,
    // Initial location is always splash due to a useEffect trigger that was
    // defined there which works in tandem with our redirect logic
    initialLocation: '/login',

    // Le redirect logic
    redirect: (context, state) async {
      // Local preview of the app shell. Remove this once login works.
      if (kDebugMode && _previewAppShell) return null;

      // 1. Splash screen handles initial routing itself
      if (state.uri.path == '/register') return null;

      // This fetches the storage service instance and gets the access token from
      // there and puts it in the token variable basically... Easy peasy
      final storage = ref.read(secureStorageProvider);
      // Lest i forget, token is a String? meaning it can be null
      final String? token = await storage.getAccessToken();

      // We set the isLoggedIn to whether or not the token is null. Null means
      // the user doesnt have a JWT access token which implies they are not logged in
      final bool isLoggedIn = token != null;
      // Reserved for the post-signup username flow below, which is not wired
      // up yet. It is referenced in the commented redirect block, so keep it
      // in place rather than deleting it out of order.
      // ignore: unused_local_variable
      final isSetUsernameRoute = state.uri.path == '/set-username';

      // This creates the authentication routes... so we can exclude them as public
      // routes for unauthenticated users
      final bool isAuthRoute = [
        '/login',
        '/register',
        '/get-started',
        '/onboarding',
        '/verify-otp',
        '/forgot-password',
        '/reset-password',
      ].contains(state.uri.path);

      // If users is not logged in, and user is not on a public route, kick them to
      // the get started screen
      if (!isLoggedIn && !isAuthRoute) {
        return '/register';
      }

      // We check the value of isLoggedIn and check if there is a user saved in state
      // The app then checks the user state to see if they have a username. if yes,
      // they are taken to the Home screen, but if no, they are taken to the screen
      // where they have to set their username after which they can go to the home screen
      // if (isLoggedIn) {
      //   // User is a UserModel?
      //   final user = ref.read(userProvider).value;
      //   // userProvider might be loading, so if it's null, we don't redirect yet unless it's an error?
      //   // Actually, GoogleAuthNotifier waits for it to load before setting isLoggedIn (via invalidating).
      //   if (user != null && (user.username == null || user.username!.isEmpty)) {
      //     if (!isSetUsernameRoute) return '/set-username';
      //   } else if (isAuthRoute || isSetUsernameRoute) {
      //     return '/home';
      //   }
      // }

      return null;
    },

    routes: [
      GoRoute(
        path: '/register',
        name: 'register',
        builder: (context, state) => const RegisterScreen(),
      ),

      GoRoute(
        path: '/login',
        name: 'login',
        builder: (context, state) => const LoginScreen(),
      ),

      GoRoute(
        path: '/verify',
        name: 'verify',
        builder: (context, state) {
          final email =
              (state.extra as String?) ??
              state.uri.queryParameters['email'] ??
              '';
          return VerifyScreen(email: email);
        },
      ),

      GoRoute(
        path: '/forgot-password',
        name: 'forgot-password',
        builder: (context, state) => const ForgotPasswordScreen(),
      ),
      GoRoute(
        path: '/reset-password',
        name: 'reset-password',
        builder: (context, state) {
          final email = (state.extra as String?) ??
              state.uri.queryParameters['email'] ??
              '';
          return ResetPasswordScreen(email: email);
        },
      ),
    ],
  );
}
