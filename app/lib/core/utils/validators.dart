// core/utils/validators.dart

/// Returns an error message if the password is invalid, null if valid.
/// Rules: 8+ chars, uppercase, lowercase, digit, special character.
String? validatePassword(String password) {
  if (password.length < 8) return 'Password must be at least 8 characters';
  if (!password.contains(RegExp(r'[A-Z]'))) {
    return 'Password must contain an uppercase letter';
  }
  if (!password.contains(RegExp(r'[a-z]'))) {
    return 'Password must contain a lowercase letter';
  }
  if (!password.contains(RegExp(r'[0-9]'))) {
    return 'Password must contain a number';
  }
  if (!password.contains(RegExp(r'[@\$!%*?&]'))) {
    return 'Password must contain a special character (@\$!%*?&)';
  }
  return null;
}

/// Returns an error message if the email is invalid, null if valid.
String? validateEmail(String email) {
  final emailRegex = RegExp(r'^[\w.-]+@[\w.-]+\.\w{2,}$');
  if (email.isEmpty) return 'Email is required';
  if (!emailRegex.hasMatch(email)) return 'Enter a valid email address';
  return null;
}

/// Returns an error message if the username is invalid, null if valid.
/// Strips a leading '@' if present before validating.
String? validateUsername(String username) {
  final cleaned = username.startsWith('@') ? username.substring(1) : username;
  final trimmed = cleaned.trim();

  if (trimmed.isEmpty) return 'Username is required';
  if (trimmed.length < 3) {
    return 'Username must be at least 3 characters';
  }
  if (!RegExp(r'^[a-zA-Z0-9_]+$').hasMatch(trimmed)) {
    return 'Only letters, numbers, and underscores allowed';
  }
  return null;
}

/// Returns each password requirement mapped to whether it's currently met.
/// Keep this in sync with validatePassword's rules above.
Map<String, bool> passwordRequirements(String password) {
  return {
    'At least 8 characters': password.length >= 8,
    'One uppercase letter': password.contains(RegExp(r'[A-Z]')),
    'One lowercase letter': password.contains(RegExp(r'[a-z]')),
    'One number': password.contains(RegExp(r'[0-9]')),
    'One special character (@\$!%*?&)': password.contains(
      RegExp(r'[@\$!%*?&]'),
    ),
  };
}
