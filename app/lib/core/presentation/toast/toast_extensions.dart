import 'package:flutter/material.dart';
import 'package:olona/core/errors/error_mapper.dart';
import 'app_toast.dart';
import 'toast_manager.dart';

export 'app_toast.dart';
export 'toast_manager.dart';

/// Extension methods on [BuildContext] to show clean, standardized app toasts.
extension AppToastContextExtension on BuildContext {
  /// Show a general toast notification.
  void showAppToast({
    required ToastType type,
    required String title,
    String? description,
    List<ToastAction>? actions,
    Duration? duration = const Duration(seconds: 4),
  }) {
    ToastManager.instance.show(
      this,
      type: type,
      title: title,
      description: description,
      actions: actions,
      duration: duration,
    );
  }

  /// Show a success toast notification (single-line or with short description).
  void showSuccessToast({
    required String title,
    String? description,
    Duration? duration = const Duration(seconds: 3),
  }) {
    showAppToast(
      type: ToastType.success,
      title: title,
      description: description,
      duration: duration,
    );
  }

  /// Show an info toast notification.
  void showInfoToast({
    required String title,
    String? description,
    Duration? duration = const Duration(seconds: 4),
  }) {
    showAppToast(
      type: ToastType.info,
      title: title,
      description: description,
      duration: duration,
    );
  }

  /// Show a warning toast notification.
  void showWarningToast({
    required String title,
    String? description,
    Duration? duration = const Duration(seconds: 5),
  }) {
    showAppToast(
      type: ToastType.warning,
      title: title,
      description: description,
      duration: duration,
    );
  }

  /// Show an error toast notification with optional retry and dismiss actions.
  void showErrorToast({
    required String title,
    String? description,
    VoidCallback? onRetry,
    VoidCallback? onDismissAction,
    Duration? duration = const Duration(seconds: 6),
  }) {
    final actions = <ToastAction>[];
    if (onRetry != null) {
      actions.add(
        ToastAction(
          label: 'Retry',
          isPrimary: true,
          onPressed: onRetry,
        ),
      );
    }
    if (onDismissAction != null) {
      actions.add(
        ToastAction(
          label: 'Dismiss',
          isPrimary: false,
          onPressed: onDismissAction,
        ),
      );
    }

    showAppToast(
      type: ToastType.error,
      title: title,
      description: description,
      actions: actions.isNotEmpty ? actions : null,
      duration: duration,
    );
  }

  /// Map any error or [Failure] into conversational, human-friendly UX copy.
  void showFailureToast(
    Object error, {
    VoidCallback? onRetry,
  }) {
    final (title, description) = _mapErrorToUxCopy(error);
    showErrorToast(
      title: title,
      description: description,
      onRetry: onRetry,
    );
  }

  (String title, String description) _mapErrorToUxCopy(Object error) {
    if (error is Failure) {
      switch (error.code) {
        case 'email_already_registered':
          return (
            'Email already registered',
            'An account with this email address already exists. Try signing in instead.',
          );
        case 'username_taken':
          return (
            'Username is taken',
            'This organization handle or username is already in use. Please choose another.',
          );
        case 'invalid_credentials':
          return (
            'Invalid credentials',
            'The email or password you entered is incorrect. Please check and try again.',
          );
        case 'otp_incorrect':
          return (
            'Incorrect verification code',
            'The code you entered does not match. Please verify and try again.',
          );
        case 'otp_expired':
          return (
            'Verification code expired',
            'This code has expired. Please request a new code to continue.',
          );
        case 'otp_max_attempts':
          return (
            'Too many failed attempts',
            'You have exceeded the maximum number of attempts. Please request a new code.',
          );
        case 'otp_cooldown':
          return (
            'Please wait a moment',
            'You requested a code recently. Please wait a short while before requesting another.',
          );
        case 'rate_limited':
          return (
            'Too many requests',
            'You are making requests too quickly. Please pause a moment and try again.',
          );
        case 'network':
          return (
            'Connection issue',
            'Unable to connect to the server. Please check your internet connection.',
          );
        case 'timeout':
          return (
            'Request timed out',
            'The server is taking too long to respond. Please check your connection and retry.',
          );
        case 'validation_failed':
          final detailMsg = error.details != null && error.details!.isNotEmpty
              ? error.details!.map((d) => '${d.field}: ${d.code}').join(', ')
              : null;
          return (
            'Check your details',
            detailMsg ?? error.message,
          );
        case 'account_suspended':
          return (
            'Account suspended',
            'Your account has been suspended. Please contact your organization administrator.',
          );
        case 'account_deactivated':
          return (
            'Account deactivated',
            'This account has been deactivated.',
          );
        case 'internal_error':
          return (
            'Server error',
            'Our service encountered an unexpected problem. We are looking into it.',
          );
        default:
          return (
            'Something went wrong',
            error.message.isNotEmpty ? error.message : 'Please try again.',
          );
      }
    }

    return (
      'Something went wrong',
      'We couldn’t complete your request. Check your connection and try again.',
    );
  }
}
