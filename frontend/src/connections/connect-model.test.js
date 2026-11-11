import { describe, expect, it } from 'vitest';
import { SAVED_PASSWORD_MASK } from '../password-field.js';
import { authInput, authKeyGroups, authValues } from './auth-model.js';
import { connectPanelData } from './connect-model.js';

const keys = [
  { source: 'bashes', name: 'main' },
  { source: 'system', name: 'id_ed25519', privateKey: '/home/sid/.ssh/id_ed25519' },
  { source: 'wsl', distribution: 'Ubuntu', name: 'id_rsa', privateKey: 'C:\\wsl\\id_rsa' },
];

describe('connection authentication model', () => {
  it('groups available keys without losing their choice values', () => {
    expect(authKeyGroups(keys)).toEqual([
      { label: 'Bashes keys', options: [{ value: 'bashes:main', label: 'main', title: 'main' }] },
      {
        label: 'System keys (~/.ssh)',
        options: [{ value: 'path:/home/sid/.ssh/id_ed25519', label: 'id_ed25519 - /home/sid/.ssh/id_ed25519', title: '/home/sid/.ssh/id_ed25519' }],
      },
      {
        label: 'WSL keys (Ubuntu)',
        options: [{ value: 'path:C:\\wsl\\id_rsa', label: 'id_rsa - C:\\wsl\\id_rsa', title: 'C:\\wsl\\id_rsa' }],
      },
    ]);
  });

  it('represents a saved password as an editable mask', () => {
    expect(authValues({ auth: { method: 'password' } }, keys, true)).toMatchObject({
      authMethod: 'password', password: SAVED_PASSWORD_MASK, hadSavedPassword: true, savePassword: true,
    });
    expect(authInput({ ...authValues({ auth: { method: 'password' } }, keys, true) }, keys)).toMatchObject({
      password: '', managePassword: true, savePassword: true,
    });
  });

  it('uses a scanned path as an existing key and preserves an unknown custom path', () => {
    expect(authValues({ auth: { method: 'path', privateKeyPath: '/home/sid/.ssh/id_ed25519' } }, keys)).toMatchObject({
      authMethod: 'key', keyChoice: 'path:/home/sid/.ssh/id_ed25519',
    });
    expect(authValues({ auth: { method: 'path', privateKeyPath: '/tmp/custom' } }, keys)).toMatchObject({
      authMethod: 'path', privateKeyPath: '/tmp/custom',
    });
  });

  it('builds the connection summary and can force a password retry', () => {
    expect(connectPanelData({
      resource: { id: 'host-1', hostname: 'Host', ip: '10.0.0.2', port: 2222, user: 'root' },
      keys,
      sessionCount: 1,
      statusMessage: 'Authentication failed',
      statusKind: 'error',
      forcePassword: true,
    })).toMatchObject({
      resourceId: 'host-1',
      summary: 'New session: root@10.0.0.2:2222',
      status: { message: 'Authentication failed', kind: 'error' },
      values: { authMethod: 'password' },
    });
  });
});
