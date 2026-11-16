import { describe, expect, it, vi } from 'vitest';
import { createBusyController } from './busy-controller.js';

function harness() {
  const state = { busy: false, connectingResourceIds: new Set() };
  const panels = [{ setBusy: vi.fn() }, { setBusy: vi.fn() }];
  const renderSelection = vi.fn();
  const writeNotice = vi.fn();
  return {
    state, panels, renderSelection, writeNotice,
    controller: createBusyController({ state, panels, renderSelection, writeNotice }),
  };
}

describe('busy controller', () => {
  it('always restores global busy state after an error', async () => {
    const test = harness();
    expect(await test.controller.run(async () => { throw new Error('failed'); })).toBeNull();
    expect(test.state.busy).toBe(false);
    expect(test.panels[0].setBusy.mock.calls).toEqual([[true], [false]]);
    expect(test.writeNotice).toHaveBeenCalledWith('Error: failed', 'error');
  });

  it('blocks only a duplicate connection for the same resource', async () => {
    const test = harness();
    test.state.connectingResourceIds.add('host-1');

    expect(await test.controller.runConnection('host-1', vi.fn())).toBeNull();
    expect(await test.controller.runConnection('host-2', async () => 'connected')).toBe('connected');
    expect(test.state.connectingResourceIds.has('host-2')).toBe(false);
  });
});
