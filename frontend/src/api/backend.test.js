import { describe, expect, it, vi } from 'vitest';
import { createBackendClient } from './backend.js';

describe('backend client', () => {
  it('dispatches through the Wails binding contract', async () => {
    const bindings = {
      ListWorkspace: vi.fn().mockResolvedValue({ hosts: [], webApps: [], topLevelOrder: [] }),
    };
    const client = createBackendClient({ bindings });

    await expect(client.listWorkspace()).resolves.toEqual({ hosts: [], webApps: [], topLevelOrder: [] });
    expect(bindings.ListWorkspace).toHaveBeenCalledOnce();
  });

  it('fails clearly when a required desktop binding is missing', async () => {
    const client = createBackendClient({ bindings: {} });
    await expect(client.startSSHSession({})).rejects.toThrow('StartSSHSession is unavailable');
  });

  it('requires an explicit demo mode without desktop bindings', () => {
    expect(() => createBackendClient({ bindings: null, demoMode: false })).toThrow('desktop bindings are unavailable');
  });

  it('keeps demo resources behind the same API', async () => {
    const client = createBackendClient({ bindings: null, demoMode: true });
    const initial = await client.listWorkspace();
    expect(initial.hosts.map((host) => host.id)).toEqual(['demo-host']);

    const host = await client.addHost({ hostname: 'test', ip: '192.0.2.10', port: 22, user: 'demo' });
    const updated = await client.listWorkspace();
    expect(updated.hosts.some((item) => item.id === host.id)).toBe(true);
    expect(updated.topLevelOrder.at(-1)).toBe(host.id);
  });
});
