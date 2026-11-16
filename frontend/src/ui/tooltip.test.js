import { describe, expect, it } from 'vitest';
import { tooltipPosition } from './tooltip.js';

describe('tooltipPosition', () => {
  const anchor = { left: 100, right: 140, top: 50, bottom: 70, width: 40, height: 20 };
  const tooltip = { width: 80, height: 30 };
  const viewport = { width: 300, height: 200 };

  it('positions a right tooltip beside its anchor', () => {
    expect(tooltipPosition(anchor, tooltip, viewport)).toEqual({ left: 150, top: 45 });
  });

  it('centers a bottom tooltip and clamps it to the viewport', () => {
    expect(tooltipPosition(anchor, tooltip, viewport, 'bottom')).toEqual({ left: 80, top: 78 });
    expect(tooltipPosition({ ...anchor, left: 0, width: 10 }, tooltip, viewport, 'bottom').left).toBe(8);
  });
});
