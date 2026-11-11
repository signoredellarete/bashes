import { describe, expect, it } from 'vitest';
import { terminalKindLabel, terminalTabsData } from './model.js';

describe('terminal tabs model', () => {
  it('maps session state to tab presentation', () => {
    const sessions = new Map([
      ['local', { id: 'local', resourceId: 'localhost', title: 'localhost', target: 'local shell', kind: 'local' }],
      ['pending', { id: 'pending', resourceId: 'host-1', title: 'Host new', target: 'root@host:22', pending: true }],
      ['closed', { id: 'closed', resourceId: 'host-2', title: 'Host Two', target: 'root@host2:22', closed: true }],
      ['webapp', { id: 'webapp', resourceId: 'app-1', title: 'Dashboard', target: 'https://app', kind: 'webapp' }],
    ]);
    const data = terminalTabsData({
      sessions: sessions.values(),
      activeSessionId: 'pending',
      resourceTooltips: new Map([['host-1', 'Host One - root@host:22']]),
    });

    expect(data.tabs.map((tab) => tab.label)).toEqual(['local', 'new', 'closed', 'webapp']);
    expect(data.tabs[1]).toMatchObject({ active: true, tooltip: 'Host One - root@host:22' });
    expect(data.tabs[2].tooltip).toBe('Host Two - root@host2:22');
  });

  it('uses ssh as the default terminal kind', () => {
    expect(terminalKindLabel()).toBe('ssh');
  });
});
