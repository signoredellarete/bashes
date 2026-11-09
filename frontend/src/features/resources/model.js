import { isWebAppResource } from '../../web-apps/model.js';

export const LOCAL_RESOURCE_ID = '__bashes_localhost__';

export function localResource() {
  return {
    id: LOCAL_RESOURCE_ID,
    type: 'local',
    hostname: 'localhost',
    ip: '',
    port: 0,
    user: 'local',
    subsystems: [],
    local: true,
  };
}

export function findWorkspaceResource({ hosts = [], webApps = [], localShellSupported = false }, id) {
  if (localShellSupported && id === LOCAL_RESOURCE_ID) {
    return { type: 'local', resource: localResource(), parent: null };
  }
  for (const host of hosts) {
    if (host.id === id) return { type: 'host', resource: host, parent: null };
    const found = findNestedResource(host.subsystems, id, host);
    if (found) return found;
  }
  const webApp = webApps.find((item) => item.id === id);
  return webApp ? { type: 'webapp', resource: webApp, parent: null } : null;
}

export function nestedResourceIDs(subsystems = []) {
  const ids = [];
  for (const subsystem of subsystems ?? []) {
    ids.push(subsystem.id, ...nestedResourceIDs(subsystem.subsystems));
  }
  return ids;
}

export function resourceIDsForSelection(selected) {
  if (!selected?.resource?.id) return [];
  return [selected.resource.id, ...nestedResourceIDs(selected.resource.subsystems)];
}

export function resourceTarget(resource) {
  if (isLocalResource(resource)) return 'Local shell';
  if (isWebAppResource(resource)) {
    return resource.connectionMode === 'tunnel' ? `${resource.url} via SSH` : resource.url;
  }
  return `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
}

export function resourceName(resource) {
  return isWebAppResource(resource) ? resource.name : resource.hostname;
}

export function isLocalResource(resourceOrId) {
  const id = typeof resourceOrId === 'string' ? resourceOrId : resourceOrId?.id;
  return id === LOCAL_RESOURCE_ID;
}

function findNestedResource(subsystems = [], id, parent) {
  for (const subsystem of subsystems ?? []) {
    if (subsystem.id === id) return { type: subsystem.type, resource: subsystem, parent };
    const found = findNestedResource(subsystem.subsystems, id, subsystem);
    if (found) return found;
  }
  return null;
}
