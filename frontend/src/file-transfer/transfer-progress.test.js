import { describe, expect, it } from 'vitest';
import { formatDuration, remainingSeconds } from './transfer-progress.js';

describe('remainingSeconds', () => {
  it('calculates a rounded-up ETA', () => {
    expect(remainingSeconds(1_000, 250, 200)).toBe(4);
  });

  it('returns zero without useful progress data', () => {
    expect(remainingSeconds(0, 0, 100)).toBe(0);
    expect(remainingSeconds(100, 10, 0)).toBe(0);
    expect(remainingSeconds(100, 100, 10)).toBe(0);
  });
});

describe('formatDuration', () => {
  it('formats seconds, minutes, and hours compactly', () => {
    expect(formatDuration(42)).toBe('42s');
    expect(formatDuration(125)).toBe('2m 5s');
    expect(formatDuration(7_500)).toBe('2h 5m');
  });
});
