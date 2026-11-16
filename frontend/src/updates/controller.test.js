import { describe, expect, it, vi } from 'vitest';
import { createUpdateController } from './controller.js';

function memoryStorage() {
  const values = new Map();
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  };
}

describe('update controller', () => {
  it('notifies once for each automatic update version', async () => {
    const storage = memoryStorage();
    const showModal = vi.fn();
    const controller = createUpdateController({
      checkForUpdate: vi.fn(async () => ({ updateAvailable: true, latestVersion: '2.0.0' })),
      storage,
      showModal,
      openURL: vi.fn(),
      now: () => 42,
    });

    await controller.runAutomatic();
    await controller.runAutomatic();

    expect(showModal).toHaveBeenCalledOnce();
    expect(storage.getItem('bashes.updateCheck.lastAt')).toBe('42');
  });

  it('shows manual up-to-date and error dialogs', () => {
    const showModal = vi.fn();
    const controller = createUpdateController({
      checkForUpdate: vi.fn(), storage: memoryStorage(), showModal, openURL: vi.fn(),
    });

    controller.show({ updateAvailable: false, currentVersion: '2.0.0', latestVersion: '2.0.0' }, true);
    controller.showError('offline');

    expect(showModal).toHaveBeenNthCalledWith(1, expect.objectContaining({ title: 'Bashes Is Up To Date' }));
    expect(showModal).toHaveBeenNthCalledWith(2, expect.objectContaining({ title: 'Could Not Check Updates' }));
  });
});
