import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
// import 'package:olona/core/constants/constants.dart';
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
      colorScheme: const Neutral.light(),
      // radius: BorderRadius.circular(UIConstants.lg),

      //===================================
      // TYPOGRAPHY
      //===================================
      textTheme: ShadTextTheme.fromGoogleFont(GoogleFonts.inter),

      //===================================
      // BUTTONS
      //===================================
      primaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      secondaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      destructiveButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      outlineButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
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
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
          focusedBorder: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
          errorBorder: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      selectTheme: const ShadSelectTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
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
      colorScheme: const Neutral.dark(),
      // radius: BorderRadius.circular(UIConstants.lg),

      //===================================
      // TYPOGRAPHY
      //===================================
      textTheme: ShadTextTheme.fromGoogleFont(GoogleFonts.inter),

      //===================================
      // BUTTONS
      //===================================
      primaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      secondaryButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      destructiveButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      outlineButtonTheme: const ShadButtonTheme(
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
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
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
          focusedBorder: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
          errorBorder: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
        ),
      ),
      selectTheme: const ShadSelectTheme(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: ShadDecoration(
          border: ShadBorder(
            // radius: BorderRadius.all(Radius.circular(UIConstants.lg)),
          ),
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
