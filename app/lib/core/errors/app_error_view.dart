import 'package:olona/core/errors/error_mapper.dart';

/// Clean model representing user-facing copy for an error toast or error dialog.
class AppErrorView {
  const AppErrorView({required this.title, this.description});

  final String title;
  final String? description;

  /// Maps any error or [Failure] into conversational, human-friendly UX copy.
  factory AppErrorView.from(Object error) {
    if (error is! Failure) {
      return const AppErrorView(
        title: 'Something went wrong',
        description:
            'We couldn’t complete your request. Check your connection and try again.',
      );
    }

    switch (error.code) {
      //========================================================================
      //Auth & Credentials
      //========================================================================
      case 'email_already_registered':
        return const AppErrorView(
          title: 'Email already registered',
          description:
              'An account with this email address already exists. Try signing in instead.',
        );
      case 'email_not_verified':
        return const AppErrorView(
          title: 'Email not verified',
          description:
              'Please verify your email address to access your account.',
        );
      case 'invalid_credentials':
        return const AppErrorView(
          title: 'Invalid credentials',
          description:
              'The email or password you entered is incorrect. Please check and try again.',
        );
      case 'username_taken':
        return const AppErrorView(
          title: 'Username is taken',
          description:
              'This organization handle or username is already in use. Please choose another.',
        );

      //========================================================================
      // OTP & Verification
      //========================================================================
      case 'otp_incorrect':
        return const AppErrorView(
          title: 'Incorrect verification code',
          description:
              'The code you entered does not match. Please verify and try again.',
        );
      case 'otp_expired':
        return const AppErrorView(
          title: 'Verification code expired',
          description:
              'This code has expired. Please request a new code to continue.',
        );
      case 'otp_max_attempts':
        return const AppErrorView(
          title: 'Too many failed attempts',
          description:
              'You have exceeded the maximum number of attempts. Please request a new code.',
        );
      case 'otp_cooldown':
        return const AppErrorView(
          title: 'Please wait a moment',
          description:
              'You requested a code recently. Please wait a short while before requesting another.',
        );

      //========================================================================
      // Account Status
      //========================================================================
      case 'account_suspended':
        return const AppErrorView(
          title: 'Account suspended',
          description:
              'Your account has been suspended. Please contact your organization administrator.',
        );
      case 'account_deactivated':
        return const AppErrorView(
          title: 'Account deactivated',
          description: 'This account has been deactivated.',
        );

      //========================================================================
      // Network & Rate Limits
      //========================================================================
      case 'rate_limited':
        return const AppErrorView(
          title: 'Too many requests',
          description:
              'You are making requests too quickly. Please pause a moment and try again.',
        );
      case 'network':
        return const AppErrorView(
          title: 'Connection issue',
          description:
              'Unable to connect to the server. Please check your internet connection.',
        );
      case 'timeout':
        return const AppErrorView(
          title: 'Request timed out',
          description:
              'The server is taking too long to respond. Please check your connection and retry.',
        );

      //========================================================================
      // Validation & Server
      //========================================================================
      case 'validation_failed':
        final detailMsg = error.details != null && error.details!.isNotEmpty
            ? error.details!.map((d) => '${d.field}: ${d.code}').join(', ')
            : null;
        return AppErrorView(
          title: 'Check your details',
          description: detailMsg ?? error.message,
        );
      case 'internal_error':
        return const AppErrorView(
          title: 'Server error',
          description:
              'Our service encountered an unexpected problem. We are looking into it.',
        );

      default:
        return AppErrorView(
          title: 'Something went wrong',
          description: error.message.isNotEmpty
              ? error.message
              : 'Please try again.',
        );
    }
  }
}
