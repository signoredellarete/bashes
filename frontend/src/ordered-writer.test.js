import { describe, expect, it, vi } from 'vitest';
import { createOrderedWriter } from './ordered-writer.js';

describe('ordered terminal writer', () => {
  it('does not start a later write before the previous write completes', async () => {
    let releaseFirst;
    const firstPending = new Promise((resolve) => {
      releaseFirst = resolve;
    });
    const calls = [];
    const write = vi.fn(async (data) => {
      calls.push(data);
      if (data === 'osc-response') await firstPending;
    });
    const writeOrdered = createOrderedWriter(write);

    const first = writeOrdered('osc-response');
    const second = writeOrdered('cursor-response');
    await Promise.resolve();

    expect(calls).toEqual(['osc-response']);
    releaseFirst();
    await Promise.all([first, second]);
    expect(calls).toEqual(['osc-response', 'cursor-response']);
  });

  it('continues processing after a failed write', async () => {
    const write = vi.fn(async (data) => {
      if (data === 'failed') throw new Error('closed');
    });
    const writeOrdered = createOrderedWriter(write);

    await expect(writeOrdered('failed')).rejects.toThrow('closed');
    await expect(writeOrdered('next')).resolves.toBeUndefined();
    expect(write).toHaveBeenCalledTimes(2);
  });
});
