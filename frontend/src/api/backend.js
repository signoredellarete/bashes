import { canonicalTag, mergeTagCatalog } from '../resource-tags.js';

const METHOD_NAMES = Object.freeze({
  listHosts: 'ListHosts',
  listWorkspace: 'ListWorkspace',
  listTags: 'ListTags',
  createTag: 'CreateTag',
  setResourceTags: 'SetResourceTags',
  renameTag: 'RenameTag',
  deleteTag: 'DeleteTag',
  getAppInfo: 'GetAppInfo',
  checkForUpdate: 'CheckForUpdate',
  supportsLocalShell: 'SupportsLocalShell',
  addHost: 'AddHost',
  addWebApp: 'AddWebApp',
  updateWebApp: 'UpdateWebApp',
  deleteWebApp: 'DeleteWebApp',
  addSubsystem: 'AddSubsystem',
  updateResource: 'UpdateResource',
  reorderHosts: 'ReorderHosts',
  reorderTopLevel: 'ReorderTopLevel',
  deleteResource: 'DeleteResource',
  startSSHSession: 'StartSSHSession',
  hasSavedPassword: 'HasSavedPassword',
  startLocalSession: 'StartLocalSession',
  writeSSHSession: 'WriteSSHSession',
  resizeSSHSession: 'ResizeSSHSession',
  stopSSHSession: 'StopSSHSession',
  listSSHTunnels: 'ListSSHTunnels',
  startSSHTunnel: 'StartSSHTunnel',
  stopSSHTunnel: 'StopSSHTunnel',
  listWebAppBrowsers: 'ListWebAppBrowsers',
  getWebAppBrowserSettings: 'GetWebAppBrowserSettings',
  saveWebAppBrowserSettings: 'SaveWebAppBrowserSettings',
  launchWebApp: 'LaunchWebApp',
  stopWebAppTunnel: 'StopWebAppTunnel',
  startWebAppTunnel: 'StartWebAppTunnel',
  listWebAppRuntimes: 'ListWebAppRuntimes',
  listSSHKeys: 'ListSSHKeys',
  getSSHKeyInventory: 'GetSSHKeyInventory',
  getSSHKeySettings: 'GetSSHKeySettings',
  saveSSHKeySettings: 'SaveSSHKeySettings',
  chooseSSHKeyDirectory: 'ChooseSSHKeyDirectory',
  importFromHostsFile: 'ImportFromHostsFile',
  generateSSHKey: 'GenerateSSHKey',
  importSSHKeyPair: 'ImportSSHKeyPair',
  deleteSSHKey: 'DeleteSSHKey',
  readSSHPublicKey: 'ReadSSHPublicKey',
  readSSHPublicKeyPath: 'ReadSSHPublicKeyPath',
  readSSHPrivateKey: 'ReadSSHPrivateKey',
  readSSHPrivateKeyPath: 'ReadSSHPrivateKeyPath',
  installSSHKey: 'InstallSSHKey',
});

const ARRAY_RESULTS = new Set([
  'listHosts',
  'listTags',
  'listSSHTunnels',
  'listWebAppBrowsers',
  'listWebAppRuntimes',
  'listSSHKeys',
]);
const BOOLEAN_RESULTS = new Set(['supportsLocalShell', 'hasSavedPassword']);

export function createBackendClient({
  bindings = globalThis.go?.main?.App ?? globalThis.go?.desktop?.App,
  demoMode = false,
  storage = globalThis.localStorage,
  getSessionTerminal = () => null,
  getWebAppRuntime = () => null,
  listWebAppRuntimes = () => [],
} = {}) {
  if (bindings) return createWailsClient(bindings);
  if (demoMode) {
    return createDemoClient({ storage, getSessionTerminal, getWebAppRuntime, listWebAppRuntimes });
  }
  throw new Error('Bashes desktop bindings are unavailable. Demo mode must be enabled explicitly for browser development.');
}

function createWailsClient(bindings) {
  const client = {};
  for (const [clientName, bindingName] of Object.entries(METHOD_NAMES)) {
    client[clientName] = async (...args) => {
      const method = bindings?.[bindingName];
      if (typeof method !== 'function') {
        throw new Error(`Bashes desktop binding ${bindingName} is unavailable.`);
      }
      const result = await method.apply(bindings, args);
      if (ARRAY_RESULTS.has(clientName)) return result ?? [];
      if (BOOLEAN_RESULTS.has(clientName)) return Boolean(result);
      return result;
    };
  }
  return Object.freeze(client);
}

