import { describe, expect, it } from 'vitest';
import { countNestedResources, tagPanelData } from './panel-model.js';

const hosts = [{
  id: 'host-1',
  tags: ['Prod'],
  subsystems: [{ id: 'vm-1', tags: ['prod', 'VM'], subsystems: [] }],
}];

describe('tag panel model', () => {
  it('counts nested systems without counting Web Apps', () => {
    expect(countNestedResources(hosts)).toBe(2);
    expect(tagPanelData({ hosts, webApps: [{ id: 'app-1', tags: ['Web'] }] }).systemCount).toBe(2);
  });

  it('merges catalog and assigned tags while preserving panel state', () => {
    expect(tagPanelData({
      catalog: ['Unused'],
      hosts,
      webApps: [],
      activeFilters: ['prod'],
      busy: true,
    })).toEqual({
      tags: [
        { key: 'prod', name: 'Prod', count: 2 },
        { key: 'unused', name: 'Unused', count: 0 },
        { key: 'vm', name: 'VM', count: 1 },
      ],
      systemCount: 2,
      activeFilters: ['prod'],
      busy: true,
    });
  });
});
