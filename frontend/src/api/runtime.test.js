import { describe, expect, it, vi } from 'vitest';
import {
  onRuntimeEvent,
  openRuntimeURL,
  readClipboardText,
  resolveRuntimeFilePaths,
  writeClipboardText,
} from './runtime.js';

describe('runtime adapter', () => {
  it('returns a safe no-op unsubscribe without the desktop runtime', () => {
    expect(onRuntimeEvent('test', () => {}, null)).toBeTypeOf('function');
  });

  it('prefers the native clipboard', async () => {
    const runtime = { ClipboardSetText: vi.fn().mockResolvedValue(true), ClipboardGetText: vi.fn().mockResolvedValue('native') };
    const clipboard = { writeText: vi.fn(), readText: vi.fn() };
    await writeClipboardText('value', runtime, clipboard);
    await expect(readClipboardText(runtime, clipboard)).resolves.toBe('native');
    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('value');
    expect(clipboard.writeText).not.toHaveBeenCalled();
  });

  it('reports whether a URL was handled by Wails', () => {
    const runtime = { BrowserOpenURL: vi.fn() };
    expect(openRuntimeURL('https://example.test', runtime)).toBe(true);
    expect(runtime.BrowserOpenURL).toHaveBeenCalledWith('https://example.test');
    expect(openRuntimeURL('https://example.test', null)).toBe(false);
  });

  it('resolves native dropped file paths when supported', async () => {
    const runtime = {
      CanResolveFilePaths: vi.fn().mockResolvedValue(true),
      ResolveFilePaths: vi.fn().mockResolvedValue(['/tmp/a', '', '/tmp/b']),
    };
    await expect(resolveRuntimeFilePaths(['files'], runtime)).resolves.toEqual(['/tmp/a', '/tmp/b']);
  });
});
