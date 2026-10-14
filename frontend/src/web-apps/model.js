import { canonicalTag, resourceTags } from '../resource-tags.js';

export function isWebAppResource(resource) {
  return Boolean(resource && typeof resource === 'object' && 'url' in resource && 'connectionMode' in resource);
}

export function webAppMatchesFilters(webApp, { query = '', activeTags = [], mode = 'any' } = {}) {
  const tags = resourceTags(webApp);
  const normalizedQuery = String(query).trim().toLowerCase();
  const searchable = [webApp?.name, webApp?.url, ...tags.map((tag) => tag.name)]
    .map((value) => String(value ?? '').toLowerCase());
  if (normalizedQuery && !searchable.some((value) => value.includes(normalizedQuery))) return false;

  const selected = [...new Set(activeTags.map(canonicalTag).filter(Boolean))];
  if (selected.length === 0) return true;
  const assigned = new Set(tags.map((tag) => tag.key));
  return mode === 'all'
    ? selected.every((tag) => assigned.has(tag))
    : selected.some((tag) => assigned.has(tag));
}
