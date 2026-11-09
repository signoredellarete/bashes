import { describe, expect, it } from 'vitest';
import {
  LOCAL_RESOURCE_ID,
  findWorkspaceResource,
  isLocalResource,
  localResource,
  nestedResourceIDs,
  resourceIDsForSelection,
  resourceName,
  resourceTarget,
} from './model.js';

const vm = { id: 'vm-1', type: 'vm', hostname: 'vm', ip: '10.0.0.2', port: 22, user: 'root' };
const host = {
  id: 'host-1', hostname: 'server', ip: '10.0.0.1', port: 2222, user: 'admin',
  subsystems: [{ id: 'lxc-1', type: 'lxc', hostname: 'container', subsystems: [vm] }],
};
const webApp = { id: 'web-1', type: 'webapp', name: 'Console', url: 'https://console.internal', connectionMode: 'tunnel' };

describe('resource model', () => {
  it('finds top-level and nested resources with their parent', () => {
    const workspace = { hosts: [host], webApps: [webApp] };

    expect(findWorkspaceResource(workspace, host.id)).toEqual({ type: 'host', resource: host, parent: null });
    expect(findWorkspaceResource(workspace, vm.id)).toMatchObject({ type: 'vm', resource: vm, parent: host.subsystems[0] });
    expect(findWorkspaceResource(workspace, webApp.id)).toEqual({ type: 'webapp', resource: webApp, parent: null });
    expect(findWorkspaceResource(workspace, 'missing')).toBeNull();
  });

  it('only exposes localhost when local shell support is enabled', () => {
    expect(findWorkspaceResource({ localShellSupported: false }, LOCAL_RESOURCE_ID)).toBeNull();
    expect(findWorkspaceResource({ localShellSupported: true }, LOCAL_RESOURCE_ID)?.resource).toEqual(localResource());
    expect(isLocalResource(LOCAL_RESOURCE_ID)).toBe(true);
  });

  it('collects a selected resource and all nested subsystem IDs', () => {
    expect(nestedResourceIDs(host.subsystems)).toEqual(['lxc-1', 'vm-1']);
    expect(resourceIDsForSelection({ resource: host })).toEqual(['host-1', 'lxc-1', 'vm-1']);
    expect(resourceIDsForSelection(null)).toEqual([]);
  });

  it('formats terminal and Web App labels without UI state', () => {
    expect(resourceTarget(host)).toBe('admin@10.0.0.1:2222');
    expect(resourceTarget(webApp)).toBe('https://console.internal via SSH');
    expect(resourceTarget(localResource())).toBe('Local shell');
    expect(resourceName(host)).toBe('server');
    expect(resourceName(webApp)).toBe('Console');
  });
});
