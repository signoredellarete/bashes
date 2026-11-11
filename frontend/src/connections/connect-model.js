import { authKeyGroups, authValues } from './auth-model.js';

export function connectPanelData({
  resource,
  keys = [],
  sessionCount = 0,
  hasSavedPassword = false,
  statusMessage = '',
  statusKind = '',
  forcePassword = false,
} = {}) {
  const values = authValues(resource, keys, hasSavedPassword);
  if (forcePassword) values.authMethod = 'password';
  return {
    resourceId: resource?.id ?? '',
    summary: `${sessionCount > 0 ? 'New session: ' : ''}${resource?.user ?? ''}@${resource?.ip || resource?.hostname}:${resource?.port ?? 22}`,
    keyGroups: authKeyGroups(keys),
    keys,
    status: { message: statusMessage, kind: statusKind },
    values,
  };
}
