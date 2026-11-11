import { authKeyGroups, authValues } from '../connections/auth-model.js';

export function webAppAuthPanelData({ webApp, gateway, keys = [], hasSavedPassword = false, message = '' } = {}) {
  return {
    webAppId: webApp?.id ?? '',
    gatewayId: gateway?.id ?? '',
    summary: `${webApp?.name ?? ''} via ${gateway?.user ?? ''}@${gateway?.ip || gateway?.hostname}:${gateway?.port ?? 22}`,
    keys,
    keyGroups: authKeyGroups(keys),
    status: { message, kind: message ? 'error' : '' },
    values: authValues(gateway, keys, hasSavedPassword),
  };
}
