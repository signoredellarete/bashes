import { describe, expect, it } from 'vitest';
import { revealSidebarResource, sidebarListInset } from './sidebar-scroll.js';

describe('sidebar scrollbar spacing', () => {
  it('reserves the visual scrollbar width for overlay scrollbars', () => {
    expect(sidebarListInset(320, 320)).toBe(16);
  });

  it('does not reserve scrollbar width already removed from layout', () => {
    expect(sidebarListInset(320, 312)).toBe(8);
  });

  it('keeps the minimum card gap for wider native scrollbars', () => {
    expect(sidebarListInset(320, 308)).toBe(8);
  });
});

describe('sidebar resource visibility', () => {
  it('reveals the matching card after rendering', () => {
    const calls = [];
    const rows = [
      { dataset: { id: 'one' }, scrollIntoView: () => calls.push('one') },
      { dataset: { id: 'two' }, scrollIntoView: (options) => calls.push(options) },
    ];
    const container = { querySelectorAll: () => rows };

    revealSidebarResource(container, 'two', (callback) => callback());

    expect(calls).toEqual([{ behavior: 'smooth', block: 'nearest', inline: 'nearest' }]);
  });

  it('does nothing when the resource is no longer rendered', () => {
    const container = { querySelectorAll: () => [] };
    expect(() => revealSidebarResource(container, 'missing', (callback) => callback())).not.toThrow();
  });
});
