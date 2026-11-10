import { describe, expect, it } from 'vitest';
import { webAppWorkspaceData } from './workspace-model.js';

const webApp = { name: 'Dashboard', url: 'https://dashboard.internal', connectionMode: 'tunnel' };

describe('Web App workspace model', () => {
  it('hides the workspace without an active Web App tab', () => {
    expect(webAppWorkspaceData()).toEqual({ visible: false });
  });

  it('describes active and stopped tunnel runtimes', () => {
    expect(webAppWorkspaceData({
      webApp,
      gateway: { hostname: 'Bastion' },
      runtime: { tunnel: true, tunnelActive: true, socksPort: 45678 },
    })).toMatchObject({
      visible: true,
      mode: 'SSH tunnel via Bastion',
      status: 'Tunnel active on 127.0.0.1:45678',
      tunnelAvailable: true,
      tunnelActive: true,
    });

    expect(webAppWorkspaceData({
      webApp,
      runtime: { tunnel: true, tunnelActive: false, socksPort: 45678 },
      busy: true,
    })).toMatchObject({
      status: 'Tunnel stopped on 127.0.0.1:45678',
      tunnelActive: false,
      busy: true,
    });
  });

  it('reports direct browser runtimes without tunnel controls', () => {
    expect(webAppWorkspaceData({
      webApp: { ...webApp, connectionMode: 'direct' },
      runtime: { browser: 'Firefox' },
    })).toMatchObject({
      mode: 'Direct connection',
      status: 'Open in Firefox',
      tunnelAvailable: false,
    });
  });
});
