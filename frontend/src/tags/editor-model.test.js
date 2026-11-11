import { describe, expect, it } from 'vitest';
import {
  addTagSelection,
  normalizeTagSelection,
  removeTagSelection,
  tagEditorSuggestions,
} from './editor-model.js';

describe('tag editor model', () => {
  it('normalizes tag selections case-insensitively', () => {
    expect(normalizeTagSelection([' Prod ', 'prod', '', 'Lab'])).toEqual(['Prod', 'Lab']);
  });

  it('filters selected tags and query mismatches from suggestions', () => {
    const catalog = [
      { key: 'lab', name: 'Lab' },
      { key: 'prod', name: 'Prod' },
      { key: 'project', name: 'Project' },
    ];
    expect(tagEditorSuggestions(catalog, ['Prod'], 'pro')).toEqual([{ key: 'project', name: 'Project' }]);
  });

  it('adds and removes selections without duplicates', () => {
    expect(addTagSelection(['Prod'], 'prod')).toEqual(['Prod']);
    expect(addTagSelection(['Prod'], 'Lab')).toEqual(['Prod', 'Lab']);
    expect(removeTagSelection(['Prod', 'Lab'], 'PROD')).toEqual(['Lab']);
  });
});