function createDemoClient({ storage, getSessionTerminal, getWebAppRuntime, listWebAppRuntimes }) {
  const store = createDemoStore();
  const settingsStorage = storage ?? createMemoryStorage();

  return Object.freeze({
    async listHosts() {
      return clone(store.hosts);
    },
    async listWorkspace() {
      return clone({ hosts: store.hosts, webApps: store.webApps, topLevelOrder: store.topLevelOrder });
    },
    async listTags() {
      return mergeTagCatalog(store.tags, [...store.hosts, ...store.webApps]).map((tag) => tag.name);
    },
    async createTag(name) {
      name = String(name ?? '').trim();
      if (!name) throw new Error('Tag name is required');
      if (store.tags.some((tag) => canonicalTag(tag) === canonicalTag(name))) {
        throw new Error(`Tag ${name} already exists`);
      }
      store.tags.push(name);
      return name;
    },
    async setResourceTags(id, tags) {
      const resource = findDemoResource(store, id);
      if (!resource) throw new Error(`Resource ${id} not found`);
      resource.tags = [...tags];
    },
    async renameTag(currentName, newName) {
      const currentKey = canonicalTag(currentName);
      const index = store.tags.findIndex((tag) => canonicalTag(tag) === currentKey);
      if (index < 0) throw new Error(`Tag ${currentName} not found`);
      const duplicate = store.tags.findIndex((tag) => canonicalTag(tag) === canonicalTag(newName));
      if (duplicate >= 0 && duplicate !== index) throw new Error(`Tag ${newName} already exists`);
      store.tags[index] = newName;
      visitDemoResources(store, (resource) => {
        resource.tags = (resource.tags ?? []).map((tag) => canonicalTag(tag) === currentKey ? newName : tag);
      });
    },
    async deleteTag(name) {
      const key = canonicalTag(name);
      store.tags = store.tags.filter((tag) => canonicalTag(tag) !== key);
      visitDemoResources(store, (resource) => {
        resource.tags = (resource.tags ?? []).filter((tag) => canonicalTag(tag) !== key);
      });
    },
    async getAppInfo() {
      return {
        name: 'Bashes',
        version: 'dev',
        platform: 'browser',
        arch: 'demo',
        dataPath: 'demo',
        repoUrl: 'https://github.com/signoredellarete/bashes',
        readmeUrl: 'https://github.com/signoredellarete/bashes#readme',
        releasesUrl: 'https://github.com/signoredellarete/bashes/releases',
      };
    },
    async checkForUpdate() {
      return {
        currentVersion: 'dev',
        latestVersion: 'dev',
        updateAvailable: false,
        releaseUrl: 'https://github.com/signoredellarete/bashes/releases',
        repoUrl: 'https://github.com/signoredellarete/bashes',
        message: 'Update check is available only in the desktop app.',
      };
    },
    async supportsLocalShell() {
      return false;
    },
    async addHost(input) {
      const host = {
        id: `host-${Date.now()}`,
        hostname: input.hostname,
        ip: input.ip,
        port: input.port,
        user: input.user,
        tags: input.tags ?? [],
        subsystems: [],
      };
      store.hosts.push(host);
      store.topLevelOrder.push(host.id);
      return clone(host);
    },
    async addWebApp(input) {
      const webApp = { id: `webapp-${Date.now()}`, ...input };
      store.webApps.push(webApp);
      store.topLevelOrder.push(webApp.id);
      return clone(webApp);
    },
    async updateWebApp(id, input) {
      const index = store.webApps.findIndex((webApp) => webApp.id === id);
      if (index < 0) throw new Error(`Web App ${id} not found`);
      store.webApps[index] = { id, ...input };
    },
    async deleteWebApp(id) {
      const index = store.webApps.findIndex((webApp) => webApp.id === id);
      if (index < 0) throw new Error(`Web App ${id} not found`);
      store.webApps.splice(index, 1);
      store.topLevelOrder = store.topLevelOrder.filter((item) => item !== id);
    },
    async addSubsystem(parentID, input) {
      const parent = findDemoResource(store, parentID);
      if (!parent) throw new Error(`Resource ${parentID} not found`);
      if (!parent.subsystems) parent.subsystems = [];
      const subsystem = {
        id: `${input.type}-${Date.now()}`,
        type: input.type,
        hostname: input.hostname,
        ip: input.ip,
        port: input.port,
        user: input.user,
        tags: input.tags ?? [],
        subsystems: [],
      };
      parent.subsystems.push(subsystem);
      return clone(subsystem);
    },
    async updateResource(id, input) {
      const resource = findDemoResource(store, id);
      if (!resource || store.webApps.includes(resource)) throw new Error(`Resource ${id} not found`);
      if (resource.type) resource.type = input.type;
      resource.hostname = input.hostname;
      resource.ip = input.ip;
      resource.port = input.port;
      resource.user = input.user;
      resource.tags = input.tags ?? resource.tags ?? [];
    },
    async reorderHosts(order) {
      if (order.length !== store.hosts.length) throw new Error('Invalid host order');
      const byID = new Map(store.hosts.map((host) => [host.id, host]));
      store.hosts = order.map((id) => {
        const host = byID.get(id);
        if (!host) throw new Error(`Host ${id} not found`);
        return host;
      });
    },
    async reorderTopLevel(order) {
      if (order.length !== store.topLevelOrder.length) throw new Error('Invalid connection order');
      store.topLevelOrder = [...order];
    },
    async deleteResource(id) {
      const hostIndex = store.hosts.findIndex((host) => host.id === id);
      if (hostIndex >= 0) {
        store.topLevelOrder = store.topLevelOrder.filter((item) => item !== id);
        store.hosts.splice(hostIndex, 1);
        return;
      }
      for (const host of store.hosts) {
        if (deleteDemoNestedResource(host.subsystems ?? [], id)) return;
      }
      throw new Error(`Resource ${id} not found`);
    },
    async startSSHSession() {
      return `demo-session-${Date.now()}`;
    },
    async hasSavedPassword() {
      return false;
    },
    async startLocalSession() {
      throw new Error('Local shell is available only in the desktop app.');
    },
    async writeSSHSession(sessionID, data) {
      getSessionTerminal(sessionID)?.write(data);
    },
    async resizeSSHSession() {},
    async stopSSHSession() {},
    async listSSHTunnels() {
      return [];
    },
    async startSSHTunnel(input) {
      return {
        tunnelId: `demo-tunnel-${Date.now()}`,
        resourceId: input.resourceId,
        type: input.type,
        localHost: input.localHost,
        localPort: input.localPort,
        remoteHost: input.remoteHost,
        remotePort: input.remotePort,
        localAddress: `${input.localHost}:${input.localPort}`,
        target: 'demo tunnel',
        forwardTarget: input.type === 'socks' ? 'dynamic' : `${input.remoteHost}:${input.remotePort}`,
        startedAt: new Date().toISOString(),
      };
    },
    async stopSSHTunnel() {},
    async listWebAppBrowsers() {
      return [{ id: 'firefox', name: 'Firefox', path: '/usr/bin/firefox', family: 'firefox' }];
    },
    async getWebAppBrowserSettings() {
      return {
        preferredBrowserId: settingsStorage.getItem('bashes.webapps.browser') || '',
        customBrowserPath: settingsStorage.getItem('bashes.webapps.browserPath') || '',
      };
    },
    async saveWebAppBrowserSettings(input) {
      settingsStorage.setItem('bashes.webapps.browser', input.preferredBrowserId || '');
      settingsStorage.setItem('bashes.webapps.browserPath', input.customBrowserPath || '');
      return input;
    },
    async launchWebApp(input) {
      const webApp = store.webApps.find((item) => item.id === input.webAppId);
      return {
        runtimeId: `demo-webapp-${Date.now()}`,
        webAppId: input.webAppId,
        browserId: 'firefox',
        browser: 'Firefox',
        socksPort: webApp?.connectionMode === 'tunnel' ? 49100 : 0,
        tunnel: webApp?.connectionMode === 'tunnel',
        tunnelActive: webApp?.connectionMode === 'tunnel',
        startedAt: new Date().toISOString(),
      };
    },
    async stopWebAppTunnel(webAppID) {
      const runtime = getWebAppRuntime(webAppID);
      if (!runtime?.tunnel) throw new Error('Web App does not use an SSH tunnel');
      return { ...runtime, tunnelActive: false };
    },
    async startWebAppTunnel(webAppID) {
      const runtime = getWebAppRuntime(webAppID);
      if (!runtime?.tunnel) throw new Error('Web App does not use an SSH tunnel');
      return { ...runtime, tunnelActive: true };
    },
    async listWebAppRuntimes() {
      return [...listWebAppRuntimes()];
    },
    async listSSHKeys() {
      return clone(store.keys);
    },
    async getSSHKeyInventory() {
      const keys = clone(store.keys);
      return {
        keys,
        sources: ['bashes', 'system', 'custom'].map((source) => ({
          source,
          label: source === 'bashes' ? 'Managed by Bashes' : source === 'system' ? 'System SSH directory' : 'Custom directory',
          path: source === 'custom' ? settingsStorage.getItem('bashes.keys.customDirectory') || '' : '',
          count: keys.filter((key) => keySource(key) === source).length,
        })),
      };
    },
    async getSSHKeySettings() {
      return { customDirectory: settingsStorage.getItem('bashes.keys.customDirectory') || '' };
    },
    async saveSSHKeySettings(input) {
      settingsStorage.setItem('bashes.keys.customDirectory', input.customDirectory || '');
      return { customDirectory: input.customDirectory || '' };
    },
    async chooseSSHKeyDirectory() {
      return '';
    },
    async importFromHostsFile() {
      return { imported: 0, skipped: 0, hosts: [], hostnames: [] };
    },
    async generateSSHKey(input) {
      const key = { name: input.name || `bashes-${Date.now()}`, privateKey: '', publicKey: '', source: 'bashes' };
      store.keys.push(key);
      return clone(key);
    },
    async importSSHKeyPair(input) {
      const key = { name: input.name || `imported-key-${Date.now()}`, privateKey: '', publicKey: '', source: 'bashes' };
      store.keys.push(key);
      return clone(key);
    },
    async deleteSSHKey(name) {
      const index = store.keys.findIndex((key) => keySource(key) === 'bashes' && key.name === name);
      if (index >= 0) store.keys.splice(index, 1);
    },
    async readSSHPublicKey(name) {
      return `ssh-ed25519 demo ${name}`;
    },
    async readSSHPublicKeyPath(path) {
      return `ssh-ed25519 demo ${path}`;
    },
    async readSSHPrivateKey(name) {
      return `-----BEGIN OPENSSH PRIVATE KEY-----\ndemo-${name}\n-----END OPENSSH PRIVATE KEY-----`;
    },
    async readSSHPrivateKeyPath(path) {
      return `-----BEGIN OPENSSH PRIVATE KEY-----\ndemo-${path}\n-----END OPENSSH PRIVATE KEY-----`;
    },
    async installSSHKey() {},
  });
}

