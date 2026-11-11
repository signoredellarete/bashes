export function webAppGatewayOptions(hosts) {
  const options = [];
  const append = (resource, depth) => {
    options.push({
      value: resource.id,
      label: `${'  '.repeat(depth)}${resource.hostname} (${resource.user}@${resource.ip || resource.hostname})`,
    });
    for (const child of resource.subsystems ?? []) append(child, depth + 1);
  };
  for (const host of hosts ?? []) append(host, 0);
  return options;
}

export function webAppPanelData({ webApp = null, hosts = [], catalog = [] } = {}) {
  const gateways = webAppGatewayOptions(hosts);
  const selectedGateway = webApp?.gatewayResourceId ?? '';
  const gatewayResourceId = gateways.some((option) => option.value === selectedGateway)
    ? selectedGateway
    : gateways[0]?.value ?? '';
  return {
    editId: webApp?.id ?? '',
    title: webApp ? 'Edit Web App' : 'Add Web App',
    submitLabel: webApp ? 'Save Changes' : 'Save Web App',
    gateways,
    catalog,
    values: {
      name: webApp?.name ?? '',
      url: webApp?.url ?? '',
      tags: webApp?.tags ?? [],
      connectionMode: webApp?.connectionMode ?? 'direct',
      gatewayResourceId,
      portMode: webApp?.portMode || 'automatic',
      localSocksPort: webApp?.localSocksPort || 1080,
      browserMode: webApp?.browserMode || 'standard',
      keepCookies: webApp ? Boolean(webApp.keepCookies) : true,
      keepLocalStorage: webApp ? Boolean(webApp.keepLocalStorage) : true,
      restoreSession: webApp ? Boolean(webApp.restoreSession) : true,
      clearCacheOnClose: webApp ? Boolean(webApp.clearCacheOnClose) : true,
    },
  };
}
