import { mergeTagCatalog } from '../resource-tags.js';

export function tagFilterData({ catalog, hosts, webApps, activeFilters, mode } = {}) {
  const tags = mergeTagCatalog(catalog, [...(hosts ?? []), ...(webApps ?? [])]);
  const available = new Set(tags.map((tag) => tag.key));
  return {
    tags,
    activeFilters: (activeFilters ?? []).filter((tag) => available.has(tag)),
    mode: mode === 'all' ? 'all' : 'any',
  };
}
