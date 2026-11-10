export function webAppWorkspaceData({ webApp, gateway, runtime, busy } = {}) {
  if (!webApp) return { visible: false };

  let status;
  if (runtime?.tunnelActive) {
    status = `Tunnel active on 127.0.0.1:${runtime.socksPort}`;
  } else if (runtime?.tunnel) {
    status = `Tunnel stopped on 127.0.0.1:${runtime.socksPort}`;
  } else if (runtime) {
    status = `Open in ${runtime.browser}`;
  } else {
    status = webApp.connectionMode === 'tunnel' ? 'Tunnel inactive' : 'Ready to open';
  }

  return {
    visible: true,
    name: webApp.name,
    url: webApp.url,
    mode: gateway ? `SSH tunnel via ${gateway.hostname}` : 'Direct connection',
    status,
    tunnelAvailable: Boolean(runtime?.tunnel),
    tunnelActive: Boolean(runtime?.tunnelActive),
    busy: Boolean(busy),
  };
}
