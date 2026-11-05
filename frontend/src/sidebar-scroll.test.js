import { describe, expect, it } from 'vitest';
import { sidebarListInset } from './sidebar-scroll.js';

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
