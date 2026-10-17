import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:shadcn_ui/shadcn_ui.dart';
import 'color_scheme.dart';

class AppTheme {
  //===================================
  // LIGHT THEME
  //===================================
  static ShadThemeData light() {
    return ShadThemeData(
      //===================================
      // BASE COLORS & BORDERS
      //===================================
      colorScheme: const AppColorScheme.light(),
      radius: BorderRadius.circular(12),

      //===================================
      // TYPOGRAPHY
      //===================================
      textTheme: ShadTextTheme.fromGoogleFont(GoogleFonts.inter),

      //===================================
      // BUTTONS
      //===================================
      primaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      secondaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      destructiveButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      outlineButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      buttonSizesTheme: const ShadButtonSizesTheme(
        regular: ShadButtonSizeTheme(
          height: 48,
          padding: EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        ),
        sm: ShadButtonSizeTheme(
          height: 40,
          padding: EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        ),
        lg: ShadButtonSizeTheme(
          height: 52,
          padding: EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        ),
        icon: ShadButtonSizeTheme(
          height: 44,
          width: 44,
          padding: EdgeInsets.zero,
        ),
      ),

      //===================================
      // INPUTS & FORMS
      //===================================
      inputTheme: const ShadInputTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
          focusedBorder: ShadBorder(
            radius: BorderRadius.all(Radius.circular(26)),
          ),
          errorBorder: ShadBorder(
            radius: BorderRadius.all(Radius.circular(26)),
          ),
        ),
      ),
      selectTheme: const ShadSelectTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),

      //===================================
      // DIALOGS & OVERLAYS
      //===================================
      alertDialogTheme: const ShadDialogTheme(
        removeBorderRadiusWhenTiny: false,
        expandActionsWhenTiny: false,
        actionsAxis: Axis.horizontal,
        titleTextAlign: TextAlign.start,
        descriptionTextAlign: TextAlign.start,
        padding: EdgeInsets.fromLTRB(20, 16, 20, 12),
      ),
    );
  }

  //===================================
  // DARK THEME
  //===================================
  static ShadThemeData dark() {
    return ShadThemeData(
      //===================================
      // BASE COLORS & BORDERS
      //===================================
      colorScheme: const AppColorScheme.dark(),
      radius: BorderRadius.circular(12),

      //===================================
      // TYPOGRAPHY
      //===================================
      textTheme: ShadTextTheme.fromGoogleFont(GoogleFonts.inter),

      //===================================
      // BUTTONS
      //===================================
      primaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      secondaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      destructiveButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      outlineButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),
      buttonSizesTheme: const ShadButtonSizesTheme(
        regular: ShadButtonSizeTheme(
          height: 48,
          padding: EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        ),
        sm: ShadButtonSizeTheme(
          height: 40,
          padding: EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        ),
        lg: ShadButtonSizeTheme(
          height: 52,
          padding: EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        ),
        icon: ShadButtonSizeTheme(
          height: 44,
          width: 44,
          padding: EdgeInsets.zero,
        ),
      ),

      //===================================
      // INPUTS & FORMS
      //===================================
      inputTheme: const ShadInputTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
          focusedBorder: ShadBorder(
            radius: BorderRadius.all(Radius.circular(26)),
          ),
          errorBorder: ShadBorder(
            radius: BorderRadius.all(Radius.circular(26)),
          ),
        ),
      ),
      selectTheme: const ShadSelectTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(radius: BorderRadius.all(Radius.circular(26))),
        ),
      ),

      //===================================
      // DIALOGS & OVERLAYS
      //===================================
      alertDialogTheme: const ShadDialogTheme(
        removeBorderRadiusWhenTiny: false,
        expandActionsWhenTiny: false,
        actionsAxis: Axis.horizontal,
        titleTextAlign: TextAlign.start,
        descriptionTextAlign: TextAlign.start,
        padding: EdgeInsets.fromLTRB(20, 16, 20, 12),
      ),
    );
  }
}
