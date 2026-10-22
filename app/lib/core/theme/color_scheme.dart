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
    super.background = const Color(0xffffffff),
    super.foreground = const Color(0xff020817),
    super.card = const Color(0xffffffff),
    super.cardForeground = const Color(0xff020817),
    super.popover = const Color(0xffffffff),
    super.popoverForeground = const Color(0xff020817),
    super.primary = const Color(0xff0f172a),
    super.primaryForeground = const Color(0xfff8fafc),
    super.secondary = const Color(0xfff1f5f9),
    super.secondaryForeground = const Color(0xff0f172a),
    super.muted = const Color(0xfff1f5f9),
    super.mutedForeground = const Color(0xff64748b),
    super.accent = const Color(0xfff1f5f9),
    super.accentForeground = const Color(0xff0f172a),
    super.destructive = const Color(0xffef4444),
    super.destructiveForeground = const Color(0xfff8fafc),
    super.border = const Color(0xffe2e8f0),
    super.input = const Color(0xffe2e8f0),
    super.ring = const Color(0xff020817),
    super.selection = const Color(0xFFB4D7FF),
    super.custom,
  });

  const Neutral.light({
    super.background = const Color(0xffffffff),
    super.foreground = const Color(0xff030712),
    super.card = const Color(0xffffffff),
    super.cardForeground = const Color(0xff030712),
    super.popover = const Color(0xffffffff),
    super.popoverForeground = const Color(0xff030712),
    super.primary = const Color(0xff111827),
    super.primaryForeground = const Color(0xfff9fafb),
    super.secondary = const Color(0xfff3f4f6),
    super.secondaryForeground = const Color(0xff111827),
    super.muted = const Color(0xfff3f4f6),
    super.mutedForeground = const Color(0xff6b7280),
    super.accent = const Color(0xfff3f4f6),
    super.accentForeground = const Color(0xff111827),
    super.destructive = const Color(0xffef4444),
    super.destructiveForeground = const Color(0xfff9fafb),
    super.border = const Color(0xffe5e7eb),
    super.input = const Color(0xffe5e7eb),
    super.ring = const Color(0xff030712),
    super.selection = const Color(0xFFB4D7FF),
    super.custom,
  });

  const Neutral.dark({
    super.background = const Color(0xff030712),
    super.foreground = const Color(0xfff9fafb),
    super.card = const Color(0xff030712),
    super.cardForeground = const Color(0xfff9fafb),
    super.popover = const Color(0xff030712),
    super.popoverForeground = const Color(0xfff9fafb),
    super.primary = const Color(0xfff9fafb),
    super.primaryForeground = const Color(0xff111827),
    super.secondary = const Color(0xff1f2937),
    super.secondaryForeground = const Color(0xfff9fafb),
    super.muted = const Color(0xff1f2937),
    super.mutedForeground = const Color(0xff9ca3af),
    super.accent = const Color(0xff1f2937),
    super.accentForeground = const Color(0xfff9fafb),
    super.destructive = const Color(0xffef4444),
    super.destructiveForeground = const Color(0xfff9fafb),
    super.border = const Color(0xff1f2937),
    super.input = const Color(0xff1f2937),
    super.ring = const Color(0xffd1d5db),
    super.selection = const Color(0xFF355172),
    super.custom,
  });
}
