import { describe, expect, it, vi } from 'vitest';
import { terminalOptions } from './options.js';

describe('terminal options', () => {
  it('maps persisted appearance and input state to xterm options', () => {
    const onLink = vi.fn();
    const options = terminalOptions({
      terminalFontFamily: 'JetBrains Mono',
      terminalFontSize: 18,
      terminalScrollback: 100000,
    }, { disableStdin: true, onLink });

    expect(options).toMatchObject({
      cursorBlink: false,
      convertEol: true,
      disableStdin: true,
      fontFamily: 'JetBrains Mono',
      fontSize: 18,
      scrollback: 100000,
    });
    expect(options.linkHandler.activate).toBe(onLink);
  });
});