function createDemoStore() {
  return {
    tags: ['Production', 'Development'],
    hosts: [{
      id: 'demo-host',
      hostname: 'demo-host',
      ip: '127.0.0.1',
      port: 22,
      user: 'demo',
      subsystems: [{
        id: 'demo-vm',
        type: 'vm',
        hostname: 'demo-vm',
        ip: '127.0.0.2',
        port: 22,
        user: 'demo',
      }],
    }],
    keys: [],
    webApps: [],
    topLevelOrder: ['demo-host'],
  };
}

function findDemoResource(store, id) {
  for (const host of store.hosts) {
    if (host.id === id) return host;
    const subsystem = findDemoNestedResource(host.subsystems ?? [], id);
    if (subsystem) return subsystem;
  }
  return store.webApps.find((webApp) => webApp.id === id) ?? null;
}

function findDemoNestedResource(subsystems, id) {
  for (const subsystem of subsystems ?? []) {
    if (subsystem.id === id) return subsystem;
    const child = findDemoNestedResource(subsystem.subsystems ?? [], id);
    if (child) return child;
  }
  return null;
}

function deleteDemoNestedResource(subsystems, id) {
  for (let index = 0; index < subsystems.length; index += 1) {
    if (subsystems[index].id === id) {
      subsystems.splice(index, 1);
      return true;
    }
    if (deleteDemoNestedResource(subsystems[index].subsystems ?? [], id)) return true;
  }
  return false;
}

function visitDemoResources(store, visitor) {
  const visit = (resource) => {
    visitor(resource);
    for (const child of resource.subsystems ?? []) visit(child);
  };
  for (const host of store.hosts) visit(host);
  for (const webApp of store.webApps) visitor(webApp);
}

function keySource(key) {
  return key?.source || 'bashes';
}

function createMemoryStorage() {
  const values = new Map();
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
  };
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}
