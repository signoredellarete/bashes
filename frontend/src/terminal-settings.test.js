import { describe, expect, it } from 'vitest';
import {
  DEFAULT_TERMINAL_FONT_FAMILY,
  DEFAULT_TERMINAL_SCROLLBACK,
  MAX_TERMINAL_FONT_SIZE,
  TERMINAL_FONT_FAMILY_OPTIONS,
  loadTerminalSettings,
  persistTerminalSettings,
} from './terminal-settings.js';

function memoryStorage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  };
}

describe('terminal settings', () => {
  it('provides a compact list of distinct terminal font stacks', () => {
    expect(TERMINAL_FONT_FAMILY_OPTIONS.length).toBeGreaterThanOrEqual(5);
    expect(new Set(TERMINAL_FONT_FAMILY_OPTIONS.map(({ value }) => value)).size).toBe(
      TERMINAL_FONT_FAMILY_OPTIONS.length,
    );
    expect(TERMINAL_FONT_FAMILY_OPTIONS[0]).toEqual({
      label: 'JetBrains Mono',
      value: DEFAULT_TERMINAL_FONT_FAMILY,
    });
    expect(TERMINAL_FONT_FAMILY_OPTIONS.map(({ label }) => label)).toEqual(expect.arrayContaining([
      'Ubuntu Mono',
      'Roboto Mono',
      'System Monospace',
    ]));
  });

  it('uses the high scrollback default and clamps invalid values', () => {
    const settings = loadTerminalSettings(memoryStorage({ 'bashes.terminalFontSize': '999' }));
    expect(settings.terminalFontSize).toBe(MAX_TERMINAL_FONT_SIZE);
    expect(settings.terminalFontFamily).toBe(DEFAULT_TERMINAL_FONT_FAMILY);
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
