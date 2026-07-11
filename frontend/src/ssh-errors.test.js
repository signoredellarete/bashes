import { describe, expect, it } from 'vitest';
import {
  ERROR_CODE,
  isAuthError,
  parseHostKeyMismatchError,
  parsePublicTunnelBindError,
  parseUnknownHostKeyError,
} from './ssh-errors.js';

function codedError(code, details) {
  const json = JSON.stringify({ code, message: code, details });
  const encoded = btoa(json).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
  return new Error(`Error: BASHES_ERROR:${encoded}`);
}

describe('SSH error contracts', () => {
  it('decodes unknown host key details', () => {
    const error = codedError(ERROR_CODE.unknownHostKey, {
      host: 'server.example:22',
      fingerprint: 'SHA256:test',
    });
    expect(parseUnknownHostKeyError(error)).toEqual({
      host: 'server.example:22',
      fingerprint: 'SHA256:test',
    });
  });

  it('decodes host key mismatches', () => {
    const error = codedError(ERROR_CODE.hostKeyMismatch, {
      expected: 'SHA256:old',
      actual: 'SHA256:new',
    });
    expect(parseHostKeyMismatchError(error)).toEqual({
      expected: 'SHA256:old',
      actual: 'SHA256:new',
    });
  });

  it('decodes public tunnel bind requests', () => {
    const error = codedError(ERROR_CODE.publicTunnelBind, { host: '0.0.0.0', type: 'socks' });
    expect(parsePublicTunnelBindError(error)).toEqual({ host: '0.0.0.0', type: 'socks' });
  });

  it('classifies common SSH authentication failures', () => {
    expect(isAuthError(new Error('ssh: unable to authenticate'))).toBe(true);
    expect(isAuthError(new Error('connection refused'))).toBe(false);
  });
});
