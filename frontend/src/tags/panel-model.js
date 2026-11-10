import { mergeTagCatalog } from '../resource-tags.js';

export function countNestedResources(resources) {
  let count = 0;
  const visit = (resource) => {
    count += 1;
    for (const child of resource?.subsystems ?? []) visit(child);
  };
  for (const resource of resources ?? []) visit(resource);
  return count;
}

export function tagPanelData({ catalog, hosts, webApps, activeFilters, busy } = {}) {
  return {
    tags: mergeTagCatalog(catalog, [...(hosts ?? []), ...(webApps ?? [])]),
    systemCount: countNestedResources(hosts),
    activeFilters: [...(activeFilters ?? [])],
    busy: Boolean(busy),
  };
}
