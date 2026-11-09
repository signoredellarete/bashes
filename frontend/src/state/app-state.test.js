import { describe, expect, it } from 'vitest';
import { createAppState } from './app-state.js';

describe('application state', () => {
  it('loads persisted shell preferences without sharing mutable collections', () => {
    const storage = { getItem: (key) => key === 'bashes.sidebarCollapsed' ? 'true' : null };
    const first = createAppState({ terminalSettings: { terminalFontSize: 18 }, storage });
    const second = createAppState({ terminalSettings: { terminalFontSize: 14 }, storage });

    first.sessions.set('one', {});
    first.connectingResourceIds.add('host');

    expect(first.sidebarCollapsed).toBe(true);
    expect(first.terminalFontSize).toBe(18);
    expect(second.terminalFontSize).toBe(14);
    expect(second.sessions.size).toBe(0);
    expect(second.connectingResourceIds.size).toBe(0);
  });
});
