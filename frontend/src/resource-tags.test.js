import { describe, expect, it } from 'vitest';
import { collectResourceTags, filterResourceTree, mergeTagCatalog, resourceMatches, resourceTags, tagHue } from './resource-tags.js';

const resources = [
  {
    id: 'host-1', hostname: 'Proxmox', ip: '10.0.0.1', user: 'root', tags: ['Prod', 'Milan'],
    subsystems: [
      { id: 'vm-1', type: 'vm', hostname: 'Database', ip: '10.0.0.2', user: 'postgres', tags: ['prod', 'data'] },
      { id: 'vm-2', type: 'vm', hostname: 'Test', ip: '10.0.0.3', user: 'root', tags: ['lab'] },
    ],
  },
  { id: 'host-2', hostname: 'Cluster', ip: '10.0.1.1', user: 'hpc', tags: ['HPC'], subsystems: [] },
];

describe('resource tag filtering', () => {
  it('treats omitted resource tags as an empty selection', () => {
    expect(resourceTags({ id: 'untagged' })).toEqual([]);
  });

  it('collects nested tag counts case-insensitively', () => {
    expect(collectResourceTags(resources)).toEqual([
      { key: 'data', name: 'data', count: 1 },
      { key: 'hpc', name: 'HPC', count: 1 },
      { key: 'lab', name: 'lab', count: 1 },
      { key: 'milan', name: 'Milan', count: 1 },
      { key: 'prod', name: 'Prod', count: 2 },
    ]);
  });

  it('supports ANY and ALL matching', () => {
    expect(resourceMatches(resources[0], { activeTags: ['prod', 'milan'], mode: 'all' })).toBe(true);
    expect(resourceMatches(resources[0].subsystems[0], { activeTags: ['milan', 'data'], mode: 'any' })).toBe(true);
    expect(resourceMatches(resources[0].subsystems[0], { activeTags: ['milan', 'data'], mode: 'all' })).toBe(false);
  });

  it('keeps parents as context for matching children', () => {
    const filtered = filterResourceTree(resources, { activeTags: ['data'] });
    expect(filtered).toHaveLength(1);
    expect(filtered[0].resource.id).toBe('host-1');
    expect(filtered[0].children.map((child) => child.resource.id)).toEqual(['vm-1']);
  });

  it('shows descendants when their parent directly matches', () => {
    const filtered = filterResourceTree(resources, { activeTags: ['milan'] });
    expect(filtered[0].children.map((child) => child.resource.id)).toEqual(['vm-1', 'vm-2']);
  });

  it('derives stable colors without case sensitivity', () => {
    expect(tagHue('Prod')).toBe(tagHue('prod'));
  });

  it('keeps unassigned catalog tags and resource counts', () => {
    expect(mergeTagCatalog(['Unassigned', 'prod'], resources)).toEqual([
      { key: 'data', name: 'data', count: 1 },
      { key: 'hpc', name: 'HPC', count: 1 },
      { key: 'lab', name: 'lab', count: 1 },
      { key: 'milan', name: 'Milan', count: 1 },
      { key: 'prod', name: 'prod', count: 2 },
      { key: 'unassigned', name: 'Unassigned', count: 0 },
    ]);
  });
});
