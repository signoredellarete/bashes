import { describe, expect, it, vi } from 'vitest';
import { createTerminalOutputBuffer, trimChunks } from './output-buffer.js';

describe('terminal output buffer', () => {
  it('buffers output until a terminal exists and then flushes in order', () => {
    const sessions = new Map([['session-1', {}]]);
    const output = createTerminalOutputBuffer({ sessions: () => sessions });
    output.write('session-1', 'one');
    output.write('session-1', 'two');
    const terminal = { write: vi.fn() };
    sessions.set('session-1', { terminal });

    output.flush('session-1');

    expect(terminal.write.mock.calls.flat()).toEqual(['one', 'two']);
    expect(output.size('session-1')).toBe(0);
  });

  it('bounds pending output by dropping the oldest bytes', () => {
    expect(trimChunks(['1234', '5678'], 5)).toEqual(['5678']);
    expect(trimChunks(['12345678'], 5)).toEqual(['45678']);
  });

  it('decodes byte events without changing legacy text events', () => {
    const output = createTerminalOutputBuffer({
      sessions: () => new Map(),
      decodeBase64: vi.fn(() => new Uint8Array([65])),
    });
    expect(output.decode({ bytes: 'QQ==' })).toEqual(new Uint8Array([65]));
    expect(output.decode({ data: 'text' })).toBe('text');
  });
});
