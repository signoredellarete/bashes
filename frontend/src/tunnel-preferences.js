const STORAGE_KEY = 'bashes.tunnelPreferences.v1';
const TUNNEL_TYPES = new Set(['socks', 'local', 'remote']);

export function loadTunnelPreference(storage, resourceId) {
  if (!storage || !resourceId) return null;
  return normalizeTunnelPreference(readPreferences(storage)[resourceId]);
}

export function saveTunnelPreference(storage, resourceId, input) {
  const preference = normalizeTunnelPreference(input);
  if (!storage || !resourceId || !preference) return false;

  try {
    const preferences = readPreferences(storage);
    preferences[resourceId] = preference;
    storage.setItem(STORAGE_KEY, JSON.stringify(preferences));
    return true;
  } catch {
    return false;
  }
}

function readPreferences(storage) {
  try {
    const preferences = JSON.parse(storage.getItem(STORAGE_KEY) || '{}');
    return preferences && typeof preferences === 'object' && !Array.isArray(preferences) ? preferences : {};
  } catch {
    return {};
  }
}

function normalizeTunnelPreference(input) {
  if (!input || !TUNNEL_TYPES.has(input.type)) return null;
  const localPort = validPort(input.localPort);
  const remotePort = validPort(input.remotePort);
  if (!localPort || !remotePort) return null;

  return {
    type: input.type,
    localHost: normalizedHost(input.localHost),
    localPort,
    remoteHost: normalizedHost(input.remoteHost),
    remotePort,
  };
}

function normalizedHost(value) {
  return String(value || '').trim() || '127.0.0.1';
}

function validPort(value) {
  const port = Number.parseInt(value, 10);
  return Number.isInteger(port) && port >= 1 && port <= 65535 ? port : 0;
}
