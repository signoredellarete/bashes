import { describe, expect, it } from 'vitest';
import { DEFAULT_TERMINAL_SCROLLBACK, loadTerminalSettings, persistTerminalSettings } from './terminal-settings.js';

function memoryStorage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  };
}

describe('terminal settings', () => {
  it('uses the high scrollback default and clamps invalid values', () => {
    const settings = loadTerminalSettings(memoryStorage({ 'bashes.terminalFontSize': '999' }));
    expect(settings.terminalFontSize).toBe(22);
    expect(settings.terminalScrollback).toBe(DEFAULT_TERMINAL_SCROLLBACK);
  });

  it('persists portable primitive values', () => {
    const storage = memoryStorage();
    persistTerminalSettings(storage, {
      terminalFontSize: 15,
      terminalFontFamily: 'monospace',
      terminalScrollback: 200000,
      terminalCopyOnSelect: false,
    });
    expect(loadTerminalSettings(storage)).toEqual({
      terminalFontSize: 15,
      terminalFontFamily: 'monospace',
      terminalScrollback: 200000,
      terminalCopyOnSelect: false,
    });
  });
});
