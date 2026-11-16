import { describe, expect, it, vi } from 'vitest';
import { createMessageCenter, noticeLevel } from './controller.js';

describe('message center', () => {
  it('classifies common messages', () => {
    expect(noticeLevel('Connection failed')).toBe('error');
    expect(noticeLevel('Settings saved')).toBe('success');
    expect(noticeLevel('Connecting ...')).toBe('info');
  });

  it('updates status, bounds history and renders the log', () => {
    const statusElement = {};
    const notify = vi.fn();
    const showLog = vi.fn();
    const center = createMessageCenter({
      statusElement,
      notify,
      showLog,
      now: () => new Date('2026-11-16T12:00:00Z'),
      limit: 2,
    });

    center.write('first');
    center.write('second', 'warning');
    center.write('third');
    center.openLog();

    expect(statusElement).toEqual({ textContent: 'third', title: 'third' });
    expect(center.snapshot().map(({ message }) => message)).toEqual(['third', 'second']);
    expect(notify).toHaveBeenLastCalledWith('third', 'info');
    expect(showLog).toHaveBeenCalledWith(expect.objectContaining({
      details: expect.arrayContaining([expect.arrayContaining(['third'])]),
    }));
  });
});
