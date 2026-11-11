import { describe, expect, it } from 'vitest';
import { tunnelMode, tunnelPanelData, tunnelStatus, tunnelSummary, tunnelValuesForMode } from './model.js';

const resource = { id: 'host-1', hostname: 'Gateway', ip: '10.0.0.2', user: 'root' };

describe('SSH tunnel panel model', () => {
  it('describes each tunnel mode', () => {
    expect(tunnelMode('socks')).toMatchObject({ title: 'SOCKS Proxy', hasTarget: false });
    expect(tunnelMode('local')).toMatchObject({ title: 'Local Forward', targetHostLabel: 'Remote Host', hasTarget: true });
    expect(tunnelMode('remote')).toMatchObject({ title: 'Remote Forward', targetHostLabel: 'Local Host', hasTarget: true });
  });

  it('keeps the established port defaults when changing mode', () => {
    const values = { localPort: 1080, remotePort: 0 };
    expect(tunnelValuesForMode(values, 'local')).toMatchObject({ localPort: 8080, remotePort: 80 });
    expect(tunnelValuesForMode(values, 'remote')).toMatchObject({ localPort: 8080, remotePort: 3000 });
  });

  it('renders standard SSH command summaries', () => {
    expect(tunnelSummary(resource, { type: 'socks', localHost: '127.0.0.1', localPort: 45678 }))
      .toBe('ssh -D 127.0.0.1:45678 root@10.0.0.2');
    expect(tunnelSummary(resource, {
      type: 'local', localHost: '127.0.0.1', localPort: 8080, remoteHost: 'db.internal', remotePort: 5432,
    })).toBe('ssh -L 127.0.0.1:8080:db.internal:5432 root@10.0.0.2');
  });

  it('builds persisted form values and active status', () => {
    const tunnel = { type: 'socks', localAddress: '127.0.0.1:45678', target: 'gateway' };
    expect(tunnelPanelData({
      resource,
      preference: { type: 'socks', localHost: '127.0.0.1', localPort: 45678, remoteHost: '127.0.0.1', remotePort: 80 },
      tunnel,
    })).toMatchObject({ resourceId: 'host-1', tunnel, values: { localPort: 45678, authMethod: 'agent' } });
    expect(tunnelStatus(tunnel)).toBe('SOCKS tunnel active on 127.0.0.1:45678 -> gateway');
  });
});
