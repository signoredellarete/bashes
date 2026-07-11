export const DEFAULT_TERMINAL_FONT_SIZE = 13;
export const DEFAULT_TERMINAL_FONT_FAMILY = 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace';
export const DEFAULT_TERMINAL_SCROLLBACK = 100000;

export function clampNumber(value, min, max, fallback) {
  const number = Number.parseInt(value, 10);
  if (!Number.isFinite(number)) return fallback;
  return Math.min(Math.max(number, min), max);
}

export function loadTerminalSettings(storage) {
  return {
    terminalFontSize: clampNumber(storage.getItem('bashes.terminalFontSize'), 10, 22, DEFAULT_TERMINAL_FONT_SIZE),
    terminalFontFamily: storage.getItem('bashes.terminalFontFamily')?.trim() || DEFAULT_TERMINAL_FONT_FAMILY,
    terminalScrollback: clampNumber(storage.getItem('bashes.terminalScrollback'), 1000, 500000, DEFAULT_TERMINAL_SCROLLBACK),
    terminalCopyOnSelect: readBoolean(storage.getItem('bashes.terminalCopyOnSelect'), true),
  };
}

export function persistTerminalSettings(storage, settings) {
  storage.setItem('bashes.terminalFontSize', String(settings.terminalFontSize));
  storage.setItem('bashes.terminalFontFamily', settings.terminalFontFamily);
  storage.setItem('bashes.terminalScrollback', String(settings.terminalScrollback));
  storage.setItem('bashes.terminalCopyOnSelect', String(settings.terminalCopyOnSelect));
}

function readBoolean(value, fallback) {
  if (value === 'true') return true;
  if (value === 'false') return false;
  return fallback;
}
