const REMOTE_ROOT_ID = '/remote';

export function normalizeRemoteStartId(value) {
  const parts = String(value ?? '').split('/').filter(Boolean);
  if (parts[0] !== 'remote' || parts.some((part) => part === '.' || part === '..')) {
    return REMOTE_ROOT_ID;
  }
  return `/${parts.join('/')}`;
}

export function remotePathChain(startId) {
  const normalized = normalizeRemoteStartId(startId);
  const parts = normalized.split('/').filter(Boolean);
  const paths = [REMOTE_ROOT_ID];
  for (let index = 2; index <= parts.length; index += 1) {
    paths.push(`/${parts.slice(0, index).join('/')}`);
  }
  return paths;
}
