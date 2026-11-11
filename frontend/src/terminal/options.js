export function terminalOptions(settings, { disableStdin = false, onLink } = {}) {
  return {
    cursorBlink: !disableStdin,
    convertEol: true,
    disableStdin,
    fontFamily: settings.terminalFontFamily,
    fontSize: settings.terminalFontSize,
    rightClickSelectsWord: false,
    scrollback: settings.terminalScrollback,
    linkHandler: { activate: onLink },
    theme: {
      background: '#101418',
      foreground: '#d7dde5',
      cursor: '#f5c542',
      selectionBackground: '#3d4a58',
    },
  };
}
