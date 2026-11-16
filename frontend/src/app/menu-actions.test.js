import { describe, expect, it, vi } from 'vitest';
import { createMenuActions } from './menu-actions.js';

function harness() {
  const dependencies = {
    showModal: vi.fn(), openRuntimeURL: vi.fn(() => false), openWindow: vi.fn(),
    importHosts: vi.fn(async () => ({ imported: 2, skipped: 1 })), refreshHosts: vi.fn(),
    withBusy: vi.fn(async (task) => task()), writeNotice: vi.fn(), restoreFocus: vi.fn(),
  };
  return { actions: createMenuActions(dependencies), dependencies };
}

describe('desktop menu actions', () => {
  it('previews hosts and exposes the confirmed import action', async () => {
    const test = harness();
    test.actions.showHostsPreview({
      path: '/etc/hosts', user: 'alice', skipped: 1,
      hosts: [{ user: 'alice', hostname: 'server', port: 22, ip: '10.0.0.1' }],
    });
    const modal = test.dependencies.showModal.mock.calls[0][0];
    expect(modal).toEqual(expect.objectContaining({ title: 'Import From Hosts File', primaryLabel: 'Import' }));

    await modal.primaryAction();
    expect(test.dependencies.importHosts).toHaveBeenCalledOnce();
    expect(test.dependencies.refreshHosts).toHaveBeenCalledOnce();
    expect(test.dependencies.writeNotice).toHaveBeenCalledWith('Imported 2 hosts from hosts file. Skipped 1.');
  });

  it('falls back to a browser window when the desktop runtime cannot open a URL', () => {
    const test = harness();
    test.actions.openURL('https://example.com');
    expect(test.dependencies.openWindow).toHaveBeenCalledWith('https://example.com');
  });
});
