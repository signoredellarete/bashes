import { describe, expect, it } from 'vitest';
import { normalizeRemoteStartId, remotePathChain } from './remote-path.js';

describe('remote file transfer paths', () => {
  it('normalizes a backend-provided remote start path', () => {
    expect(normalizeRemoteStartId('/remote/home/alice')).toBe('/remote/home/alice');
    expect(normalizeRemoteStartId('/local/home/alice')).toBe('/remote');
    expect(normalizeRemoteStartId('/remote/../etc')).toBe('/remote');
  });

  it('creates the path chain needed to load the initial folder', () => {
    expect(remotePathChain('/remote/work/team/alice')).toEqual([
      '/remote',
      '/remote/work',
      '/remote/work/team',
      '/remote/work/team/alice',
    ]);
  });

  it('does not duplicate the remote root', () => {
    expect(remotePathChain('/remote')).toEqual(['/remote']);
  });
});
