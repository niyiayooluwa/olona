import 'package:flutter/material.dart';
import 'package:olona/core/errors/app_error_view.dart';
import 'app_toast.dart';
import 'toast_manager.dart';

export 'app_toast.dart';
export 'toast_manager.dart';

/// Clean extension methods on [BuildContext] to display app toasts.
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

  /// Show an error toast notification with optional primary/secondary actions.
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
    final errorView = AppErrorView.from(error);
    showErrorToast(
      title: errorView.title,
      description: errorView.description,
      onRetry: onRetry,
    );
  }
}
