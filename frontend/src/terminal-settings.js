export const DEFAULT_TERMINAL_FONT_SIZE = 13;
export const SYSTEM_TERMINAL_FONT_FAMILY = 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace';
export const DEFAULT_TERMINAL_FONT_FAMILY = '"Bashes JetBrains Mono", "JetBrains Mono", ui-monospace, monospace';
export const TERMINAL_FONT_FAMILY_OPTIONS = Object.freeze([
  { label: 'JetBrains Mono', value: DEFAULT_TERMINAL_FONT_FAMILY },
  { label: 'Ubuntu Mono', value: '"Bashes Ubuntu Mono", "Ubuntu Mono", ui-monospace, monospace' },
  { label: 'Roboto Mono', value: '"Bashes Roboto Mono", "Roboto Mono", ui-monospace, monospace' },
  { label: 'System Monospace', value: SYSTEM_TERMINAL_FONT_FAMILY },
  { label: 'Cascadia Mono', value: '"Cascadia Mono", "Cascadia Code", Consolas, monospace' },
  { label: 'Consolas', value: 'Consolas, "Liberation Mono", monospace' },
  { label: 'Menlo', value: 'Menlo, Monaco, "Courier New", monospace' },
  { label: 'DejaVu Sans Mono', value: '"DejaVu Sans Mono", "Liberation Mono", monospace' },
  { label: 'Liberation Mono', value: '"Liberation Mono", "Courier New", monospace' },
  { label: 'Courier New', value: '"Courier New", Courier, monospace' },
]);
export const DEFAULT_TERMINAL_SCROLLBACK = 100000;
export const MIN_TERMINAL_FONT_SIZE = 10;
export const MAX_TERMINAL_FONT_SIZE = 36;

export function clampNumber(value, min, max, fallback) {
  const number = Number.parseInt(value, 10);
  if (!Number.isFinite(number)) return fallback;
  return Math.min(Math.max(number, min), max);
}

export function loadTerminalSettings(storage) {
  return {
    terminalFontSize: clampNumber(
      storage.getItem('bashes.terminalFontSize'),
      MIN_TERMINAL_FONT_SIZE,
      MAX_TERMINAL_FONT_SIZE,
      DEFAULT_TERMINAL_FONT_SIZE,
    ),
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
