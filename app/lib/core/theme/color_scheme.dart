import 'package:flutter/widgets.dart';
import 'package:shadcn_ui/shadcn_ui.dart';
/*
@immutable
class AppColorScheme extends ShadColorScheme {
  //===================================
  // BASE CONSTRUCTOR
  //===================================
  const AppColorScheme({
    required super.background,
    required super.foreground,
    required super.card,
    required super.cardForeground,
    required super.popover,
    required super.popoverForeground,
    required super.primary,
    required super.primaryForeground,
    required super.secondary,
    required super.secondaryForeground,
    required super.muted,
    required super.mutedForeground,
    required super.accent,
    required super.accentForeground,
    required super.destructive,
    required super.destructiveForeground,
    required super.border,
    required super.input,
    required super.ring,
    required super.selection,
    super.custom,
  });

  //===================================
  // LIGHT MODE COLORS
  //===================================
  const AppColorScheme.light({
    super.background = const Color(0xffffffff),
    super.foreground = const Color(0xff09090b),
    super.card = const Color(0xffffffff),
    super.cardForeground = const Color(0xff09090b),
    super.popover = const Color(0xffffffff),
    super.popoverForeground = const Color(0xff09090b),
    super.primary = const Color(0xff16a34a),
    super.primaryForeground = const Color(0xfffff1f2),
    super.secondary = const Color(0xfff4f4f5),
    super.secondaryForeground = const Color(0xff18181b),
    super.muted = const Color(0xfff4f4f5),
    super.mutedForeground = const Color(0xff71717a),
    super.accent = const Color(0xfff4f4f5),
    super.accentForeground = const Color(0xff18181b),
    super.destructive = const Color(0xffef4444),
    super.destructiveForeground = const Color(0xfffafafa),
    super.border = const Color(0xffe4e4e7),
    super.input = const Color(0xffe4e4e7),
    super.ring = const Color(0xff16a34a),
    super.selection = const Color(0xFFB4D7FF),
    super.custom,
  });

  //===================================
  // DARK MODE COLORS
  //===================================
  const AppColorScheme.dark({
    super.background = const Color(0xff0c0a09),
    super.foreground = const Color(0xfff2f2f2),
    super.card = const Color(0xff1c1917),
    super.cardForeground = const Color(0xfff2f2f2),
    super.popover = const Color(0xff171717),
    super.popoverForeground = const Color(0xfff2f2f2),
    super.primary = const Color(0xff22c55e),
    super.primaryForeground = const Color(0xff052e16),
    super.secondary = const Color(0xff27272a),
    super.secondaryForeground = const Color(0xfffafafa),
    super.muted = const Color(0xff262626),
    super.mutedForeground = const Color(0xffa1a1aa),
    super.accent = const Color(0xff292524),
    super.accentForeground = const Color(0xfffafafa),
    super.destructive = const Color(0xffef4444),
    super.destructiveForeground = const Color(0xfffef2f2),
    super.border = const Color(0xff27272a),
    super.input = const Color(0xff27272a),
    super.ring = const Color(0xff15803d),
    super.selection = const Color(0xFF355172),
    super.custom,
  });
}*/

@immutable
class Neutral extends ShadColorScheme {
  const Neutral({
    required super.background,
    required super.foreground,
    required super.card,
    required super.cardForeground,
    required super.popover,
    required super.popoverForeground,
    required super.primary,
    required super.primaryForeground,
    required super.secondary,
    required super.secondaryForeground,
    required super.muted,
    required super.mutedForeground,
    required super.accent,
    required super.accentForeground,
    required super.destructive,
    required super.destructiveForeground,
    required super.border,
    required super.input,
    required super.ring,
    required super.selection,
    super.custom,
  });

  const Neutral.light({
    super.background = const Color(0xFFFFFFFF),
    super.foreground = const Color(0xFF030712),
    super.card = const Color(0xFFFFFFFF),
    super.cardForeground = const Color(0xFF030712),
    super.popover = const Color(0xFFFFFFFF),
    super.popoverForeground = const Color(0xFF030712),
    super.primary = const Color(0xFF111827),
    super.primaryForeground = const Color(0xFFF9FAFB),
    super.secondary = const Color(0xFFF3F4F6),
    super.secondaryForeground = const Color(0xFF111827),
    super.muted = const Color(0xFFF3F4F6),
    super.mutedForeground = const Color(0xFF6B7280),
    super.accent = const Color(0xFFF3F4F6),
    super.accentForeground = const Color(0xFF111827),
    super.destructive = const Color(0xFFEF4444),
    super.destructiveForeground = const Color(0xFFF9FAFB),
    super.border = const Color(0xFFE5E7EB),
    super.input = const Color(0xFFE5E7EB),
    super.ring = const Color(0xFF030712),
    super.selection = const Color(0xFFB4D7FF),
    super.custom = const {
      'success': Color(0xFF16A34A),
      'successSurface': Color(0xFFDCF5E5),
      'warning': Color(0xFFA8650A),
      'warningSurface': Color(0xFFFBEFD3),
      'destructiveSurface': Color(0xFFFDE7E7),
    },
  });

  const Neutral.dark({
    super.background = const Color(0xFF111113),
    super.foreground = const Color(0xFFFAFAFA),
    super.card = const Color(0xFF161618),
    super.cardForeground = const Color(0xFFFAFAFA),
    super.popover = const Color(0xFF1A1A1D),
    super.popoverForeground = const Color(0xFFFAFAFA),
    super.primary = const Color(0xFFFAFAFA),
    super.primaryForeground = const Color(0xFF18181B),
    super.secondary = const Color(0xFF232326),
    super.secondaryForeground = const Color(0xFFFAFAFA),
    super.muted = const Color(0xFF1B1B1E),
    super.mutedForeground = const Color(0xFFA1A1AA),
    super.accent = const Color(0xFF26262A),
    super.accentForeground = const Color(0xFFFAFAFA),
    super.destructive = const Color(0xFFEF4444),
    super.destructiveForeground = const Color(0xFFFAFAFA),
    super.border = const Color(0xFF27272A),
    super.input = const Color(0xFF303034),
    super.ring = const Color(0xFFD4D4D8),
    super.selection = const Color(0xFF355172),
    super.custom = const {
      'success': Color(0xFF4ADE80),
      'successSurface': Color(0xFF122419),
      'warning': Color(0xFFFBBF24),
      'warningSurface': Color(0xFF2A2110),
      'destructiveSurface': Color(0xFF2D1618),
    },
  });
}

/// Ledger status colours, read from [ShadColorScheme.custom].
/// Usage: `ShadTheme.of(context).colorScheme.success`
extension NeutralStatusColors on ShadColorScheme {
  Color get success => custom['success']!;
  Color get successSurface => custom['successSurface']!;
  Color get warning => custom['warning']!;
  Color get warningSurface => custom['warningSurface']!;
  Color get destructiveSurface => custom['destructiveSurface']!;
}
