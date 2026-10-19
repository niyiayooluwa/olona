enum WindowSize { compact, medium, expanded, large }

WindowSize windowSizeFor(double width) {
  if (width < 600) return WindowSize.compact;
  if (width < 840) return WindowSize.medium;
  if (width < 1200) return WindowSize.expanded;
  return WindowSize.large;
}
