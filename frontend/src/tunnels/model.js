import { authKeyGroups, authValues } from '../connections/auth-model.js';

export function tunnelMode(type = 'socks') {
  if (type === 'local') {
    return {
      title: 'Local Forward', bindLabel: 'Local Bind', portLabel: 'Local Port',
      targetHostLabel: 'Remote Host', targetPortLabel: 'Remote Port', hasTarget: true,
    };
  }
  if (type === 'remote') {
    return {
      title: 'Remote Forward', bindLabel: 'Remote Bind', portLabel: 'Remote Port',
      targetHostLabel: 'Local Host', targetPortLabel: 'Local Port', hasTarget: true,
    };
  }
  return {
    title: 'SOCKS Proxy', bindLabel: 'Bind', portLabel: 'Port',
    targetHostLabel: 'Target Host', targetPortLabel: 'Target Port', hasTarget: false,
  };
}

export function tunnelValuesForMode(values, type) {
  const next = { ...values, type };
  if (type === 'local') {
    if (!next.localPort || String(next.localPort) === '1080') next.localPort = 8080;
    if (!next.remotePort) next.remotePort = 80;
  } else if (type === 'remote') {
    if (!next.localPort || String(next.localPort) === '1080') next.localPort = 8080;
    if (!next.remotePort) next.remotePort = 3000;
  } else if (!next.localPort) {
    next.localPort = 1080;
  }
  return next;
}

export function tunnelSummary(resource, values) {
  const type = values?.type || 'socks';
  const bind = String(values?.localHost || '').trim() || '127.0.0.1';
  const port = values?.localPort || 1080;
  const targetHost = String(values?.remoteHost || '').trim() || '127.0.0.1';
  const targetPort = values?.remotePort || (type === 'remote' ? 3000 : 80);
  const login = `${resource?.user ?? ''}@${resource?.ip || resource?.hostname}`;
  if (type === 'local') return `ssh -L ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
  if (type === 'remote') return `ssh -R ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
  return `ssh -D ${bind}:${port} ${login}`;
}

export function tunnelLabel(type) {
  if (type === 'local') return 'Local tunnel';
  if (type === 'remote') return 'Remote tunnel';
  return 'SOCKS tunnel';
}

export function tunnelStatus(tunnel) {
  if (!tunnel) return '';
  return `${tunnelLabel(tunnel.type)} active on ${tunnel.localAddress} -> ${tunnel.forwardTarget || tunnel.target}`;
}

export function tunnelPanelData({ resource, keys = [], preference, hasSavedPassword = false, tunnel = null } = {}) {
  return {
    resourceId: resource?.id ?? '',
    resource,
    keys,
    keyGroups: authKeyGroups(keys),
    tunnel,
    values: {
      type: preference?.type || 'socks',
      localHost: preference?.localHost || '127.0.0.1',
      localPort: preference?.localPort || 1080,
      remoteHost: preference?.remoteHost || '127.0.0.1',
      remotePort: preference?.remotePort || 80,
      ...authValues(resource, keys, hasSavedPassword),
    },
  };
}
