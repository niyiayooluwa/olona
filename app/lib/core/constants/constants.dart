class ApiConstants {
  // For Android Emulator, use
  // static const String baseUrl = 'http://10.0.2.2:8080';
  // For iOS Simulator/Web/Physical Device, use http://localhost:8080 or your local IP
  static const String baseUrl = 'http://localhost:8080';

  // Timeouts: because waiting forever is weird and I hate late people
  // Set to 12s because I am actually torn between 10s and 15s
  static const Duration connectTimeout = Duration(seconds: 12);
  static const Duration receiveTimeout = Duration(seconds: 12);
}

/* Spacing scale (dp)
//  static const double xs = 4;
//  static const double sm = 8;
//  static const double md = 16;
//  static const double lg = 24;
//  static const double xl = 32;
//  static const double xxl = 48;
*/
class UIConstants {
  const UIConstants._();

  // Spacing scale (dp)
  static const double xs = 4;
  static const double sm = 8;
  static const double md = 16;
  static const double lg = 24;
  static const double xl = 32;
  static const double xxl = 48;
}
