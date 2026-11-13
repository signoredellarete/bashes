import { describe, expect, it, vi } from 'vitest';
import { resolveConflictActions, uniqueTargetConflicts } from './conflict-policy.js';

const conflicts = [
  { name: 'one.txt', targetId: '/remote/one.txt' },
  { name: 'two.txt', targetId: '/remote/two.txt' },
  { name: 'three.txt', targetId: '/remote/three.txt' },
];

describe('resolveConflictActions', () => {
  it('collects a decision for each conflict', async () => {
    const prompt = vi.fn()
      .mockResolvedValueOnce({ action: 'overwrite', applyToRemaining: false })
      .mockResolvedValueOnce({ action: 'skip', applyToRemaining: false })
      .mockResolvedValueOnce({ action: 'overwrite', applyToRemaining: false });

    await expect(resolveConflictActions(conflicts, prompt)).resolves.toEqual({
      '/remote/one.txt': 'overwrite',
      '/remote/two.txt': 'skip',
      '/remote/three.txt': 'overwrite',
    });
    expect(prompt).toHaveBeenCalledTimes(3);
  });

  it('applies one decision to all remaining conflicts', async () => {
    const prompt = vi.fn().mockResolvedValue({ action: 'skip', applyToRemaining: true });

    await expect(resolveConflictActions(conflicts, prompt)).resolves.toEqual({
      '/remote/one.txt': 'skip',
      '/remote/two.txt': 'skip',
      '/remote/three.txt': 'skip',
    });
    expect(prompt).toHaveBeenCalledTimes(1);
  });

  it('returns null when the user cancels', async () => {
    await expect(resolveConflictActions(conflicts, async () => null)).resolves.toBeNull();
  });
});

describe('uniqueTargetConflicts', () => {
  it('keeps one prompt per destination', () => {
    expect(uniqueTargetConflicts([...conflicts, conflicts[0]])).toEqual(conflicts);
  });
});
