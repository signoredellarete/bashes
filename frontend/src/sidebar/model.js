import {
  isLocalResource,
  localResource,
  resourceName,
  resourceTarget,
} from '../features/resources/model.js';
import {
  canonicalTag,
  filterResourceTree,
  resourceTags,
  tagHue,
  visibleResourceTags,
} from '../resource-tags.js';
import { tunnelLabel } from '../tunnels/model.js';
import { webAppMatchesFilters } from '../web-apps/model.js';

export function compactResourceName(name) {
  const clean = String(name ?? '').trim();
  if (!clean) return '?';
  const parts = clean.split(/[-_\s.]+/).filter(Boolean);
  if (parts.length >= 2) return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  return clean.slice(0, 2).toUpperCase();
}

export function resourceTooltip(resource) {
  const tags = resourceTags(resource);
  return `${resourceName(resource)} - ${resourceTarget(resource)}${tags.length ? ` - ${tags.map((tag) => tag.name).join(', ')}` : ''}`;
}

function sidebarRow(resource, type, depth, context) {
  const tags = resourceTags(resource).map((tag) => ({
    ...tag,
    hue: tagHue(tag.name),
    active: context.activeTags.includes(tag.key),
  }));
  const { shown, hidden } = visibleResourceTags(tags);
  const tunnel = context.tunnelsByResource.get(resource.id);
  const runtime = type === 'webapp' ? context.webAppRuntimes.get(resource.id) : null;
  const webAppTunnelActive = Boolean(runtime?.tunnelActive);
  return {
    id: resource.id,
    type,
    typeLabel: type === 'webapp' ? 'WebApp' : type,
    depth,
    local: isLocalResource(resource),
    webApp: type === 'webapp',
    name: resourceName(resource),
    compactName: compactResourceName(resourceName(resource)),
    target: resourceTarget(resource),
    tooltip: resourceTooltip(resource),
    tags,
    shownTags: shown,
    hiddenTags: hidden,
    visibleTagCount: shown.length + (hidden.length > 0 ? 1 : 0),
    tunnelActive: Boolean(tunnel || webAppTunnelActive),
    tunnelLabel: webAppTunnelActive
      ? `SOCKS tunnel on 127.0.0.1:${runtime.socksPort}`
      : tunnel ? tunnelLabel(tunnel.type) : '',
  };
}

function sidebarRows(node, type, depth, context) {
  return [
    sidebarRow(node.resource, type, depth, context),
    ...node.children.flatMap((child) => sidebarRows(child, child.resource.type, depth + 1, context)),
  ];
}

export function sidebarTreeData({
  hosts = [],
  webApps = [],
  topLevelOrder = [],
  query = '',
  activeTags = [],
  tagFilterMode = 'any',
  localShellSupported = false,
  tunnels = [],
  webAppRuntimes = new Map(),
  selectedId = '',
  sidebarCollapsed = false,
} = {}) {
  const normalizedQuery = String(query).trim().toLowerCase();
  const normalizedTags = [...new Set(activeTags.map(canonicalTag).filter(Boolean))];
  const filters = { query: normalizedQuery, activeTags: normalizedTags, mode: tagFilterMode };
  const filteredHosts = filterResourceTree(hosts, filters);
  const filteredWebApps = webApps.filter((webApp) => webAppMatchesFilters(webApp, filters));
  const context = {
    activeTags: normalizedTags,
    tunnelsByResource: new Map([...tunnels].map((tunnel) => [tunnel.resourceId, tunnel])),
    webAppRuntimes,
  };
  const blocksById = new Map();
  for (const node of filteredHosts) {
    blocksById.set(node.resource.id, {
      id: node.resource.id,
      webApp: false,
      rows: sidebarRows(node, 'host', 0, context),
    });
  }
  for (const webApp of filteredWebApps) {
    blocksById.set(webApp.id, {
      id: webApp.id,
      webApp: true,
      rows: [sidebarRow(webApp, 'webapp', 0, context)],
    });
  }
  const blocks = [];
  for (const id of topLevelOrder) {
    const block = blocksById.get(id);
    if (!block) continue;
    blocks.push(block);
    blocksById.delete(id);
  }
  blocks.push(...blocksById.values());

  const hasFilters = normalizedQuery !== '' || normalizedTags.length > 0;
  return {
    selectedId,
    tooltipDelay: sidebarCollapsed ? 0 : 500,
    localRow: localShellSupported ? sidebarRow(localResource(), 'local', 0, context) : null,
    blocks,
    empty: hasFilters && filteredHosts.length === 0 && filteredWebApps.length === 0,
    canReorder: !hasFilters && topLevelOrder.length > 1,
  };
}
