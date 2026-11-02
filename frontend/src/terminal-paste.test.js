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

    expect(pasteIntoTerminal(terminal, 'one\r\ntwo', {})).toBe(true);
    expect(terminal.paste).toHaveBeenCalledWith('one\ntwo');
    expect(pasteIntoTerminal(terminal, '', {})).toBe(false);
    expect(terminal.paste).toHaveBeenCalledTimes(1);
  });

  it('routes programmatic paste through the same event used by keyboard paste', () => {
    class DataTransfer {
      setData(type, value) {
        this.type = type;
        this.value = value;
      }

      getData(type) {
        return type === this.type ? this.value : '';
      }
    }
    class ClipboardEvent {
      constructor(type, options) {
        this.type = type;
        Object.assign(this, options);
      }
    }
    const terminal = {
      focus: vi.fn(),
      paste: vi.fn(),
      textarea: { dispatchEvent: vi.fn() },
    };

    expect(pasteIntoTerminal(terminal, 'one\r\ntwo', { ClipboardEvent, DataTransfer })).toBe(true);
    expect(terminal.focus).toHaveBeenCalledOnce();
    expect(terminal.textarea.dispatchEvent).toHaveBeenCalledOnce();
    const event = terminal.textarea.dispatchEvent.mock.calls[0][0];
    expect(event.type).toBe('paste');
    expect(event.clipboardData.type).toBe('text/plain');
    expect(event.clipboardData.value).toBe('one\ntwo');
    expect(terminal.paste).not.toHaveBeenCalled();
  });

  it('falls back when the webview rejects synthetic clipboard data', () => {
    class DataTransfer {
      setData() {}
    }
    class ClipboardEvent {
      constructor() {
        this.clipboardData = null;
      }
    }
    const terminal = {
      paste: vi.fn(),
      textarea: { dispatchEvent: vi.fn() },
    };

    expect(pasteIntoTerminal(terminal, 'one\ntwo', { ClipboardEvent, DataTransfer })).toBe(true);
    expect(terminal.textarea.dispatchEvent).not.toHaveBeenCalled();
    expect(terminal.paste).toHaveBeenCalledWith('one\ntwo');
  });
});
