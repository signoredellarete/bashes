import { describe, expect, it, vi } from 'vitest';
import { createSettingsController } from './controller.js';

function harness(overrides = {}) {
  const state = {
    terminalFontSize: 14,
    terminalScrollback: 100000,
    terminalFontFamily: 'monospace',
    terminalCopyOnSelect: true,
    webAppBrowsers: [],
    webAppBrowserSettings: {},
    pluginInventory: {
      directory: '/plugins',
      plugins: [{ id: 'org.bashes.test', name: 'Test', enabled: false, available: true, permissions: [] }],
    },
  };
  const dependencies = {
    state,
    panel: { open: vi.fn(), close: vi.fn(), update: vi.fn(), isOpen: vi.fn(() => false) },
    fontOptions: [{ label: 'System', value: 'monospace' }],
    limits: { min: 8, max: 36 },
    loadBrowserSettings: vi.fn(),
    getPluginInventory: vi.fn(async () => state.pluginInventory),
    saveBrowserSettings: vi.fn(async (settings) => settings),
    setPluginEnabled: vi.fn(),
    confirmPluginEnable: vi.fn(async () => true),
    persistTerminalSettings: vi.fn(),
    applyTerminalSettings: vi.fn(),
    restoreFocus: vi.fn(),
    notifyError: vi.fn(),
    notifySuccess: vi.fn(),
    ...overrides,
  };
  return { controller: createSettingsController(dependencies), dependencies, state };
}

describe('settings controller', () => {
  it('previews appearance without mutating saved settings', () => {
    const test = harness();
    test.controller.preview({ terminalFontSize: '20', terminalFontFamily: 'JetBrains Mono' });

    expect(test.state.terminalFontSize).toBe(14);
    expect(test.dependencies.applyTerminalSettings).toHaveBeenCalledWith(
      expect.objectContaining({ terminalFontSize: 20, terminalFontFamily: 'JetBrains Mono' }),
      { focus: false },
    );
  });

  it('does not enable plugins when confirmation is rejected', async () => {
    const test = harness({ confirmPluginEnable: vi.fn(async () => false) });
    const saved = await test.controller.submit({
      terminalFontSize: '14', terminalScrollback: '100000', terminalFontFamily: 'monospace',
      terminalCopyOnSelect: true, webAppBrowser: '', customBrowserPath: '',
      plugins: [{ id: 'org.bashes.test', name: 'Test', enabled: true, permissions: [] }],
    });

    expect(saved).toBe(false);
    expect(test.dependencies.setPluginEnabled).not.toHaveBeenCalled();
    expect(test.dependencies.saveBrowserSettings).not.toHaveBeenCalled();
  });

  it('rolls back plugin changes when a later change fails', async () => {
    const plugins = [
      { id: 'org.bashes.one', name: 'One', enabled: false, available: true },
      { id: 'org.bashes.two', name: 'Two', enabled: false, available: true },
    ];
    const setPluginEnabled = vi.fn()
      .mockResolvedValueOnce({})
      .mockRejectedValueOnce(new Error('write failed'))
      .mockResolvedValueOnce({});
    const test = harness({ setPluginEnabled });
    test.state.pluginInventory.plugins = plugins;

    const saved = await test.controller.submit({
      terminalFontSize: '14', terminalScrollback: '100000', terminalFontFamily: 'monospace',
      terminalCopyOnSelect: true, webAppBrowser: '', customBrowserPath: '',
      plugins: plugins.map((plugin) => ({ ...plugin, enabled: true })),
    });

    expect(saved).toBe(false);
    expect(setPluginEnabled).toHaveBeenNthCalledWith(3, 'org.bashes.one', false);
    expect(test.dependencies.notifyError).toHaveBeenCalledWith('Could not save plugin settings: write failed');
  });
});
