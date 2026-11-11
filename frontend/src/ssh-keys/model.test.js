import { describe, expect, it } from 'vitest';
import {
  installInputForKey,
  keyChoice,
  keyChoiceValue,
  keyGroups,
  sshKeysPanelData,
} from './model.js';

const keys = [
  { name: 'managed', source: 'bashes' },
  { name: 'id_ed25519', source: 'system', privateKey: '/home/user/.ssh/id_ed25519' },
  { name: 'work', source: 'wsl', distribution: 'Ubuntu', privateKey: '/wsl/work' },
];

describe('SSH keys panel model', () => {
  it('groups and filters keys by source', () => {
    expect(keyGroups(keys).map((group) => group.label)).toEqual(['Managed by Bashes', 'System', 'WSL: Ubuntu']);
    expect(keyGroups(keys, 'ed25519')[0].keys.map((key) => key.name)).toEqual(['id_ed25519']);
  });

  it('normalizes managed and path key choices', () => {
    expect(keyChoiceValue(keys[0])).toBe('bashes:managed');
    expect(keyChoice(keys, 'path:/tmp/id_rsa')).toMatchObject({ source: 'path', privateKey: '/tmp/id_rsa' });
    expect(installInputForKey(keys[1])).toMatchObject({ privateKeyPath: '/home/user/.ssh/id_ed25519' });
  });

  it('keeps a valid selection and exposes an optional install target', () => {
    const data = sshKeysPanelData({
      keys,
      selectedValue: 'bashes:managed',
      installEnabled: true,
      installTarget: { id: 'host', hostname: 'Server', user: 'root', ip: '10.0.0.1', port: 22 },
    });
    expect(data.selectedValue).toBe('bashes:managed');
    expect(data.installTarget.label).toBe('root@10.0.0.1:22');
  });
});
