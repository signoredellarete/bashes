import { describe, expect, it } from 'vitest';
import { resourcePanelData } from './panel-model.js';

describe('resource panel model', () => {
  it('builds defaults for a new host', () => {
    expect(resourcePanelData()).toMatchObject({
      mode: 'host',
      subsystem: false,
      title: 'Add host',
      submitLabel: 'Save host',
      values: { port: 22, type: '', tags: [] },
    });
  });

  it('describes a nested subsystem edit', () => {
    const parent = { id: 'host-1', hostname: 'Host', ip: '10.0.0.1', user: 'root' };
    const resource = { id: 'vm-1', type: 'vm', hostname: 'VM', ip: '10.0.0.2', port: 2222, user: 'sid', tags: ['Lab'] };
    expect(resourcePanelData({ mode: 'edit', parent, resource })).toMatchObject({
      parentId: 'host-1',
      editId: 'vm-1',
      subsystem: true,
      title: 'Edit subsystem',
      submitLabel: 'Save changes',
      parentSummary: 'Parent: Host (root@10.0.0.1)',
      values: {
        type: 'vm',
        hostname: 'VM',
        ip: '10.0.0.2',
        port: 2222,
        user: 'sid',
        tags: ['Lab'],
      },
    });
  });
});
