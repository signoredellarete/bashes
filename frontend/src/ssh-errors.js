export const ERROR_CODE = Object.freeze({
  unknownHostKey: 'ssh_host_key_unknown',
  hostKeyMismatch: 'ssh_host_key_mismatch',
  publicTunnelBind: 'public_tunnel_bind',
});

export function errorDetail(error) {
  return String(error?.message ?? error ?? '').trim();
}

export function parseCodedError(error) {
  const detail = errorDetail(error);
  const marker = 'BASHES_ERROR:';
  const index = detail.indexOf(marker);
  if (index < 0) return null;
  const encoded = detail.slice(index + marker.length).split(/\s/, 1)[0];
  try {
    const normalized = encoded.replaceAll('-', '+').replaceAll('_', '/');
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=');
    return JSON.parse(atob(padded));
  } catch {
    return null;
  }
}

export function parseUnknownHostKeyError(error) {
  const coded = parseCodedError(error);
  if (coded?.code === ERROR_CODE.unknownHostKey) return coded.details ?? null;

  const match = errorDetail(error).match(/BASHES_HOST_KEY_UNKNOWN\s+resource=\S+\s+host=(\S+)\s+fingerprint=(SHA256:\S+)/);
  return match ? { host: match[1], fingerprint: match[2] } : null;
}

export function parseHostKeyMismatchError(error) {
  const coded = parseCodedError(error);
  if (coded?.code === ERROR_CODE.hostKeyMismatch) return coded.details ?? null;

  const match = errorDetail(error).match(/BASHES_HOST_KEY_MISMATCH\s+resource=\S+\s+host=\S+\s+expected=(SHA256:\S+)\s+actual=(SHA256:\S+)/);
  return match ? { expected: match[1], actual: match[2] } : null;
}

export function parsePublicTunnelBindError(error) {
  const coded = parseCodedError(error);
  if (coded?.code === ERROR_CODE.publicTunnelBind) return coded.details ?? null;

  const match = errorDetail(error).match(/BASHES_PUBLIC_TUNNEL_BIND\s+host=(\S+)\s+type=(\S+)/);
  return match ? { host: match[1], type: match[2] } : null;
}

export function isAuthError(error) {
  return /unable to authenticate|no supported methods|no SSH authentication method|handshake failed|permission denied/i.test(errorDetail(error));
}

export function isUnknownHostKeyError(error) {
  return Boolean(parseUnknownHostKeyError(error));
}

export function isHostKeyMismatchError(error) {
  return Boolean(parseHostKeyMismatchError(error));
}
