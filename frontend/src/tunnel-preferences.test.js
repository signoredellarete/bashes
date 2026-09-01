import { describe, expect, it } from 'vitest';
import { loadTunnelPreference, saveTunnelPreference } from './tunnel-preferences.js';

function memoryStorage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  };
}

const socksPreference = {
  type: 'socks',
  localHost: '127.0.0.1',
  localPort: 45678,
  remoteHost: '127.0.0.1',
  remotePort: 80,
};

describe('tunnel preferences', () => {
  it('stores independent preferences for each resource', () => {
    const storage = memoryStorage();
    saveTunnelPreference(storage, 'host-1', socksPreference);
    saveTunnelPreference(storage, 'vm-1', { ...socksPreference, localPort: 1081 });

    expect(loadTunnelPreference(storage, 'host-1')).toEqual(socksPreference);
    expect(loadTunnelPreference(storage, 'vm-1').localPort).toBe(1081);
  });

  it('does not persist authentication fields', () => {
    const storage = memoryStorage();
    saveTunnelPreference(storage, 'host-1', {
      ...socksPreference,
      password: 'secret',
      privateKeyPassphrase: 'secret-key',
    });

    expect(loadTunnelPreference(storage, 'host-1')).toEqual(socksPreference);
  });

  it('ignores malformed or invalid preferences', () => {
    const storage = memoryStorage({ 'bashes.tunnelPreferences.v1': '{broken' });
    expect(loadTunnelPreference(storage, 'host-1')).toBeNull();
    expect(saveTunnelPreference(storage, 'host-1', { ...socksPreference, localPort: 70000 })).toBe(false);
    expect(saveTunnelPreference(storage, 'host-1', socksPreference)).toBe(true);
    expect(loadTunnelPreference(storage, 'host-1')).toEqual(socksPreference);
  });
});
