import { describe, expect, it } from 'vitest';
import { tagFilterData } from './filter-model.js';

describe('tag filter model', () => {
  it('drops filters that are no longer available', () => {
    expect(tagFilterData({
      catalog: ['Prod'],
      hosts: [{ tags: ['Lab'] }],
      activeFilters: ['prod', 'removed'],
      mode: 'all',
    })).toEqual({
      tags: [
        { key: 'lab', name: 'Lab', count: 1 },
        { key: 'prod', name: 'Prod', count: 0 },
      ],
      activeFilters: ['prod'],
      mode: 'all',
    });
  });

  it('normalizes an unsupported match mode to any', () => {
    expect(tagFilterData({ mode: 'unsupported' }).mode).toBe('any');
  });
});
