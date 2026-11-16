import { describe, expect, it, vi } from 'vitest';
import { createFileTransferWorkspaceController } from './workspace-controller.js';

function harness({ confirmBackground = vi.fn(async () => true) } = {}) {
  const listeners = new Map();
  const elements = [];
  const root = { append: vi.fn((element) => elements.push(element)) };
  const shell = {
    workspaceRoot: vi.fn(() => root),
    open: vi.fn(),
    hide: vi.fn(),
  };
  const unmount = vi.fn();
  const indicators = [];
  const controller = createFileTransferWorkspaceController({
    shell,
    createElement: (resource) => ({
      hidden: false,
      resource,
      addEventListener: (name, listener) => listeners.set(`${resource.id}:${name}`, listener),
      remove: vi.fn(),
    }),
    mountWorkspace: vi.fn(async () => unmount),
    confirmBackground,
    confirmCancellation: vi.fn(),
    resolveConflict: vi.fn(),
    notify: vi.fn(),
    restoreFocus: vi.fn(),
    onIndicatorChange: (indicator) => indicators.push(indicator),
  });
  return { controller, elements, indicators, listeners, shell, unmount };
}

const resource = { id: 'host-1', user: 'alice', hostname: 'server', ip: '', port: 22 };

describe('file transfer workspace controller', () => {
  it('reuses a workspace and backgrounds an active transfer', async () => {
    const test = harness();
    await test.controller.open(resource);
    await test.controller.open(resource);

    expect(test.elements).toHaveLength(1);
    test.listeners.get('host-1:bashes-file-transfer-active')({ detail: { active: true } });
    await test.controller.requestClose();

    expect(test.controller.snapshot()).toEqual({
      activeResourceId: 'host-1',
      backgroundResourceIds: ['host-1'],
      workspaceResourceIds: ['host-1'],
    });
    expect(test.indicators.at(-1)).toEqual({ active: true, backgrounded: true });
  });

  it('keeps an active workspace open when backgrounding is rejected', async () => {
    const test = harness({ confirmBackground: vi.fn(async () => false) });
    await test.controller.open(resource);
    test.listeners.get('host-1:bashes-file-transfer-active')({ detail: { active: true } });

    expect(await test.controller.requestClose()).toBe(false);
    expect(test.shell.hide).not.toHaveBeenCalled();
    expect(test.controller.snapshot().workspaceResourceIds).toEqual(['host-1']);
  });

  it('destroys inactive workspaces during close and shutdown', async () => {
    const test = harness();
    await test.controller.open(resource);
    await test.controller.requestClose();

    expect(test.unmount).toHaveBeenCalledOnce();
    expect(test.elements[0].remove).toHaveBeenCalledOnce();
    expect(test.controller.snapshot().workspaceResourceIds).toEqual([]);
  });

  it('removes a partial workspace when mounting fails', async () => {
    const test = harness();
    test.controller = createFileTransferWorkspaceController({
      shell: test.shell,
      createElement: () => test.elements[0] ?? {
        addEventListener: vi.fn(),
        remove: vi.fn(),
      },
      mountWorkspace: async () => { throw new Error('mount failed'); },
      confirmBackground: vi.fn(),
    });

    await expect(test.controller.open(resource)).rejects.toThrow('mount failed');
    expect(test.controller.snapshot().workspaceResourceIds).toEqual([]);
    expect(test.shell.hide).toHaveBeenCalledOnce();
  });
});
