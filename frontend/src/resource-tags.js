export function canonicalTag(value) {
  return String(value ?? '').trim().toLowerCase();
}

export function resourceTags(resource) {
  const seen = new Set();
  const tags = [];
  for (const value of resource?.tags ?? []) {
    const name = String(value ?? '').trim();
    const key = canonicalTag(name);
    if (!key || seen.has(key)) continue;
    seen.add(key);
    tags.push({ key, name });
  }
  return tags;
}

export function collectResourceTags(resources) {
  const tags = new Map();
  const visit = (resource) => {
    for (const tag of resourceTags(resource)) {
      const current = tags.get(tag.key);
      if (current) current.count += 1;
      else tags.set(tag.key, { ...tag, count: 1 });
    }
    for (const child of resource?.subsystems ?? []) visit(child);
  };
  for (const resource of resources ?? []) visit(resource);
  return [...tags.values()].sort((left, right) => left.name.localeCompare(right.name, undefined, { sensitivity: 'base' }));
}

export function mergeTagCatalog(catalog, resources) {
  const assigned = new Map(collectResourceTags(resources).map((tag) => [tag.key, tag]));
  const merged = new Map();
  for (const value of catalog ?? []) {
    const name = String(value ?? '').trim();
    const key = canonicalTag(name);
    if (!key || merged.has(key)) continue;
    merged.set(key, { key, name, count: assigned.get(key)?.count ?? 0 });
  }
  for (const tag of assigned.values()) {
    if (!merged.has(tag.key)) merged.set(tag.key, tag);
  }
  return [...merged.values()].sort((left, right) => left.name.localeCompare(right.name, undefined, { sensitivity: 'base' }));
}

export function resourceMatches(resource, { query = '', activeTags = [], mode = 'any' } = {}) {
  const normalizedQuery = String(query).trim().toLowerCase();
  const tags = resourceTags(resource);
  const searchable = [resource?.hostname, resource?.ip, resource?.user, resource?.type, ...tags.map((tag) => tag.name)]
    .map((value) => String(value ?? '').toLowerCase());
  const matchesQuery = !normalizedQuery || searchable.some((value) => value.includes(normalizedQuery));

  const selected = [...new Set(activeTags.map(canonicalTag).filter(Boolean))];
  if (selected.length === 0) return matchesQuery;
  const assigned = new Set(tags.map((tag) => tag.key));
  const matchesTags = mode === 'all'
    ? selected.every((tag) => assigned.has(tag))
    : selected.some((tag) => assigned.has(tag));
  return matchesQuery && matchesTags;
}

export function filterResourceTree(resources, filters = {}) {
  const visit = (resource, ancestorMatches = false) => {
    const directMatch = ancestorMatches || resourceMatches(resource, filters);
    const children = (resource?.subsystems ?? [])
      .map((child) => visit(child, directMatch))
      .filter(Boolean);
    if (!directMatch && children.length === 0) return null;
    return { resource, children };
  };
  return (resources ?? []).map((resource) => visit(resource)).filter(Boolean);
}

export function tagHue(value) {
  let hash = 0;
  for (const character of canonicalTag(value)) {
    hash = (hash * 31 + character.codePointAt(0)) % 360;
  }
  return hash;
}
