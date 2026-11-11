import { describe, expect, it } from 'vitest';
import { webAppGatewayOptions, webAppPanelData } from './panel-model.js';

const hosts = [{
  id: 'host-1', hostname: 'Host', ip: '10.0.0.1', user: 'root',
  subsystems: [{ id: 'vm-1', hostname: 'VM', ip: '', user: 'sid', subsystems: [] }],
}];

describe('Web App panel model', () => {
  it('flattens nested SSH gateways in display order', () => {
    expect(webAppGatewayOptions(hosts)).toEqual([
      { value: 'host-1', label: 'Host (root@10.0.0.1)' },
      { value: 'vm-1', label: '  VM (sid@VM)' },
    ]);
  });

  it('builds safe defaults for a new Web App', () => {
    expect(webAppPanelData({ hosts })).toMatchObject({
      editId: '',
      title: 'Add Web App',
      submitLabel: 'Save Web App',
      values: {
        connectionMode: 'direct',
        gatewayResourceId: 'host-1',
        portMode: 'automatic',
        browserMode: 'standard',
        keepCookies: true,
        clearCacheOnClose: true,
      },
    });
  });

  it('preserves persisted edit values and normalizes a missing gateway', () => {
    const webApp = {
      id: 'app-1', name: 'Dashboard', url: 'https://dashboard.internal', tags: ['Prod'],
      connectionMode: 'tunnel', gatewayResourceId: 'removed', portMode: 'manual', localSocksPort: 45678,
      browserMode: 'private', keepCookies: true, keepLocalStorage: true, restoreSession: true, clearCacheOnClose: true,
    };
    expect(webAppPanelData({ webApp, hosts })).toMatchObject({
      editId: 'app-1',
      title: 'Edit Web App',
      values: {
        gatewayResourceId: 'host-1',
        localSocksPort: 45678,
        browserMode: 'private',
      },
    });
  });
});
