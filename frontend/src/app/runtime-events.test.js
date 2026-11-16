import { describe, expect, it, vi } from 'vitest';
import { registerRuntimeEvents } from './runtime-events.js';

function harness() {
  const handlers = new Map();
  const events = {
    sshOutput: 'ssh-output', sshStatus: 'ssh-status', sshClosed: 'ssh-closed', sshTunnelClosed: 'tunnel-closed',
    webAppClosed: 'webapp-closed', webAppStatus: 'webapp-status', databaseExported: 'db-exported',
    databaseImported: 'db-imported', hostsFileImported: 'hosts-imported', hostsFilePreview: 'hosts-preview',
    openSettings: 'open-settings', openSSHKeys: 'open-keys', openAbout: 'open-about', updateCheck: 'update-check',
  };
  const state = {
    sessions: new Map(), tunnels: new Map(),
    webAppRuntimes: new Map(), selectedId: 'host-1',
  };
  const dependencies = {
    on: (name, handler) => handlers.set(name, handler), events, state,
    decodeSSHOutput: vi.fn(() => new Uint8Array([1])), writeSSHOutput: vi.fn(), discardSSHOutput: vi.fn(), markSessionClosed: vi.fn(),
    renderHosts: vi.fn(), renderTunnelStatus: vi.fn(), renderSelection: vi.fn(), writeNotice: vi.fn(),
    clearSessions: vi.fn(), closeFileTransfers: vi.fn(), refreshHosts: vi.fn(), loadTunnels: vi.fn(),
    restoreFocus: vi.fn(), showHostsFilePreview: vi.fn(), openSettings: vi.fn(), openSSHKeys: vi.fn(),
    showAbout: vi.fn(), showUpdate: vi.fn(), showUpdateError: vi.fn(),
  };
  registerRuntimeEvents(dependencies);
  return { dependencies, events, handlers, state };
}

describe('runtime event registration', () => {
  it('updates Web App runtime state before rendering', () => {
    const test = harness();
    test.handlers.get(test.events.webAppStatus)({ webAppId: 'app-1', runtimeId: 'run-1', tunnelActive: true });

    expect(test.state.webAppRuntimes.get('app-1')).toEqual(expect.objectContaining({ runtimeId: 'run-1', tunnelActive: true }));
    expect(test.dependencies.renderHosts).toHaveBeenCalledOnce();
    expect(test.dependencies.renderSelection).toHaveBeenCalledOnce();
  });

  it('cleans runtime UI before reloading an imported database', async () => {
    const test = harness();
    await test.handlers.get(test.events.databaseImported)({ message: 'Imported.' });

    expect(test.dependencies.clearSessions).toHaveBeenCalledOnce();
    expect(test.dependencies.closeFileTransfers).toHaveBeenCalledOnce();
    expect(test.state.selectedId).toBeNull();
    expect(test.dependencies.refreshHosts).toHaveBeenCalledOnce();
    expect(test.dependencies.loadTunnels).toHaveBeenCalledOnce();
  });

  it('drops buffered output for an already removed session', () => {
    const test = harness();
    test.handlers.get(test.events.sshClosed)({ sessionId: 'session-1' });
    expect(test.dependencies.discardSSHOutput).toHaveBeenCalledWith('session-1');
    expect(test.dependencies.markSessionClosed).not.toHaveBeenCalled();
  });
});
