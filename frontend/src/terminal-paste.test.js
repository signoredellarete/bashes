import { describe, expect, it, vi } from 'vitest';
import { normalizeTerminalPaste, pasteIntoTerminal } from './terminal-paste.js';

describe('terminal paste', () => {
  it('normalizes clipboard line endings without adding blank lines', () => {
    const yaml = 'apiVersion: v1\r\nkind: ConfigMap\r\nmetadata:\r\n  name: example\r\n';

    expect(normalizeTerminalPaste(yaml)).toBe(
      'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: example\n',
    );
  });

  it('handles legacy carriage returns and preserves Unix line endings', () => {
    expect(normalizeTerminalPaste('one\rtwo\r')).toBe('one\ntwo\n');
    expect(normalizeTerminalPaste('one\ntwo\n')).toBe('one\ntwo\n');
  });

  it('uses the terminal paste API so bracketed paste mode is respected', () => {
    const terminal = { paste: vi.fn() };

    expect(pasteIntoTerminal(terminal, 'one\r\ntwo')).toBe(true);
    expect(terminal.paste).toHaveBeenCalledWith('one\ntwo');
    expect(pasteIntoTerminal(terminal, '')).toBe(false);
    expect(terminal.paste).toHaveBeenCalledTimes(1);
  });
});
