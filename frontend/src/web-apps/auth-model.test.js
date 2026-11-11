import { describe, expect, it } from 'vitest';
import { SAVED_PASSWORD_MASK } from '../password-field.js';
import { webAppAuthPanelData } from './auth-model.js';

describe('Web App authentication panel model', () => {
  it('builds a gateway summary and preserves saved-password state', () => {
    expect(webAppAuthPanelData({
      webApp: { id: 'app-1', name: 'Dashboard' },
      gateway: {
        id: 'host-1', hostname: 'Gateway', ip: '10.0.0.2', port: 2222, user: 'root',
        auth: { method: 'password' },
      },
      hasSavedPassword: true,
      message: 'Authentication failed',
    })).toMatchObject({
      webAppId: 'app-1',
      gatewayId: 'host-1',
      summary: 'Dashboard via root@10.0.0.2:2222',
      status: { message: 'Authentication failed', kind: 'error' },
      values: { authMethod: 'password', password: SAVED_PASSWORD_MASK, hadSavedPassword: true },
    });
  });
});
