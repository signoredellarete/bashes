import { describe, expect, it } from 'vitest';
import { visibleFileEntries } from './file-visibility.js';

describe('file transfer visibility', () => {
  const entries = [
    { id: '/local/Documents', type: 'folder' },
    { id: '/local/.ssh', type: 'folder' },
    { id: '/local/report.txt', type: 'file' },
    { id: '/local/.env', type: 'file' },
  ];

  it('hides dotfiles and dot-directories by default', () => {
    expect(visibleFileEntries(entries).map((entry) => entry.id)).toEqual([
      '/local/Documents',
      '/local/report.txt',
    ]);
  });

  it('returns hidden entries when explicitly enabled', () => {
    expect(visibleFileEntries(entries, true)).toEqual(entries);
  });
});
