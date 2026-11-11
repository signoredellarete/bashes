import { describe, expect, it } from 'vitest';
import { compactResourceName, resourceTooltip, sidebarTreeData } from './model.js';

const hosts = [{
  id: 'host-1', hostname: 'Host One', ip: '10.0.0.1', port: 22, user: 'root', tags: ['Prod'],
  subsystems: [{ id: 'vm-1', type: 'vm', hostname: 'VM One', ip: '10.0.0.2', port: 22, user: 'sid', tags: ['Lab'], subsystems: [] }],
}];
const webApps = [{ id: 'app-1', type: 'webapp', name: 'Dashboard', url: 'https://internal', connectionMode: 'direct', tags: ['Prod'] }];

describe('sidebar tree model', () => {
  it('orders top-level blocks and flattens nested resources', () => {
    const data = sidebarTreeData({ hosts, webApps, topLevelOrder: ['app-1', 'host-1'] });
    expect(data.blocks.map((block) => block.id)).toEqual(['app-1', 'host-1']);
    expect(data.blocks[1].rows.map((row) => [row.id, row.depth])).toEqual([['host-1', 0], ['vm-1', 1]]);
  });

  it('keeps localhost visible while filtering remote resources', () => {
    const data = sidebarTreeData({ hosts, webApps, query: 'missing', localShellSupported: true });
    expect(data.localRow?.id).toBe('__bashes_localhost__');
    expect(data.blocks).toEqual([]);
    expect(data.empty).toBe(true);
    expect(data.canReorder).toBe(false);
  });

  it('exposes tag and tunnel presentation without mutating resources', () => {
    const data = sidebarTreeData({
      hosts, webApps, activeTags: ['prod'],
      tunnels: [{ resourceId: 'host-1', type: 'socks' }],
      webAppRuntimes: new Map([['app-1', { tunnelActive: true, socksPort: 45678 }]]),
    });
    expect(data.blocks[0].rows[0]).toMatchObject({ tunnelActive: true, tunnelLabel: 'SOCKS tunnel', visibleTagCount: 1 });
    expect(data.blocks[0].rows[0].shownTags[0]).toMatchObject({ name: 'Prod', active: true });
    expect(data.blocks[1].rows[0]).toMatchObject({ tunnelLabel: 'SOCKS tunnel on 127.0.0.1:45678' });
  });

  it('keeps compact names and tooltips compatible with the existing UI', () => {
    expect(compactResourceName('Host One')).toBe('HO');
    expect(resourceTooltip(hosts[0])).toBe('Host One - root@10.0.0.1:22 - Prod');
  });
});
