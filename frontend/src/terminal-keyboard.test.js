import { describe, expect, it } from 'vitest';
import { repeatedKeyData } from './terminal-keyboard.js';

describe('terminal key repeat fallback', () => {
  it('keeps handling unmodified repeated keys', () => {
    expect(repeatedKeyData({ key: 'j' })).toBe('j');
    expect(repeatedKeyData({ key: 'ArrowLeft' })).toBe('\x1b[D');
    expect(repeatedKeyData({ key: 'ArrowLeft' }, true)).toBe('\x1bOD');
    expect(repeatedKeyData({ key: 'Tab', shiftKey: true })).toBe('\x1b[Z');
  });

  it('leaves modified repeated keys to xterm', () => {
    expect(repeatedKeyData({ key: 'ArrowLeft', altKey: true })).toBe('');
    expect(repeatedKeyData({ key: 'ArrowRight', ctrlKey: true })).toBe('');
    expect(repeatedKeyData({ key: 'ArrowUp', metaKey: true })).toBe('');
  });
});
