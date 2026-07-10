import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import bashesLogo from './assets/bashes.png';
import './styles.css';

const demoStore = {
  hosts: [
    {
      id: 'demo-host',
      hostname: 'demo-host',
      ip: '127.0.0.1',
      port: 22,
      user: 'demo',
      subsystems: [
        { id: 'demo-vm', type: 'vm', hostname: 'demo-vm', ip: '127.0.0.2', port: 22, user: 'demo' },
      ],
    },
  ],
  keys: [],
};

const state = {
  hosts: [],
  keys: [],
  keySettings: { customDirectory: '' },
  tunnels: new Map(),
  localShellSupported: false,
  selectedId: null,
  activeSessionId: null,
  terminalFontSize: 13,
  sidebarCollapsed: localStorage.getItem('bashes.sidebarCollapsed') === 'true',
  busy: false,
  drawerMode: null,
  drawerHostId: null,
  editResourceId: null,
  confirmResolver: null,
  editContextTarget: null,
  draggedHostId: null,
  draggedSessionId: null,
  lastSessionByResource: new Map(),
  sessionFocusHistory: [],
  sessions: new Map(),
  pendingSSHOutput: new Map(),
  fileTransferWorkspaces: new Map(),
  activeFileTransferResourceId: null,
};

const FILE_TRANSFER_ENABLED = true;
const LOCAL_RESOURCE_ID = '__bashes_localhost__';
const customKeyPathHelp = [
  'Select the private key file, not the .pub file.',
  'Linux/macOS: ~/.ssh/id_ed25519',
  'Windows: C:\\Users\\YourUser\\.ssh\\id_ed25519',
].join('\n\n');

function localResource() {
  return {
    id: LOCAL_RESOURCE_ID,
    type: 'local',
    hostname: 'localhost',
    ip: '',
    port: 0,
    user: 'local',
    subsystems: [],
    local: true,
  };
}

const app = document.querySelector('#app');

app.innerHTML = `
  <aside class="sidebar">
    <header class="brand">
      <img src="${bashesLogo}" alt="" />
      <div>
        <h1>Bashes</h1>
        <span>Remote sessions</span>
      </div>
      <button id="toggle-sidebar" class="sidebar-toggle" type="button" title="Compact sidebar" aria-label="Compact sidebar">
        &lt;
      </button>
    </header>

    <div class="toolbar">
      <input id="search" type="search" placeholder="Search hosts" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" />
      <button id="open-host-panel" type="button" title="Add host">Add Host</button>
    </div>

    <section id="hosts" class="hosts" aria-label="Hosts"></section>
  </aside>

  <main class="workspace">
    <section class="session-header">
      <div class="session-titlebar">
        <p class="eyebrow">Session</p>
        <div class="session-title-row">
          <h2 id="session-title">No session selected</h2>
          <div class="session-actions">
            <button id="edit-resource" class="secondary" type="button" disabled>Edit</button>
            <button id="header-add-subsystem" class="secondary" type="button" disabled>Add Subsystem</button>
            <button id="open-keys-panel" class="secondary" type="button" disabled>Keys</button>
            <button id="delete-resource" class="secondary" type="button" disabled>Delete</button>
            <button id="open-file-transfer" class="secondary" type="button" disabled>Files</button>
            <button id="open-tunnel-panel" class="secondary" type="button" disabled>Tunnel</button>
            <button id="disconnect" class="secondary" type="button" disabled>Disconnect</button>
            <button id="connect" type="button" disabled>Connect</button>
          </div>
        </div>
      </div>
    </section>

    <section class="workbench">
      <section class="terminal-shell" aria-label="Terminal sessions">
        <div id="session-tabs" class="session-tabs" hidden></div>
        <div id="terminal-stack" class="terminal-stack">
          <div id="empty-terminal" class="empty-terminal">No active terminal session</div>
        </div>
      </section>
    </section>

    <footer class="workspace-footer">
      <p id="app-status" class="app-status" aria-live="polite"></p>
      <div class="terminal-font-controls" aria-label="Terminal font size">
        <button id="decrease-terminal-font" type="button" title="Decrease terminal font size">-</button>
        <span aria-hidden="true">A</span>
        <button id="increase-terminal-font" type="button" title="Increase terminal font size">+</button>
      </div>
    </footer>
  </main>

  <section id="resource-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-panel></div>
    <form id="resource-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow" id="resource-panel-kicker">Host</p>
          <h3 id="resource-panel-title">Add Host</h3>
        </div>
        <button class="close-panel" type="button" data-close-panel title="Close">X</button>
      </header>

      <p id="parent-host-summary" class="parent-summary" hidden></p>

      <label id="subsystem-type-field" hidden>
        <span>Type</span>
        <select name="type">
          <option value="vm">VM</option>
          <option value="lxc">LXC</option>
          <option value="docker">Docker</option>
        </select>
      </label>

      <label>
        <span>Hostname</span>
        <input name="hostname" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
      </label>
      <label>
        <span>IP / DNS</span>
        <input name="ip" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
      </label>
      <div class="form-grid">
        <label>
          <span>Port</span>
          <input name="port" type="number" min="1" max="65535" value="22" required />
        </label>
        <label>
          <span>User</span>
          <input name="user" autocomplete="username" autocapitalize="none" autocorrect="off" spellcheck="false" required />
        </label>
      </div>

      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-panel>Cancel</button>
        <button id="resource-submit" type="submit">Add Host</button>
      </footer>
    </form>
  </section>

  <section id="connect-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-connect></div>
    <form id="connect-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow">SSH</p>
          <h3>Connect</h3>
        </div>
        <button class="close-panel" type="button" data-close-connect title="Close">X</button>
      </header>

      <p id="connect-summary" class="parent-summary"></p>
      <p class="inline-status" id="connect-status" hidden></p>

      <label>
        <span>Private Key</span>
        <select name="keyName"></select>
      </label>
      <label>
        <span>Password</span>
        <input name="password" type="password" autocomplete="current-password" />
      </label>
      <label>
        <span class="field-label-with-help">
          Custom Key Path
          <span class="field-help-trigger" role="button" tabindex="0" aria-label="Custom key path help" aria-expanded="false">?</span>
          <span class="field-help-popover">${customKeyPathHelp}</span>
        </span>
        <input name="privateKeyPath" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="optional private key path" />
      </label>
      <label>
        <span>Key Passphrase</span>
        <input name="privateKeyPassphrase" type="password" autocomplete="off" />
      </label>
      <label class="checkbox-row">
        <input name="trustHostKey" type="checkbox" />
        <span>Skip host key verification for this session (insecure)</span>
      </label>

      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-connect>Cancel</button>
        <button type="submit">Connect</button>
      </footer>
    </form>
  </section>

  <section id="tunnel-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-tunnel></div>
    <form id="tunnel-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow">SSH Tunnel</p>
          <h3 id="tunnel-title">SOCKS Proxy</h3>
        </div>
        <button class="close-panel" type="button" data-close-tunnel title="Close">X</button>
      </header>

      <p id="tunnel-summary" class="parent-summary"></p>
      <p id="tunnel-status" class="tunnel-status" hidden></p>

      <label>
        <span>Mode</span>
        <select name="type">
          <option value="socks">SOCKS proxy (-D)</option>
          <option value="local">Local forward (-L)</option>
          <option value="remote">Remote forward (-R)</option>
        </select>
      </label>
      <div class="form-grid">
        <label>
          <span id="tunnel-bind-label">Bind</span>
          <input name="localHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" value="127.0.0.1" />
        </label>
        <label>
          <span id="tunnel-port-label">Port</span>
          <input name="localPort" type="number" min="1" max="65535" value="1080" required />
        </label>
      </div>
      <div id="tunnel-target-fields" class="form-grid" hidden>
        <label>
          <span id="tunnel-target-host-label">Target Host</span>
          <input name="remoteHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" value="127.0.0.1" />
        </label>
        <label>
          <span id="tunnel-target-port-label">Target Port</span>
          <input name="remotePort" type="number" min="1" max="65535" value="80" />
        </label>
      </div>
      <label>
        <span>Private Key</span>
        <select name="keyName"></select>
      </label>
      <label>
        <span>Password</span>
        <input name="password" type="password" autocomplete="current-password" />
      </label>
      <label>
        <span class="field-label-with-help">
          Custom Key Path
          <span class="field-help-trigger" role="button" tabindex="0" aria-label="Custom key path help" aria-expanded="false">?</span>
          <span class="field-help-popover">${customKeyPathHelp}</span>
        </span>
        <input name="privateKeyPath" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="optional private key path" />
      </label>
      <label>
        <span>Key Passphrase</span>
        <input name="privateKeyPassphrase" type="password" autocomplete="off" />
      </label>
      <label class="checkbox-row">
        <input name="trustHostKey" type="checkbox" />
        <span>Skip host key verification for this tunnel (insecure)</span>
      </label>

      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-tunnel>Close</button>
        <button id="stop-tunnel" class="secondary" type="button" disabled>Stop</button>
        <button id="start-tunnel" type="submit">Start Tunnel</button>
      </footer>
    </form>
  </section>

  <section id="file-transfer-modal" class="file-transfer-modal" hidden>
    <div class="file-transfer-scrim"></div>
    <section class="file-transfer-card">
      <header class="file-transfer-header">
        <div>
          <p class="eyebrow">File Transfer</p>
          <h3 id="file-transfer-title">Files</h3>
        </div>
        <div class="file-transfer-header-actions">
          <button id="file-transfer-background" class="secondary" type="button" title="Send transfer window to background">Background</button>
          <button class="close-panel" type="button" data-close-file-transfer title="Close">X</button>
        </div>
      </header>
      <div id="file-transfer-root" class="file-transfer-root"></div>
      <div id="file-transfer-resize-handle" class="file-transfer-resize-handle" title="Resize" aria-hidden="true"></div>
    </section>
  </section>

  <section id="keys-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-keys></div>
    <section class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow">SSH Keys</p>
          <h3>Manage Keys</h3>
        </div>
        <button class="close-panel" type="button" data-close-keys title="Close">X</button>
      </header>

      <form id="key-generate-form" class="compact-form">
        <label>
          <span>New Key Name</span>
          <input name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="bashes-main" />
        </label>
        <button type="submit">Generate</button>
      </form>

      <form id="key-directory-form" class="compact-form">
        <label>
          <span>Custom Keys Directory</span>
          <input name="directory" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="optional folder to scan" />
        </label>
        <button type="submit">Save</button>
      </form>
      <p class="inline-status" id="key-directory-status" hidden></p>

      <label>
        <span>Available Key</span>
        <select id="key-select"></select>
      </label>
      <label>
        <span>Public Key</span>
        <textarea id="public-key" rows="5" readonly></textarea>
      </label>

      <form id="key-install-form" class="compact-form">
        <p class="parent-summary" id="key-install-summary">Select a host or subsystem to install the key.</p>
        <p class="inline-status" id="key-install-status" hidden></p>
        <label>
          <span>Remote Password</span>
          <input name="password" type="password" autocomplete="current-password" />
        </label>
        <label class="checkbox-row">
          <input name="trustHostKey" type="checkbox" />
          <span>Skip host key verification for this install (insecure)</span>
        </label>
        <button type="submit">Install On Selected</button>
      </form>
    </section>
  </section>

  <section id="confirm-modal" class="confirm-modal" hidden>
    <div class="confirm-scrim" data-confirm-cancel></div>
    <section class="confirm-card" role="dialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-message">
      <header>
        <p class="eyebrow" id="confirm-kicker">Confirm</p>
        <h3 id="confirm-title">Delete resource</h3>
      </header>
      <p id="confirm-message"></p>
      <footer class="confirm-actions">
        <button id="confirm-cancel" class="secondary" type="button" data-confirm-cancel>Cancel</button>
        <button id="confirm-accept" class="danger" type="button">Delete</button>
      </footer>
    </section>
  </section>

  <section id="app-modal" class="app-modal" hidden>
    <div class="app-modal-scrim" data-close-app-modal></div>
    <section class="app-modal-card" role="dialog" aria-modal="true" aria-labelledby="app-modal-title" aria-describedby="app-modal-message">
      <header class="app-modal-header">
        <div>
          <p class="eyebrow" id="app-modal-kicker">Bashes</p>
          <h3 id="app-modal-title">Bashes</h3>
        </div>
        <button class="close-panel" type="button" data-close-app-modal title="Close">X</button>
      </header>
      <p id="app-modal-message"></p>
      <dl id="app-modal-details"></dl>
      <footer class="app-modal-actions">
        <button id="app-modal-secondary" class="secondary" type="button" hidden></button>
        <button id="app-modal-primary" type="button">OK</button>
      </footer>
    </section>
  </section>

  <div id="sidebar-tooltip" class="sidebar-tooltip" hidden></div>
  <div id="edit-context-menu" class="edit-context-menu" hidden>
    <button type="button" data-edit-command="cut">Cut</button>
    <button type="button" data-edit-command="copy">Copy</button>
    <button type="button" data-edit-command="paste">Paste</button>
  </div>
`;

const stack = document.querySelector('#terminal-stack');

window.addEventListener('resize', () => {
  scheduleTerminalFit();
  clampFileTransferModalSize();
});

if (globalThis.ResizeObserver) {
  const terminalResizeObserver = new ResizeObserver(() => scheduleTerminalFit());
  terminalResizeObserver.observe(stack);
}

registerSSHEvents();
registerAppEvents();

const searchInput = document.querySelector('#search');
document.querySelector('#toggle-sidebar').addEventListener('click', () => toggleSidebar());
searchInput.addEventListener('input', () => scheduleHostRender());
['beforeinput', 'keydown', 'keyup'].forEach((eventName) => {
  searchInput.addEventListener(eventName, (event) => event.stopPropagation());
});
document.querySelector('#open-host-panel').addEventListener('click', () => openResourcePanel('host'));
document.querySelector('#open-keys-panel').addEventListener('click', () => openKeysPanel());
document.querySelector('#open-tunnel-panel').addEventListener('click', () => openTunnelPanel());
document.querySelector('#open-file-transfer').addEventListener('click', () => openFileTransferModal());
document.querySelector('#edit-resource').addEventListener('click', () => openEditPanel());
document.querySelector('#header-add-subsystem').addEventListener('click', () => {
  const selected = findResource(state.selectedId);
  if (selected && !isLocalResource(selected.resource)) openResourcePanel('subsystem', selected.resource.id);
});
document.querySelector('#connect').addEventListener('click', () => openConnectPanel());
document.querySelector('#disconnect').addEventListener('click', () => disconnectActiveSession());
document.querySelector('#delete-resource').addEventListener('click', () => deleteSelectedResource());
document.querySelector('#decrease-terminal-font').addEventListener('click', () => adjustTerminalFontSize(-1));
document.querySelector('#increase-terminal-font').addEventListener('click', () => adjustTerminalFontSize(1));
document.querySelector('#resource-form').addEventListener('submit', (event) => submitResource(event));
document.querySelector('#connect-form').addEventListener('submit', (event) => submitConnect(event));
document.querySelector('#tunnel-form').addEventListener('submit', (event) => submitTunnel(event));
document.querySelector('#tunnel-form').addEventListener('input', () => updateTunnelSummary());
document.querySelector('#tunnel-form').elements.type.addEventListener('change', () => updateTunnelMode());
registerAuthChoiceSync(document.querySelector('#connect-form'));
registerAuthChoiceSync(document.querySelector('#tunnel-form'));
document.querySelector('#stop-tunnel').addEventListener('click', () => stopSelectedTunnel());
document.querySelector('#key-generate-form').addEventListener('submit', (event) => submitGenerateKey(event));
document.querySelector('#key-directory-form').addEventListener('submit', (event) => submitKeyDirectory(event));
document.querySelector('#key-install-form').addEventListener('submit', (event) => submitInstallKey(event));
document.querySelector('#key-select').addEventListener('change', () => renderSelectedPublicKey());
document.querySelectorAll('[data-close-panel]').forEach((element) => {
  element.addEventListener('click', () => closeResourcePanel());
});
document.querySelectorAll('[data-close-connect]').forEach((element) => {
  element.addEventListener('click', () => closeConnectPanel());
});
document.querySelectorAll('[data-close-tunnel]').forEach((element) => {
  element.addEventListener('click', () => closeTunnelPanel());
});
document.querySelectorAll('[data-close-keys]').forEach((element) => {
  element.addEventListener('click', () => closeKeysPanel());
});
document.querySelectorAll('[data-close-file-transfer]').forEach((element) => {
  element.addEventListener('click', () => closeFileTransferModal());
});
document.querySelector('#file-transfer-background').addEventListener('click', () => backgroundFileTransferModal());
registerFileTransferModalResize();
registerFileTransferModalDrag();
document.querySelectorAll('[data-confirm-cancel]').forEach((element) => {
  element.addEventListener('click', () => resolveConfirmModal(false));
});
document.querySelector('#confirm-accept').addEventListener('click', () => resolveConfirmModal(true));
document.querySelectorAll('[data-close-app-modal]').forEach((element) => {
  element.addEventListener('click', () => closeAppModal());
});
document.querySelectorAll('.field-help-trigger').forEach((button) => {
  button.addEventListener('click', (event) => toggleFieldHelp(event));
  button.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      toggleFieldHelp(event);
    }
  });
});
document.querySelector('#app-modal-primary').addEventListener('click', () => runAppModalAction('primary'));
document.querySelector('#app-modal-secondary').addEventListener('click', () => runAppModalAction('secondary'));
document.querySelector('#edit-context-menu').addEventListener('click', (event) => runEditContextCommand(event));
document.addEventListener('contextmenu', (event) => openEditContextMenu(event));
document.addEventListener('pointerdown', (event) => {
  if (!event.target.closest?.('#edit-context-menu')) hideEditContextMenu();
  if (!event.target.closest?.('.field-label-with-help')) closeFieldHelpPopovers();
});
window.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && !document.querySelector('#confirm-modal').hidden) {
    resolveConfirmModal(false);
  } else if (event.key === 'Escape' && !document.querySelector('#app-modal').hidden) {
    closeAppModal();
  } else if (event.key === 'Escape') {
    closeFieldHelpPopovers();
    hideEditContextMenu();
  }
});

await loadCapabilities();
await loadHosts();
await loadKeySettings();

function toggleFieldHelp(event) {
  event.preventDefault();
  event.stopPropagation();

  const button = event.currentTarget;
  const willOpen = button.getAttribute('aria-expanded') !== 'true';
  closeFieldHelpPopovers(button);
  button.setAttribute('aria-expanded', willOpen ? 'true' : 'false');
}

function closeFieldHelpPopovers(except = null) {
  document.querySelectorAll('.field-help-trigger[aria-expanded="true"]').forEach((button) => {
    if (button !== except) {
      button.setAttribute('aria-expanded', 'false');
    }
  });
}
await loadKeys();
await loadTunnels();
applySidebarState();
await openInitialLocalSession();
schedulePeriodicUpdateCheck();

async function loadCapabilities() {
  state.localShellSupported = await apiSupportsLocalShell();
}

async function loadHosts() {
  await withBusy(async () => {
    await refreshHosts();
  });
}

async function refreshHosts() {
  state.hosts = await apiListHosts();
  const activeSession = state.sessions.get(state.activeSessionId);
  if (activeSession && findResource(activeSession.resourceId)) {
    state.selectedId = activeSession.resourceId;
  } else if (state.selectedId && !findResource(state.selectedId)) {
    state.selectedId = null;
  }
  if (!state.selectedId && !activeSession && state.localShellSupported) {
    state.selectedId = LOCAL_RESOURCE_ID;
  } else if (!state.selectedId && !activeSession && state.hosts.length > 0) {
    state.selectedId = state.hosts[0].id;
  }
  renderHosts(searchInput.value);
  renderSelection();
}

async function loadKeys() {
  state.keys = await apiListSSHKeys();
  renderKeyOptions();
}

async function loadKeySettings() {
  state.keySettings = await apiGetSSHKeySettings();
  renderKeySettings();
}

async function loadTunnels() {
  const tunnels = await apiListSSHTunnels();
  state.tunnels = new Map(tunnels.map((tunnel) => [tunnel.tunnelId, tunnel]));
  renderHosts(searchInput.value);
}

async function submitResource(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const editing = state.drawerMode === 'edit' ? findResource(state.editResourceId) : null;
  const type = state.drawerMode === 'subsystem' || editing?.type !== 'host' ? form.elements.type.value : '';
  const input = endpointInput(form, type);

  await withBusy(async () => {
    if (state.drawerMode === 'edit') {
      const resource = findResource(state.editResourceId)?.resource;
      await apiUpdateResource(state.editResourceId, input);
      for (const session of sessionsForResource(state.editResourceId)) {
        session.title = input.hostname;
        session.target = `${input.user}@${input.ip || input.hostname}:${input.port}`;
      }
      writeNotice(`Updated ${resource?.hostname ?? 'resource'}.`);
    } else if (state.drawerMode === 'subsystem') {
      const subsystem = await apiAddSubsystem(state.drawerHostId, input);
      writeNotice(`Added ${subsystem.type} ${subsystem.hostname}.`);
    } else {
      const host = await apiAddHost(input);
      writeNotice(`Added host ${host.hostname}.`);
    }
    closeResourcePanel();
    await refreshHosts();
    renderTabs();
  });
}

async function submitConnect(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) {
    await quickConnect(selected);
    closeConnectPanel();
    return;
  }

  const form = event.currentTarget;
  await withBusy(async () => {
    const target = `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
    const auth = authInputFromForm(form);
    setConnectStatus('', '');
    writeNotice(`Connecting to ${target} ...`);
    try {
      const sessionID = await startSSHSessionWithHostKeyPrompt({
        resourceId: selected.id,
        ...auth,
        password: form.elements.password.value,
        trustHostKey: form.elements.trustHostKey.checked,
        cols: 120,
        rows: 32,
      }, selected, 'Trust and connect');
      createSession(sessionID, selected);
      closeConnectPanel();
      await refreshHosts();
      resizeActiveSession();
    } catch (error) {
      const message = connectErrorMessage(error, selected);
      setConnectStatus(message, 'error');
      writeNotice(message);
      if (isAuthError(error)) focusConnectPasswordInput(form);
    }
  });
}

async function submitTunnel(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const active = tunnelForResource(selected.id);
  if (active) {
    writeNotice(`Tunnel already active on ${active.localAddress}.`);
    renderTunnelStatus();
    return;
  }

  const form = event.currentTarget;
  await withBusy(async () => {
    const auth = authInputFromForm(form);
    try {
      const tunnel = await startSSHTunnelWithHostKeyPrompt({
        resourceId: selected.id,
        type: form.elements.type.value,
        localHost: form.elements.localHost.value.trim(),
        localPort: Number.parseInt(form.elements.localPort.value, 10),
        remoteHost: form.elements.remoteHost.value.trim(),
        remotePort: Number.parseInt(form.elements.remotePort.value, 10),
        ...auth,
        password: form.elements.password.value,
        trustHostKey: form.elements.trustHostKey.checked,
      }, selected);
      state.tunnels.set(tunnel.tunnelId, tunnel);
      await refreshHosts();
      renderHosts(searchInput.value);
      renderTunnelStatus();
      writeNotice(`${tunnelLabel(tunnel.type)} active on ${tunnel.localAddress}.`);
    } catch (error) {
      writeNotice(connectErrorMessage(error, selected));
    }
  });
}

async function quickConnect(resource) {
  await withBusy(async () => {
    try {
      if (isLocalResource(resource)) {
        writeNotice('Starting local shell ...');
        const sessionID = await apiStartLocalSession({
          cols: 120,
          rows: 32,
        });
        createSession(sessionID, resource, 'local');
        resizeActiveSession();
        return;
      }

      writeNotice(`Connecting to ${resource.user}@${resource.ip || resource.hostname}:${resource.port} ...`);
      const sessionID = await startSSHSessionWithHostKeyPrompt({
        resourceId: resource.id,
        ...authInputFromPreference(resource),
        trustHostKey: trustHostKeyFromPreference(resource),
        cols: 120,
        rows: 32,
      }, resource, 'Trust and connect');
      createSession(sessionID, resource);
      await refreshHosts();
      resizeActiveSession();
    } catch (error) {
      const message = connectErrorMessage(error, resource);
      writeNotice(message);
      if (isLocalResource(resource)) {
        const pending = pendingSessionForResource(resource.id);
        if (pending) removeSessionFromUI(pending.id);
        return;
      }
      await openConnectPanel(message, 'error');
    }
  });
}

async function submitGenerateKey(event) {
  event.preventDefault();
  const form = event.currentTarget;

  await withBusy(async () => {
    const key = await apiGenerateSSHKey({ name: form.elements.name.value.trim() });
    form.reset();
    await loadKeys();
    document.querySelector('#key-select').value = keyChoiceValue({ ...key, source: 'bashes' });
    await renderSelectedPublicKey();
    writeNotice(`Generated SSH key ${key.name}.`);
  });
}

async function submitKeyDirectory(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const directory = form.elements.directory.value.trim();

  await withBusy(async () => {
    try {
      state.keySettings = await apiSaveSSHKeySettings({ customDirectory: directory });
      renderKeySettings();
      await loadKeys();
      const customCount = state.keys.filter((key) => keySource(key) === 'custom').length;
      if (state.keySettings.customDirectory && customCount === 0) {
        setKeyDirectoryStatus('Saved, but no SSH keys were found in that directory.', 'pending');
      } else if (state.keySettings.customDirectory) {
        setKeyDirectoryStatus(`Saved. Found ${customCount} custom key${customCount === 1 ? '' : 's'}.`, 'success');
      } else {
        setKeyDirectoryStatus('Custom keys directory cleared.', 'success');
      }
    } catch (error) {
      const message = `Could not save custom keys directory: ${error?.message ?? error}`;
      setKeyDirectoryStatus(message, 'error');
      writeNotice(message);
      window.setTimeout(() => form.elements.directory.focus(), 0);
    }
  });
}

async function submitInstallKey(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  const keyChoice = selectedKeyChoice(document.querySelector('#key-select'));
  if (!selected || isLocalResource(selected)) {
    setKeyInstallStatus('Select a host or subsystem before installing a key.', 'error');
    writeNotice('Select a host or subsystem before installing a key.');
    return;
  }
  if (!keyChoice) {
    setKeyInstallStatus('Select or generate an SSH key before installing it.', 'error');
    writeNotice('Select or generate an SSH key before installing it.');
    return;
  }

  const form = event.currentTarget;
  const keyLabel = keyChoiceLabel(keyChoice);
  setKeyInstallStatus(`Installing key ${keyLabel} on ${selected.hostname} ...`, 'pending');
  await withBusy(async () => {
    try {
      await installSSHKeyWithHostKeyPrompt({
        resourceId: selected.id,
        ...installInputFromKeyChoice(keyChoice),
        password: form.elements.password.value,
        trustHostKey: form.elements.trustHostKey.checked,
      }, selected);
      form.reset();
      form.elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
      await refreshHosts();
      setKeyInstallStatus(`Installed key ${keyLabel} on ${selected.hostname}.`, 'success');
      writeNotice(`Installed key ${keyLabel} on ${selected.hostname}.`);
    } catch (error) {
      const message = keyInstallErrorMessage(error, selected);
      setKeyInstallStatus(message, 'error');
      writeNotice(message);
      window.setTimeout(() => form.elements.password.focus(), 0);
    }
  });
}

async function deleteSelectedResource() {
  const selected = findResource(state.selectedId);
  if (!selected) return;
  if (isLocalResource(selected.resource)) return;
  if (!(await confirmDeleteResource(selected))) return;

  await withBusy(async () => {
    const resourceIDs = resourceIDsForSelection(selected);
    for (const resourceID of resourceIDs) {
      for (const session of sessionsForResource(resourceID)) {
        await stopSession(session.id);
      }
      for (const tunnel of tunnelsForResource(resourceID)) {
        await stopTunnel(tunnel.tunnelId);
      }
    }
    await apiDeleteResource(selected.resource.id);
    writeNotice(`Deleted ${selected.resource.hostname}.`);
    if (!state.activeSessionId) state.selectedId = selected.parent?.id ?? null;
    await refreshHosts();
  });
}

function resourceIDsForSelection(selected) {
  const ids = [selected.resource.id];
  ids.push(...nestedResourceIDs(selected.resource.subsystems ?? []));
  return ids;
}

function nestedResourceIDs(subsystems) {
  const ids = [];
  for (const subsystem of subsystems ?? []) {
    ids.push(subsystem.id, ...nestedResourceIDs(subsystem.subsystems ?? []));
  }
  return ids;
}

function confirmDeleteResource(selected) {
  const subsystemCount = nestedResourceIDs(selected.resource.subsystems ?? []).length;
  const extra = subsystemCount > 0
    ? `\n\nThis will also delete ${subsystemCount} subsystem${subsystemCount === 1 ? '' : 's'}.`
    : '';
  return openConfirmModal({
    kicker: 'Delete',
    title: `Delete ${selected.resource.hostname}?`,
    message: `This action cannot be undone.${extra}`,
    confirmLabel: 'Delete',
  });
}

function openConfirmModal({ kicker = 'Confirm', title, message, confirmLabel = 'Confirm' }) {
  const modal = document.querySelector('#confirm-modal');
  const cancel = document.querySelector('#confirm-cancel');
  document.querySelector('#confirm-kicker').textContent = kicker;
  document.querySelector('#confirm-title').textContent = title;
  document.querySelector('#confirm-message').textContent = message;
  document.querySelector('#confirm-accept').textContent = confirmLabel;

  if (state.confirmResolver) {
    state.confirmResolver(false);
  }

  modal.hidden = false;
  requestAnimationFrame(() => modal.classList.add('open'));
  requestAnimationFrame(() => cancel.focus());

  return new Promise((resolve) => {
    state.confirmResolver = resolve;
  });
}

function resolveConfirmModal(confirmed) {
  const modal = document.querySelector('#confirm-modal');
  if (!modal || modal.hidden) return;

  modal.classList.remove('open');
  modal.hidden = true;
  restoreTerminalFocusAfterOverlay();

  const resolve = state.confirmResolver;
  state.confirmResolver = null;
  if (resolve) resolve(confirmed);
}

let appModalActions = {};

function showAppModal(options) {
  const modal = document.querySelector('#app-modal');
  document.querySelector('#app-modal-kicker').textContent = options.kicker ?? 'Bashes';
  document.querySelector('#app-modal-title').textContent = options.title ?? 'Bashes';
  document.querySelector('#app-modal-message').textContent = options.message ?? '';

  const details = document.querySelector('#app-modal-details');
  const entries = options.details ?? [];
  details.replaceChildren(...entries.flatMap(([label, value]) => {
    const term = document.createElement('dt');
    term.textContent = label;
    const description = document.createElement('dd');
    description.textContent = value;
    return [term, description];
  }));
  details.hidden = entries.length === 0;

  appModalActions = {};
  configureAppModalButton('primary', options.primaryLabel ?? 'OK', options.primaryAction);
  configureAppModalButton('secondary', options.secondaryLabel, options.secondaryAction);

  modal.hidden = false;
  requestAnimationFrame(() => modal.classList.add('open'));
}

function configureAppModalButton(name, label, action) {
  const button = document.querySelector(`#app-modal-${name}`);
  button.hidden = !label;
  if (!label) return;
  button.textContent = label;
  appModalActions[name] = action;
}

function runAppModalAction(name) {
  const action = appModalActions[name];
  closeAppModal();
  if (typeof action === 'function') action();
}

function closeAppModal() {
  const modal = document.querySelector('#app-modal');
  if (!modal || modal.hidden) return;
  modal.classList.remove('open');
  modal.hidden = true;
  appModalActions = {};
  restoreTerminalFocusAfterOverlay();
}

async function stopSelectedTunnel() {
  const selected = findResource(state.selectedId)?.resource;
  const tunnel = selected ? tunnelForResource(selected.id) : null;
  if (!tunnel) return;
  await withBusy(async () => {
    await stopTunnel(tunnel.tunnelId);
    renderTunnelStatus();
    writeNotice(`Stopped tunnel on ${tunnel.localAddress}.`);
  });
}

async function stopTunnel(tunnelID) {
  await apiStopSSHTunnel(tunnelID);
  state.tunnels.delete(tunnelID);
  renderHosts(searchInput.value);
}

async function disconnectActiveSession() {
  if (!state.activeSessionId) return;
  await stopSession(state.activeSessionId);
}

async function stopSession(sessionID) {
  const session = state.sessions.get(sessionID);
  removeSessionFromUI(sessionID);
  if (session && !session.pending && !session.closed) {
    await apiStopSSHSession(sessionID);
  }
}

function markSessionClosed(sessionID, message = '') {
  const session = state.sessions.get(sessionID);
  if (!session) {
    state.pendingSSHOutput.delete(sessionID);
    return;
  }

  flushPendingSSHOutput(sessionID);
  session.closed = true;
  session.pending = false;
  if (session.terminal) {
    session.terminal.options.disableStdin = true;
    const reason = String(message || `${terminalKindLabel(session.kind)} session closed`).trim();
    session.terminal.write(`\r\n[Session closed: ${reason}]\r\n`);
  }
  if (session.element) session.element.classList.add('closed-pane');
  if (state.lastSessionByResource.get(session.resourceId) === sessionID) {
    state.lastSessionByResource.delete(session.resourceId);
  }
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
}

function removeSessionFromUI(sessionID) {
  const session = state.sessions.get(sessionID);
  state.sessions.delete(sessionID);
  state.pendingSSHOutput.delete(sessionID);
  forgetSessionFocus(sessionID);
  if (session && state.lastSessionByResource.get(session.resourceId) === sessionID) {
    state.lastSessionByResource.delete(session.resourceId);
  }
  if (session?.terminal) session.terminal.dispose();
  if (session?.element) session.element.remove();
  if (state.activeSessionId === sessionID) {
    setActiveSession(lastFocusedSessionID());
  }
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function endpointInput(form, type) {
  return {
    hostname: form.elements.hostname.value.trim(),
    ip: form.elements.ip.value.trim(),
    port: Number.parseInt(form.elements.port.value, 10),
    user: form.elements.user.value.trim(),
    ...(type ? { type } : {}),
  };
}

let searchRenderFrame = 0;

function scheduleHostRender() {
  if (searchRenderFrame) cancelAnimationFrame(searchRenderFrame);
  searchRenderFrame = requestAnimationFrame(() => {
    searchRenderFrame = 0;
    renderHosts(searchInput.value);
    renderSelection();
  });
}

function renderHosts(filter = '') {
  const container = document.querySelector('#hosts');
  const query = filter.trim().toLowerCase();
  const canReorder = query === '';
  const rows = [];

  if (state.localShellSupported) {
    rows.push(resourceRow(localResource(), 'local', 0, LOCAL_RESOURCE_ID, false));
  }
  for (const host of state.hosts) {
    rows.push(...resourceRows(host, 'host', 0, host.id, canReorder));
  }

  const visibleRows = rows.filter((row) => row.search.includes(query));
  container.replaceChildren(...visibleRows.map((row) => row.element));
}

function resourceRows(resource, type, depth, rootHostId, canReorder) {
  const rows = [resourceRow(resource, type, depth, rootHostId, canReorder)];
  for (const subsystem of resource.subsystems ?? []) {
    rows.push(...resourceRows(subsystem, subsystem.type, depth + 1, rootHostId, canReorder));
  }
  return rows;
}

function resourceRow(resource, type, depth = 0, rootHostId = resource.id, canReorder = true) {
  const row = document.createElement('div');
  row.className = `host-row ${depth > 0 ? 'child' : ''}`;
  if (isLocalResource(resource)) row.classList.add('local-row');
  row.dataset.id = resource.id;
  row.dataset.rootHostId = rootHostId;
  row.style.setProperty('--tree-offset', `${depth * 18}px`);
  const target = resourceTarget(resource);
  const tooltip = `${resource.hostname} - ${target}`;
  const tunnel = tunnelForResource(resource.id);

  const selectButton = document.createElement('button');
  selectButton.type = 'button';
  selectButton.className = 'host-select';
  if (tunnel) selectButton.classList.add('tunnel-active');
  selectButton.draggable = canReorder && !isLocalResource(resource);
  selectButton.title = tooltip;
  selectButton.dataset.tooltip = tooltip;
  selectButton.innerHTML = `
    <span class="type"></span>
    <span class="compact-name" aria-hidden="true"></span>
    <span class="details">
      <span class="resource-name-line">
        <strong></strong>
      </span>
      <small></small>
    </span>
  `;
  if (tunnel) {
    const chip = document.createElement('span');
    chip.className = 'smart-chip tunnel-chip';
    chip.textContent = 'tun';
    chip.title = tunnelLabel(tunnel.type);
    selectButton.querySelector('.resource-name-line').append(chip);
  }
  selectButton.querySelector('.type').textContent = type;
  selectButton.querySelector('.compact-name').textContent = compactResourceName(resource.hostname);
  selectButton.querySelector('strong').textContent = resource.hostname;
  selectButton.querySelector('small').textContent = target;
  selectButton.addEventListener('click', () => {
    selectResource(resource);
  });
  selectButton.addEventListener('dblclick', async () => {
    state.selectedId = resource.id;
    const realSessions = realSessionsForResource(resource.id);
    if (realSessions.length > 0 && !isLocalResource(resource)) {
      selectResource(resource);
      await openConnectPanel();
      return;
    }
    createPendingTab(resource);
    quickConnect(resource);
  });
  selectButton.addEventListener('mouseenter', () => showSidebarTooltip(selectButton));
  selectButton.addEventListener('focus', () => showSidebarTooltip(selectButton));
  selectButton.addEventListener('mouseleave', () => hideSidebarTooltip());
  selectButton.addEventListener('blur', () => hideSidebarTooltip());
  if (canReorder && !isLocalResource(resource)) {
    selectButton.addEventListener('dragstart', (event) => startHostDrag(event, rootHostId));
    selectButton.addEventListener('dragend', () => endHostDrag());
    selectButton.addEventListener('dragover', (event) => previewHostDrop(event, row));
    selectButton.addEventListener('dragleave', () => clearHostDropPreview(row));
    selectButton.addEventListener('drop', (event) => dropHostBlock(event, row));
  }
  row.append(selectButton);

  return {
    element: row,
    search: `${resource.hostname} ${resource.ip} ${resource.user} ${type}`.toLowerCase(),
  };
}

function startHostDrag(event, rootHostId) {
  state.draggedHostId = rootHostId;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('text/plain', rootHostId);
  document.querySelectorAll(`.host-row[data-root-host-id="${cssEscape(rootHostId)}"]`).forEach((row) => {
    row.classList.add('dragging-block');
  });
}

function endHostDrag() {
  state.draggedHostId = null;
  document.querySelectorAll('.host-row.dragging-block, .host-row.drop-before, .host-row.drop-after').forEach((row) => {
    row.classList.remove('dragging-block', 'drop-before', 'drop-after');
  });
}

function previewHostDrop(event, row) {
  if (!state.draggedHostId) return;
  const targetHostId = row.dataset.rootHostId;
  if (!targetHostId || targetHostId === state.draggedHostId) return;
  event.preventDefault();
  event.dataTransfer.dropEffect = 'move';
  showHostDropPreview(row, dropAfterRow(event, row));
}

function showHostDropPreview(row, after) {
  document.querySelectorAll('.host-row.drop-before, .host-row.drop-after').forEach((element) => {
    if (element !== row) element.classList.remove('drop-before', 'drop-after');
  });
  row.classList.toggle('drop-before', !after);
  row.classList.toggle('drop-after', after);
}

function clearHostDropPreview(row) {
  row.classList.remove('drop-before', 'drop-after');
}

async function dropHostBlock(event, row) {
  if (!state.draggedHostId) return;
  const targetHostId = row.dataset.rootHostId;
  if (!targetHostId || targetHostId === state.draggedHostId) return;

  event.preventDefault();
  const after = dropAfterRow(event, row);
  const order = movedHostOrder(state.draggedHostId, targetHostId, after);
  endHostDrag();
  if (!order) return;

  await withBusy(async () => {
    await apiReorderHosts(order);
    state.hosts = order.map((id) => state.hosts.find((host) => host.id === id)).filter(Boolean);
    renderHosts(searchInput.value);
    renderSelection();
    writeNotice('Host order updated.');
  });
}

function dropAfterRow(event, row) {
  const rect = row.getBoundingClientRect();
  return event.clientY > rect.top + rect.height / 2;
}

function movedHostOrder(draggedHostId, targetHostId, after) {
  const order = state.hosts.map((host) => host.id);
  const from = order.indexOf(draggedHostId);
  const target = order.indexOf(targetHostId);
  if (from < 0 || target < 0) return null;

  order.splice(from, 1);
  let insertAt = order.indexOf(targetHostId);
  if (insertAt < 0) return null;
  if (after) insertAt += 1;
  order.splice(insertAt, 0, draggedHostId);
  return order;
}

function cssEscape(value) {
  return globalThis.CSS?.escape ? CSS.escape(value) : String(value).replace(/"/g, '\\"');
}

function compactResourceName(name) {
  const clean = String(name ?? '').trim();
  if (!clean) return '?';
  const parts = clean.split(/[-_\s.]+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  }
  return clean.slice(0, 2).toUpperCase();
}

function toggleSidebar() {
  state.sidebarCollapsed = !state.sidebarCollapsed;
  localStorage.setItem('bashes.sidebarCollapsed', String(state.sidebarCollapsed));
  applySidebarState();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function applySidebarState() {
  app.classList.toggle('sidebar-collapsed', state.sidebarCollapsed);
  const toggle = document.querySelector('#toggle-sidebar');
  toggle.textContent = state.sidebarCollapsed ? '>' : '<';
  toggle.title = state.sidebarCollapsed ? 'Expand sidebar' : 'Compact sidebar';
  toggle.setAttribute('aria-label', toggle.title);
  toggle.setAttribute('aria-expanded', String(!state.sidebarCollapsed));
  if (!state.sidebarCollapsed) hideSidebarTooltip();
}

function showSidebarTooltip(anchor) {
  if (!state.sidebarCollapsed) return;
  const tooltip = document.querySelector('#sidebar-tooltip');
  const text = anchor.dataset.tooltip;
  if (!tooltip || !text) return;

  const rect = anchor.getBoundingClientRect();
  tooltip.textContent = text;
  tooltip.style.left = `${Math.round(rect.right + 10)}px`;
  tooltip.style.top = `${Math.round(rect.top + rect.height / 2)}px`;
  tooltip.hidden = false;
}

function hideSidebarTooltip() {
  const tooltip = document.querySelector('#sidebar-tooltip');
  if (tooltip) tooltip.hidden = true;
}

function createPendingTab(resource) {
  clearPendingTabs(resource.id);
  const existing = pendingSessionForResource(resource.id);
  if (existing) {
    focusSession(existing.id);
    return existing.id;
  }

  const sessionID = `pending-${resource.id}`;
  const pane = document.createElement('section');
  pane.className = 'terminal-pane pending-pane';
  pane.dataset.sessionId = sessionID;
  pane.textContent = isLocalResource(resource)
    ? 'Ready to start local shell'
    : `Ready to connect to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  stack.append(pane);

  state.sessions.set(sessionID, {
    id: sessionID,
    resourceId: resource.id,
    title: `${resource.hostname} new`,
    target: resourceTarget(resource),
    element: pane,
    pending: true,
    closed: false,
  });
  setActiveSession(sessionID);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  return sessionID;
}

function clearPendingTabs(exceptResourceId = '') {
  for (const session of [...state.sessions.values()]) {
    if (!session.pending || session.resourceId === exceptResourceId) continue;
    if (session.element) session.element.remove();
    state.sessions.delete(session.id);
    forgetSessionFocus(session.id);
    if (state.activeSessionId === session.id) setActiveSession(lastFocusedSessionID());
  }
}

function selectResource(resource) {
  state.selectedId = resource.id;
  const session = preferredSessionForResource(resource.id);
  if (session) {
    if (!session.pending) clearPendingTabs(resource.id);
    focusSession(session.id);
  } else {
    createPendingTab(resource);
  }
}

function createSession(sessionID, resource, kind = 'ssh') {
  const pending = pendingSessionForResource(resource.id);
  if (pending?.pending) {
    pending.element.remove();
    state.sessions.delete(pending.id);
  }
  const ordinal = realSessionsForResource(resource.id).length + 1;

  const pane = document.createElement('section');
  pane.className = 'terminal-pane';
  pane.dataset.sessionId = sessionID;
  stack.append(pane);

  const terminal = new Terminal({
    cursorBlink: true,
    convertEol: true,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    fontSize: state.terminalFontSize,
    theme: {
      background: '#101418',
      foreground: '#d7dde5',
      cursor: '#f5c542',
      selectionBackground: '#3d4a58',
    },
  });
  const fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);
  terminal.open(pane);
  installTerminalKeyRepeatFallback(terminal, sessionID);
  terminal.onData((data) => {
    if (state.sessions.get(sessionID)?.closed) return;
    apiWriteSSHSession(sessionID, data).catch((error) => {
      writeNotice(`${terminalKindLabel(kind)} input error: ${error?.message ?? error}`);
    });
  });
  terminal.onSelectionChange(() => {
    const selected = terminal.getSelection();
    if (!selected) return;
    writeClipboard(selected).catch(() => {});
  });
  pane.addEventListener('contextmenu', (event) => {
    event.preventDefault();
    event.stopPropagation();
    readClipboard()
      .then((text) => {
        if (state.sessions.get(sessionID)?.closed) return undefined;
        if (text) return apiWriteSSHSession(sessionID, text);
        return undefined;
      })
      .catch(() => {});
  });

  state.sessions.set(sessionID, {
    id: sessionID,
    resourceId: resource.id,
    title: sessionTitle(resource.hostname, ordinal),
    ordinal,
    target: resourceTarget(resource),
    kind,
    terminal,
    fitAddon,
    element: pane,
    closed: false,
  });
  flushPendingSSHOutput(sessionID);
  setActiveSession(sessionID);
  writeNotice(isLocalResource(resource) ? 'Local shell started.' : `Connected to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function installTerminalKeyRepeatFallback(terminal, sessionID) {
  terminal.attachCustomKeyEventHandler((event) => {
    if (event.type !== 'keydown' || !event.repeat || event.isComposing) {
      return true;
    }

    const data = repeatedKeyData(event, terminal);
    if (!data) return true;
    if (state.sessions.get(sessionID)?.closed) {
      event.preventDefault();
      return false;
    }

    event.preventDefault();
    apiWriteSSHSession(sessionID, data).catch((error) => {
      const session = state.sessions.get(sessionID);
      writeNotice(`${terminalKindLabel(session?.kind)} input error: ${error?.message ?? error}`);
    });
    return false;
  });
}

function repeatedKeyData(event, terminal) {
  if (event.key.length === 1) {
    if (event.ctrlKey || event.metaKey || event.altKey) return '';
    return event.key;
  }

  const applicationCursor = Boolean(terminal.modes?.applicationCursorKeysMode);
  switch (event.key) {
    case 'ArrowUp':
      return applicationCursor ? '\x1bOA' : '\x1b[A';
    case 'ArrowDown':
      return applicationCursor ? '\x1bOB' : '\x1b[B';
    case 'ArrowRight':
      return applicationCursor ? '\x1bOC' : '\x1b[C';
    case 'ArrowLeft':
      return applicationCursor ? '\x1bOD' : '\x1b[D';
    case 'Backspace':
      return '\x7f';
    case 'Delete':
      return '\x1b[3~';
    case 'Home':
      return applicationCursor ? '\x1bOH' : '\x1b[H';
    case 'End':
      return applicationCursor ? '\x1bOF' : '\x1b[F';
    case 'PageUp':
      return '\x1b[5~';
    case 'PageDown':
      return '\x1b[6~';
    case 'Tab':
      return event.shiftKey ? '\x1b[Z' : '\t';
    case 'Enter':
      return '\r';
    case 'Escape':
      return '\x1b';
    default:
      return '';
  }
}

function renderTabs() {
  const tabs = document.querySelector('#session-tabs');
  const empty = document.querySelector('#empty-terminal');
  const sessions = [...state.sessions.values()];
  tabs.hidden = sessions.length === 0;
  empty.hidden = Boolean(state.activeSessionId);

  tabs.replaceChildren(...sessions.map((session) => {
    const tab = document.createElement('button');
    tab.type = 'button';
    tab.className = `session-tab ${session.id === state.activeSessionId ? 'active' : ''} ${session.closed ? 'closed' : ''} ${session.pending ? 'pending' : ''}`;
    tab.draggable = true;
    tab.innerHTML = '<span></span><strong></strong>';
    tab.querySelector('span').textContent = session.closed ? 'closed' : session.pending ? 'new' : terminalKindLabel(session.kind);
    tab.querySelector('strong').textContent = session.title;
    tab.title = session.closed ? 'Double-click to reconnect' : session.target;
    tab.addEventListener('click', () => {
      if (!session.pending) clearPendingTabs(session.resourceId);
      setActiveSession(session.id);
      renderTabs();
      renderSelection();
      scheduleTerminalFit();
      focusActiveTerminal();
    });
    tab.addEventListener('dblclick', () => {
      if (session.closed) reconnectClosedSession(session.id);
    });
    tab.addEventListener('dragstart', (event) => startSessionTabDrag(event, session.id));
    tab.addEventListener('dragend', () => endSessionTabDrag());

    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'tab-close';
    close.textContent = 'X';
    close.title = `Close ${session.title}`;
    close.addEventListener('click', (event) => {
      event.stopPropagation();
      stopSession(session.id);
    });

    const wrapper = document.createElement('div');
    wrapper.className = 'session-tab-wrap';
    wrapper.dataset.sessionId = session.id;
    wrapper.addEventListener('dragover', (event) => previewSessionTabDrop(event, wrapper));
    wrapper.addEventListener('dragleave', () => clearSessionTabDropPreview(wrapper));
    wrapper.addEventListener('drop', (event) => dropSessionTab(event, wrapper));
    wrapper.append(tab, close);
    return wrapper;
  }));

  document.querySelectorAll('.terminal-pane').forEach((pane) => {
    pane.hidden = pane.dataset.sessionId !== state.activeSessionId;
  });
}

function startSessionTabDrag(event, sessionId) {
  state.draggedSessionId = sessionId;
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('text/plain', sessionId);
  event.currentTarget.closest('.session-tab-wrap')?.classList.add('dragging-tab');
}

function endSessionTabDrag() {
  state.draggedSessionId = null;
  document.querySelectorAll('.session-tab-wrap.dragging-tab, .session-tab-wrap.drop-before, .session-tab-wrap.drop-after').forEach((tab) => {
    tab.classList.remove('dragging-tab', 'drop-before', 'drop-after');
  });
}

function previewSessionTabDrop(event, wrapper) {
  if (!state.draggedSessionId) return;
  const targetSessionId = wrapper.dataset.sessionId;
  if (!targetSessionId || targetSessionId === state.draggedSessionId) return;
  event.preventDefault();
  const after = dropAfterTab(event, wrapper);
  wrapper.classList.toggle('drop-before', !after);
  wrapper.classList.toggle('drop-after', after);
}

function clearSessionTabDropPreview(wrapper) {
  wrapper.classList.remove('drop-before', 'drop-after');
}

function dropSessionTab(event, wrapper) {
  if (!state.draggedSessionId) return;
  const targetSessionId = wrapper.dataset.sessionId;
  if (!targetSessionId || targetSessionId === state.draggedSessionId) return;
  event.preventDefault();
  reorderSessionTabs(state.draggedSessionId, targetSessionId, dropAfterTab(event, wrapper));
  endSessionTabDrag();
  renderTabs();
  focusActiveTerminal();
}

function dropAfterTab(event, wrapper) {
  const rect = wrapper.getBoundingClientRect();
  return event.clientX > rect.left + rect.width / 2;
}

function reorderSessionTabs(draggedSessionId, targetSessionId, after) {
  const order = [...state.sessions.keys()];
  const from = order.indexOf(draggedSessionId);
  if (from < 0 || !order.includes(targetSessionId)) return;
  order.splice(from, 1);
  let insertAt = order.indexOf(targetSessionId);
  if (after) insertAt += 1;
  order.splice(insertAt, 0, draggedSessionId);
  const reordered = new Map();
  for (const id of order) {
    const session = state.sessions.get(id);
    if (session) reordered.set(id, session);
  }
  state.sessions = reordered;
}

function openResourcePanel(mode, hostID = '') {
  const panel = document.querySelector('#resource-panel');
  const form = document.querySelector('#resource-form');
  const typeField = document.querySelector('#subsystem-type-field');
  const parentSummary = document.querySelector('#parent-host-summary');
  const title = document.querySelector('#resource-panel-title');
  const kicker = document.querySelector('#resource-panel-kicker');
  const submit = document.querySelector('#resource-submit');

  state.drawerMode = mode;
  state.drawerHostId = hostID;
  state.editResourceId = null;
  form.reset();
  form.elements.port.value = '22';

  const subsystemMode = mode === 'subsystem';
  typeField.hidden = !subsystemMode;
  parentSummary.hidden = !subsystemMode;
  if (subsystemMode) {
    kicker.textContent = 'Subsystem';
    title.textContent = 'Add Subsystem';
    submit.textContent = 'Add Subsystem';
    const parent = findResource(hostID)?.resource;
    parentSummary.textContent = parent ? `Parent: ${parent.hostname} (${parent.user}@${parent.ip || parent.hostname})` : '';
  } else {
    kicker.textContent = 'Host';
    title.textContent = 'Add Host';
    submit.textContent = 'Add Host';
    parentSummary.textContent = '';
  }

  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  form.elements.hostname.focus();
}

function openEditPanel() {
  const selected = findResource(state.selectedId);
  if (!selected) return;

  const panel = document.querySelector('#resource-panel');
  const form = document.querySelector('#resource-form');
  const typeField = document.querySelector('#subsystem-type-field');
  const parentSummary = document.querySelector('#parent-host-summary');
  const title = document.querySelector('#resource-panel-title');
  const kicker = document.querySelector('#resource-panel-kicker');
  const submit = document.querySelector('#resource-submit');
  const resource = selected.resource;
  const subsystemMode = selected.type !== 'host';

  state.drawerMode = 'edit';
  state.drawerHostId = selected.parent?.id ?? '';
  state.editResourceId = resource.id;

  form.reset();
  form.elements.hostname.value = resource.hostname;
  form.elements.ip.value = resource.ip;
  form.elements.port.value = String(resource.port);
  form.elements.user.value = resource.user;
  typeField.hidden = !subsystemMode;
  parentSummary.hidden = !subsystemMode;
  if (subsystemMode) {
    form.elements.type.value = resource.type;
    kicker.textContent = 'Subsystem';
    title.textContent = 'Edit Subsystem';
    submit.textContent = 'Save Changes';
    parentSummary.textContent = selected.parent
      ? `Parent: ${selected.parent.hostname} (${selected.parent.user}@${selected.parent.ip || selected.parent.hostname})`
      : '';
  } else {
    kicker.textContent = 'Host';
    title.textContent = 'Edit Host';
    submit.textContent = 'Save Changes';
    parentSummary.textContent = '';
  }

  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  form.elements.hostname.focus();
}

function closeResourcePanel() {
  const panel = document.querySelector('#resource-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  state.drawerMode = null;
  state.drawerHostId = null;
  state.editResourceId = null;
  restoreTerminalFocusAfterOverlay();
}

async function openConnectPanel(statusMessage = '', statusKind = '') {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) {
    await quickConnect(selected);
    return;
  }

  await loadKeys();
  const panel = document.querySelector('#connect-panel');
  const form = document.querySelector('#connect-form');
  form.reset();
  form.elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
  renderKeyOptions(form.elements.keyName, true);
  applyConnectDefaults(form, selected);
  const realSessionCount = realSessionsForResource(selected.id).length;
  document.querySelector('#connect-summary').textContent =
    `${realSessionCount > 0 ? 'New session: ' : ''}${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
  form.querySelector('button[type="submit"]').textContent = 'Connect';
  setConnectStatus(statusMessage, statusKind);
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  focusConnectPasswordInput(form);
}

function focusConnectPasswordInput(form) {
  window.setTimeout(() => {
    requestAnimationFrame(() => {
      form.elements.password.focus();
      form.elements.password.select();
    });
  }, 0);
}

function closeConnectPanel() {
  const panel = document.querySelector('#connect-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  setConnectStatus('', '');
  restoreTerminalFocusAfterOverlay();
}

async function openTunnelPanel() {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  await loadKeys();
  await loadTunnels();
  const panel = document.querySelector('#tunnel-panel');
  const form = document.querySelector('#tunnel-form');
  form.reset();
  form.elements.type.value = 'socks';
  form.elements.localHost.value = '127.0.0.1';
  form.elements.localPort.value = '1080';
  form.elements.remoteHost.value = '127.0.0.1';
  form.elements.remotePort.value = '80';
  form.elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
  renderKeyOptions(form.elements.keyName, true);
  applyConnectDefaults(form, selected);
  updateTunnelMode();
  renderTunnelStatus();
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  form.elements.localPort.focus();
}

function closeTunnelPanel() {
  const panel = document.querySelector('#tunnel-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  restoreTerminalFocusAfterOverlay();
}

function updateTunnelSummary() {
  const selected = findResource(state.selectedId)?.resource;
  const form = document.querySelector('#tunnel-form');
  const summary = document.querySelector('#tunnel-summary');
  if (!selected || !form || !summary) return;

  const type = form.elements.type.value || 'socks';
  const bind = form.elements.localHost.value.trim() || '127.0.0.1';
  const port = form.elements.localPort.value || '1080';
  const targetHost = form.elements.remoteHost.value.trim() || '127.0.0.1';
  const targetPort = form.elements.remotePort.value || (type === 'remote' ? '3000' : '80');
  const login = `${selected.user}@${selected.ip || selected.hostname}`;

  if (type === 'local') {
    summary.textContent = `ssh -L ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
    return;
  }
  if (type === 'remote') {
    summary.textContent = `ssh -R ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
    return;
  }
  summary.textContent = `ssh -D ${bind}:${port} ${login}`;
}

function updateTunnelMode() {
  const form = document.querySelector('#tunnel-form');
  if (!form) return;

  const type = form.elements.type.value || 'socks';
  const title = document.querySelector('#tunnel-title');
  const bindLabel = document.querySelector('#tunnel-bind-label');
  const portLabel = document.querySelector('#tunnel-port-label');
  const targetFields = document.querySelector('#tunnel-target-fields');
  const targetHostLabel = document.querySelector('#tunnel-target-host-label');
  const targetPortLabel = document.querySelector('#tunnel-target-port-label');

  targetFields.hidden = type === 'socks';
  if (type === 'local') {
    title.textContent = 'Local Forward';
    bindLabel.textContent = 'Local Bind';
    portLabel.textContent = 'Local Port';
    targetHostLabel.textContent = 'Remote Host';
    targetPortLabel.textContent = 'Remote Port';
    if (!form.elements.localPort.value || form.elements.localPort.value === '1080') form.elements.localPort.value = '8080';
    if (!form.elements.remotePort.value) form.elements.remotePort.value = '80';
  } else if (type === 'remote') {
    title.textContent = 'Remote Forward';
    bindLabel.textContent = 'Remote Bind';
    portLabel.textContent = 'Remote Port';
    targetHostLabel.textContent = 'Local Host';
    targetPortLabel.textContent = 'Local Port';
    if (!form.elements.localPort.value || form.elements.localPort.value === '1080') form.elements.localPort.value = '8080';
    if (!form.elements.remotePort.value) form.elements.remotePort.value = '3000';
  } else {
    title.textContent = 'SOCKS Proxy';
    bindLabel.textContent = 'Bind';
    portLabel.textContent = 'Port';
    if (!form.elements.localPort.value) form.elements.localPort.value = '1080';
  }
  updateTunnelSummary();
}

async function openKeysPanel() {
  const selected = findResource(state.selectedId)?.resource;
  if (isLocalResource(selected)) return;

  await loadKeySettings();
  await loadKeys();
  setKeyDirectoryStatus('', '');
  setKeyInstallStatus('', '');
  renderKeyInstallSummary();
  document.querySelector('#key-install-form').elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
  const panel = document.querySelector('#keys-panel');
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
}

function closeKeysPanel() {
  const panel = document.querySelector('#keys-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  restoreTerminalFocusAfterOverlay();
}

async function openFileTransferModal() {
  if (!FILE_TRANSFER_ENABLED) return;
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const panel = document.querySelector('#file-transfer-modal');
  const title = document.querySelector('#file-transfer-title');
  const root = document.querySelector('#file-transfer-root');
  title.textContent = `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));

  let workspace = state.fileTransferWorkspaces.get(selected.id);
  if (!workspace) {
    const { mountFileTransfer } = await import('./file-transfer/mount.js');
    const element = document.createElement('div');
    element.className = 'file-transfer-workspace';
    element.dataset.resourceId = selected.id;
    element.addEventListener('bashes-file-transfer-active', (event) => {
      workspace.active = Boolean(event.detail?.active);
    });
    root.append(element);
    workspace = {
      active: false,
      element,
      resource: selected,
      unmount: mountFileTransfer(element, { resource: selected }),
    };
    state.fileTransferWorkspaces.set(selected.id, workspace);
  }

  showFileTransferWorkspace(selected.id);
}

function closeFileTransferModal() {
  const workspace = state.fileTransferWorkspaces.get(state.activeFileTransferResourceId);
  if (workspace?.active) {
    backgroundFileTransferModal();
    writeNotice('File transfer continues in background.');
    return;
  }
  if (workspace) destroyFileTransferWorkspace(state.activeFileTransferResourceId);
  hideFileTransferModal();
}

function backgroundFileTransferModal() {
  hideFileTransferModal();
}

function hideFileTransferModal() {
  const panel = document.querySelector('#file-transfer-modal');
  panel.classList.remove('open');
  panel.hidden = true;
  restoreTerminalFocusAfterOverlay();
}

function showFileTransferWorkspace(resourceId) {
  for (const [id, workspace] of state.fileTransferWorkspaces) {
    workspace.element.hidden = id !== resourceId;
  }
  state.activeFileTransferResourceId = resourceId;
}

function destroyFileTransferWorkspace(resourceId) {
  if (!resourceId) return;
  const workspace = state.fileTransferWorkspaces.get(resourceId);
  if (!workspace) return;
  workspace.unmount();
  workspace.element.remove();
  state.fileTransferWorkspaces.delete(resourceId);
  if (state.activeFileTransferResourceId === resourceId) {
    state.activeFileTransferResourceId = null;
  }
}

function closeAllFileTransferWorkspaces() {
  for (const resourceId of [...state.fileTransferWorkspaces.keys()]) {
    destroyFileTransferWorkspace(resourceId);
  }
  hideFileTransferModal();
}

function registerFileTransferModalResize() {
  const card = document.querySelector('.file-transfer-card');
  const handle = document.querySelector('#file-transfer-resize-handle');
  if (!card || !handle) return;

  let resizeState = null;
  handle.addEventListener('pointerdown', (event) => {
    event.preventDefault();
    resizeState = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      width: card.offsetWidth,
      height: card.offsetHeight,
    };
    handle.setPointerCapture(event.pointerId);
    document.body.classList.add('resizing-file-transfer');
  });

  handle.addEventListener('pointermove', (event) => {
    if (!resizeState || resizeState.pointerId !== event.pointerId) return;
    resizeFileTransferModal(
      resizeState.width + event.clientX - resizeState.startX,
      resizeState.height + event.clientY - resizeState.startY,
    );
  });

  const stopResize = (event) => {
    if (!resizeState || resizeState.pointerId !== event.pointerId) return;
    resizeState = null;
    document.body.classList.remove('resizing-file-transfer');
  };
  handle.addEventListener('pointerup', stopResize);
  handle.addEventListener('pointercancel', stopResize);
}

function registerFileTransferModalDrag() {
  const card = document.querySelector('.file-transfer-card');
  const header = document.querySelector('.file-transfer-header');
  if (!card || !header) return;

  let dragState = null;
  header.addEventListener('pointerdown', (event) => {
    if (event.target.closest('button, input, select, textarea, a')) return;
    event.preventDefault();
    const rect = card.getBoundingClientRect();
    dragState = {
      pointerId: event.pointerId,
      offsetX: event.clientX - rect.left,
      offsetY: event.clientY - rect.top,
      width: rect.width,
      height: rect.height,
    };
    card.style.position = 'fixed';
    card.style.left = `${rect.left}px`;
    card.style.top = `${rect.top}px`;
    card.style.margin = '0';
    header.setPointerCapture(event.pointerId);
    document.body.classList.add('dragging-file-transfer');
  });

  header.addEventListener('pointermove', (event) => {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    const maxLeft = Math.max(18, window.innerWidth - dragState.width - 18);
    const maxTop = Math.max(18, window.innerHeight - dragState.height - 18);
    card.style.left = `${clamp(event.clientX - dragState.offsetX, 18, maxLeft)}px`;
    card.style.top = `${clamp(event.clientY - dragState.offsetY, 18, maxTop)}px`;
  });

  const stopDrag = (event) => {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    dragState = null;
    document.body.classList.remove('dragging-file-transfer');
  };
  header.addEventListener('pointerup', stopDrag);
  header.addEventListener('pointercancel', stopDrag);
}

function resizeFileTransferModal(width, height) {
  const card = document.querySelector('.file-transfer-card');
  if (!card) return;
  const minWidth = Math.min(720, window.innerWidth - 36);
  const minHeight = Math.min(460, window.innerHeight - 36);
  const maxWidth = window.innerWidth - 36;
  const maxHeight = window.innerHeight - 36;
  card.style.width = `${clamp(width, minWidth, maxWidth)}px`;
  card.style.height = `${clamp(height, minHeight, maxHeight)}px`;
}

function clampFileTransferModalSize() {
  const panel = document.querySelector('#file-transfer-modal');
  const card = document.querySelector('.file-transfer-card');
  if (!panel || panel.hidden || !card || !card.style.width) return;
  resizeFileTransferModal(card.offsetWidth, card.offsetHeight);
}

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

function renderSelection() {
  document.querySelectorAll('.host-row').forEach((row) => {
    row.classList.toggle('selected', row.dataset.id === state.selectedId);
  });

  const selected = findResource(state.selectedId);
  const activeSession = state.sessions.get(state.activeSessionId);
  const title = document.querySelector('#session-title');
  const edit = document.querySelector('#edit-resource');
  const addSubsystem = document.querySelector('#header-add-subsystem');
  const keys = document.querySelector('#open-keys-panel');
  const fileTransfer = document.querySelector('#open-file-transfer');
  const tunnel = document.querySelector('#open-tunnel-panel');
  const connect = document.querySelector('#connect');
  const disconnect = document.querySelector('#disconnect');
  const remove = document.querySelector('#delete-resource');

  if (activeSession) {
    title.textContent = activeSession.target;
  } else if (selected) {
    title.textContent = isLocalResource(selected.resource) ? 'localhost' : `${selected.resource.user}@${selected.resource.hostname}`;
  } else {
    title.textContent = 'No session selected';
  }

  if (!selected) {
    edit.disabled = true;
    addSubsystem.disabled = true;
    addSubsystem.hidden = false;
    keys.disabled = true;
    fileTransfer.disabled = true;
    fileTransfer.hidden = !FILE_TRANSFER_ENABLED;
    tunnel.disabled = true;
    connect.disabled = true;
    disconnect.disabled = true;
    remove.disabled = true;
    return;
  }

  const localSelected = isLocalResource(selected.resource);
  addSubsystem.hidden = false;
  edit.disabled = state.busy || !selected || localSelected;
  addSubsystem.disabled = state.busy || !selected || localSelected;
  keys.disabled = state.busy || !selected || localSelected;
  fileTransfer.hidden = !FILE_TRANSFER_ENABLED;
  fileTransfer.disabled = state.busy || !selected || localSelected || !FILE_TRANSFER_ENABLED;
  tunnel.disabled = state.busy || !selected || localSelected;
  connect.textContent = realSessionsForResource(selected.resource.id).length > 0 ? 'New Session' : 'Connect';
  connect.disabled = state.busy || !selected;
  disconnect.disabled = state.busy || !activeSession || activeSession.closed;
  remove.disabled = state.busy || !selected || localSelected;
  renderKeyInstallSummary();
  renderTunnelStatus();
}

function renderKeyOptions(select = document.querySelector('#key-select'), includeEmpty = false) {
  if (!select) return;
  const options = [];
  if (includeEmpty) {
    const empty = document.createElement('option');
    empty.value = '';
    empty.textContent = 'Use password/agent or custom path';
    options.push(empty);
  }

  const bashesGroup = keyOptionGroup('Bashes keys', state.keys.filter((key) => keySource(key) === 'bashes'));
  const systemGroup = keyOptionGroup('System keys (~/.ssh)', state.keys.filter((key) => keySource(key) === 'system'));
  const customGroup = keyOptionGroup('Custom keys directory', state.keys.filter((key) => keySource(key) === 'custom'));
  if (bashesGroup) options.push(bashesGroup);
  if (systemGroup) options.push(systemGroup);
  if (customGroup) options.push(customGroup);

  select.replaceChildren(...options);
  renderSelectedPublicKey();
}

function keyOptionGroup(label, keys) {
  if (keys.length === 0) return null;
  const group = document.createElement('optgroup');
  group.label = label;
  for (const key of keys) {
    const option = document.createElement('option');
    option.value = keyChoiceValue(key);
    option.textContent = keyOptionLabel(key);
    option.title = key.privateKey || key.name;
    group.append(option);
  }
  return group;
}

function keySource(key) {
  return key?.source || 'bashes';
}

function keyChoiceValue(key) {
  if (keySource(key) === 'system' || keySource(key) === 'custom') return `path:${key.privateKey || ''}`;
  return `bashes:${key.name || ''}`;
}

function keyOptionLabel(key) {
  if (keySource(key) === 'system') return `${key.name} - ${key.privateKey}`;
  if (keySource(key) === 'custom') return `${key.name} - ${key.privateKey}`;
  return key.name;
}

function registerAuthChoiceSync(form) {
  if (!form?.elements?.keyName || !form?.elements?.privateKeyPath) return;

  form.elements.privateKeyPath.addEventListener('input', () => {
    if (form.elements.privateKeyPath.value.trim()) {
      form.elements.keyName.value = '';
    }
  });

  form.elements.keyName.addEventListener('change', () => {
    if (form.elements.keyName.value) {
      form.elements.privateKeyPath.value = '';
    }
  });
}

function authInputFromForm(form) {
  const privateKeyPath = form.elements.privateKeyPath.value.trim();
  if (privateKeyPath) {
    return {
      keyName: '',
      privateKeyPath,
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }

  const keyChoice = selectedKeyChoice(form.elements.keyName);
  if (keyChoice?.source === 'system') {
    return {
      keyName: '',
      privateKeyPath: keyChoice.privateKey,
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }
  if (keyChoice?.source === 'custom') {
    return {
      keyName: '',
      privateKeyPath: keyChoice.privateKey,
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }
  if (keyChoice?.source === 'bashes') {
    return {
      keyName: keyChoice.name,
      privateKeyPath: '',
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }

  return {
    keyName: '',
    privateKeyPath: '',
    privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
  };
}

function applyConnectDefaults(form, resource) {
  const auth = resource?.auth;
  if (!auth) return;

  if (auth.trustHostKey) {
    form.elements.trustHostKey.checked = true;
  }
  if (auth.method === 'key' && auth.keyName) {
    const value = `bashes:${auth.keyName}`;
    const hasKey = [...form.elements.keyName.options].some((option) => option.value === value);
    if (hasKey) {
      form.elements.keyName.value = value;
      form.elements.privateKeyPath.value = '';
    }
  }
  if (auth.method === 'path' && auth.privateKeyPath) {
    const value = `path:${auth.privateKeyPath}`;
    const hasPath = [...form.elements.keyName.options].some((option) => option.value === value);
    if (hasPath) {
      form.elements.keyName.value = value;
      form.elements.privateKeyPath.value = '';
    } else {
      form.elements.keyName.value = '';
      form.elements.privateKeyPath.value = auth.privateKeyPath;
    }
  }
}

function selectedKeyChoice(select) {
  if (!select?.value) return null;
  const key = state.keys.find((item) => keyChoiceValue(item) === select.value);
  if (key) return key;
  if (select.value.startsWith('path:')) {
    const privateKey = select.value.slice('path:'.length);
    return { source: 'path', name: privateKey.split(/[\\/]/).pop(), privateKey, publicKey: `${privateKey}.pub` };
  }
  if (select.value.startsWith('bashes:')) {
    const name = select.value.slice('bashes:'.length);
    return { source: 'bashes', name };
  }
  return null;
}

function installInputFromKeyChoice(keyChoice) {
  if (keyChoice.source === 'system' || keyChoice.source === 'custom' || keyChoice.source === 'path') {
    return {
      keyName: '',
      privateKeyPath: keyChoice.privateKey,
      publicKeyPath: keyChoice.publicKey || keyChoice.privateKey,
    };
  }
  return { keyName: keyChoice.name };
}

function keyChoiceLabel(keyChoice) {
  if (!keyChoice) return '';
  return keyChoice.source === 'system' || keyChoice.source === 'custom' || keyChoice.source === 'path'
    ? keyChoice.privateKey
    : keyChoice.name;
}

function authInputFromPreference(resource) {
  const auth = resource?.auth;
  if (!auth) return {};

  if (auth.method === 'key' && auth.keyName) {
    return { keyName: auth.keyName };
  }
  if (auth.method === 'path' && auth.privateKeyPath) {
    return { privateKeyPath: auth.privateKeyPath };
  }
  return {};
}

function trustHostKeyFromPreference(resource) {
  return Boolean(resource?.auth?.trustHostKey);
}

async function renderSelectedPublicKey() {
  const select = document.querySelector('#key-select');
  const output = document.querySelector('#public-key');
  if (!select || !output) return;
  const keyChoice = selectedKeyChoice(select);
  if (!keyChoice) {
    output.value = '';
    return;
  }
  try {
    output.value = keyChoice.source === 'system' || keyChoice.source === 'custom' || keyChoice.source === 'path'
      ? await apiReadSSHPublicKeyPath(keyChoice.publicKey || keyChoice.privateKey)
      : await apiReadSSHPublicKey(keyChoice.name);
    setKeyInstallStatus('', '');
  } catch (error) {
    output.value = '';
    const message = `Could not read SSH key ${keyChoiceLabel(keyChoice)}: ${error?.message ?? error}`;
    setKeyInstallStatus(message, 'error');
    writeNotice(message);
  }
}

function renderKeyInstallSummary() {
  const summary = document.querySelector('#key-install-summary');
  if (!summary) return;
  const selected = findResource(state.selectedId)?.resource;
  summary.textContent = selected && !isLocalResource(selected)
    ? `Install selected key on ${selected.user}@${selected.ip || selected.hostname}:${selected.port}`
    : 'Select a host or subsystem to install the key.';
}

function renderKeySettings() {
  const form = document.querySelector('#key-directory-form');
  if (!form) return;
  form.elements.directory.value = state.keySettings?.customDirectory || '';
}

function setKeyDirectoryStatus(message, kind) {
  const status = document.querySelector('#key-directory-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function setKeyInstallStatus(message, kind) {
  const status = document.querySelector('#key-install-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function setConnectStatus(message, kind) {
  const status = document.querySelector('#connect-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

async function startSSHSessionWithHostKeyPrompt(input, resource, confirmLabel = 'Trust and continue') {
  try {
    return await apiStartSSHSession(input);
  } catch (error) {
    if (input.acceptHostKey) throw error;
    if (!(await confirmUnknownHostKey(error, resource, confirmLabel))) throw error;
    return await apiStartSSHSession({ ...input, acceptHostKey: true });
  }
}

async function startSSHTunnelWithHostKeyPrompt(input, resource) {
  try {
    return await apiStartSSHTunnel(input);
  } catch (error) {
    if (input.acceptHostKey) throw error;
    if (!(await confirmUnknownHostKey(error, resource, 'Trust and start tunnel'))) throw error;
    return await apiStartSSHTunnel({ ...input, acceptHostKey: true });
  }
}

async function installSSHKeyWithHostKeyPrompt(input, resource) {
  try {
    return await apiInstallSSHKey(input);
  } catch (error) {
    if (input.acceptHostKey) throw error;
    if (!(await confirmUnknownHostKey(error, resource, 'Trust and install key'))) throw error;
    return await apiInstallSSHKey({ ...input, acceptHostKey: true });
  }
}

async function confirmUnknownHostKey(error, resource, confirmLabel) {
  const hostKey = parseUnknownHostKeyError(error);
  if (!hostKey) return false;
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  return await openConfirmModal({
    kicker: 'SSH Host Key',
    title: `Trust ${resource.hostname}?`,
    message: `Bashes has not seen this SSH server key before.\n\nTarget: ${target}\nServer: ${hostKey.host}\nFingerprint: ${hostKey.fingerprint}\n\nContinue only if this fingerprint matches the expected server.`,
    confirmLabel,
  });
}

function parseUnknownHostKeyError(error) {
  const detail = String(error?.message ?? error ?? '');
  const match = detail.match(/BASHES_HOST_KEY_UNKNOWN\s+resource=\S+\s+host=(\S+)\s+fingerprint=(SHA256:\S+)/);
  if (!match) return null;
  return { host: match[1], fingerprint: match[2] };
}

function connectErrorMessage(error, resource) {
  const detail = String(error?.message ?? error ?? '').trim();
  if (isLocalResource(resource)) {
    return `Could not start local shell: ${detail || 'unknown error'}`;
  }
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  const unknownHostKey = parseUnknownHostKeyError(error);
  if (unknownHostKey) {
    return `Could not verify ${target}: SSH host key was not trusted.`;
  }
  const mismatch = parseHostKeyMismatchError(error);
  if (mismatch) {
    return `Could not verify ${target}: SSH host key changed. Expected ${mismatch.expected}, got ${mismatch.actual}.`;
  }
  if (isAuthError(error)) {
    return `Could not authenticate to ${target}. Enter the remote password or configure a valid SSH key.`;
  }
  if (/host key|knownhosts|known host/i.test(detail)) {
    return `Could not verify ${target}. Use "Skip host key verification" only if you accept the security risk.`;
  }
  if (/timeout|deadline|i\/o timeout|operation timed out|context canceled|context deadline exceeded/i.test(detail)) {
    return `Could not connect to ${target}: connection timed out.`;
  }
  if (/no route to host|network is unreachable|host is down/i.test(detail)) {
    return `Could not connect to ${target}: host is unreachable.`;
  }
  if (/connection refused/i.test(detail)) {
    return `Could not connect to ${target}: connection refused.`;
  }
  return `Could not connect to ${target}: ${detail || 'unknown error'}`;
}

function parseHostKeyMismatchError(error) {
  const detail = String(error?.message ?? error ?? '');
  const match = detail.match(/BASHES_HOST_KEY_MISMATCH\s+resource=\S+\s+host=\S+\s+expected=(SHA256:\S+)\s+actual=(SHA256:\S+)/);
  if (!match) return null;
  return { expected: match[1], actual: match[2] };
}

function isAuthError(error) {
  const detail = String(error?.message ?? error ?? '');
  return /unable to authenticate|no supported methods|no SSH authentication method|handshake failed|permission denied/i.test(detail);
}

function keyInstallErrorMessage(error, resource) {
  const detail = String(error?.message ?? error ?? '').trim();
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  const unknownHostKey = parseUnknownHostKeyError(error);
  if (unknownHostKey) {
    return `Could not verify ${target}: SSH host key was not trusted.`;
  }
  const mismatch = parseHostKeyMismatchError(error);
  if (mismatch) {
    return `Could not verify ${target}: SSH host key changed. Expected ${mismatch.expected}, got ${mismatch.actual}.`;
  }
  if (isAuthError(error)) {
    return `Could not connect to ${target} to install the SSH key. Enter the remote password or configure a valid existing key.`;
  }
  if (/host key|knownhosts|known host/i.test(detail)) {
    return `Could not verify ${target}. Use "Skip host key verification" only if you accept the security risk.`;
  }
  if (/timeout|deadline|i\/o timeout/i.test(detail)) {
    return `Could not reach ${target} to install the SSH key: connection timed out.`;
  }
  if (/connection refused/i.test(detail)) {
    return `Could not reach ${target} to install the SSH key: connection refused.`;
  }
  return `Could not install the SSH key on ${target}: ${detail || 'unknown error'}`;
}

function renderTunnelStatus() {
  const status = document.querySelector('#tunnel-status');
  const stop = document.querySelector('#stop-tunnel');
  const start = document.querySelector('#start-tunnel');
  if (!status || !stop || !start) return;

  const selected = findResource(state.selectedId)?.resource;
  const tunnel = selected && !isLocalResource(selected) ? tunnelForResource(selected.id) : null;
  status.hidden = !tunnel;
  status.textContent = tunnel
    ? `${tunnelLabel(tunnel.type)} active on ${tunnel.localAddress} -> ${tunnel.forwardTarget || tunnel.target}`
    : '';
  stop.disabled = state.busy || !tunnel;
  start.disabled = state.busy || Boolean(tunnel);
}

function tunnelLabel(type) {
  if (type === 'local') return 'Local tunnel';
  if (type === 'remote') return 'Remote tunnel';
  return 'SOCKS tunnel';
}

function findResource(id) {
  if (state.localShellSupported && id === LOCAL_RESOURCE_ID) {
    return { type: 'local', resource: localResource(), parent: null };
  }
  for (const host of state.hosts) {
    if (host.id === id) return { type: 'host', resource: host, parent: null };
    const found = findNestedResource(host.subsystems ?? [], id, host);
    if (found) return found;
  }
  return null;
}

function findNestedResource(subsystems, id, parent) {
  for (const subsystem of subsystems ?? []) {
    if (subsystem.id === id) return { type: subsystem.type, resource: subsystem, parent };
    const found = findNestedResource(subsystem.subsystems ?? [], id, subsystem);
    if (found) {
      return found;
    }
  }
  return null;
}

function sessionsForResource(resourceId) {
  return [...state.sessions.values()].filter((session) => session.resourceId === resourceId);
}

function realSessionsForResource(resourceId) {
  return sessionsForResource(resourceId).filter((session) => !session.closed && !session.pending);
}

function pendingSessionForResource(resourceId) {
  return sessionsForResource(resourceId).find((session) => !session.closed && session.pending);
}

function preferredSessionForResource(resourceId) {
  const lastSessionID = state.lastSessionByResource.get(resourceId);
  const lastSession = lastSessionID ? state.sessions.get(lastSessionID) : null;
  if (lastSession && !lastSession.closed && !lastSession.pending) {
    return lastSession;
  }

  const realSessions = realSessionsForResource(resourceId);
  if (realSessions.length > 0) {
    return realSessions[realSessions.length - 1];
  }
  return pendingSessionForResource(resourceId);
}

function sessionTitle(hostname, ordinal) {
  return ordinal > 1 ? `${hostname} #${ordinal}` : hostname;
}

function resourceTarget(resource) {
  if (isLocalResource(resource)) return 'Local shell';
  return `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
}

function terminalKindLabel(kind = 'ssh') {
  return kind === 'local' ? 'local' : 'ssh';
}

function isLocalResource(resourceOrId) {
  const id = typeof resourceOrId === 'string' ? resourceOrId : resourceOrId?.id;
  return id === LOCAL_RESOURCE_ID;
}

async function openInitialLocalSession() {
  if (!state.localShellSupported || state.sessions.size > 0) return;
  await quickConnect(localResource());
}

function tunnelForResource(resourceId) {
  if (resourceId === LOCAL_RESOURCE_ID) return null;
  return [...state.tunnels.values()].find((tunnel) => tunnel.resourceId === resourceId);
}

function tunnelsForResource(resourceId) {
  if (resourceId === LOCAL_RESOURCE_ID) return [];
  return [...state.tunnels.values()].filter((tunnel) => tunnel.resourceId === resourceId);
}

function focusSession(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session) return;
  if (!session.pending) clearPendingTabs(session.resourceId);
  setActiveSession(session.id);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function reconnectClosedSession(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session?.closed) return;
  const resource = isLocalResource(session.resourceId)
    ? localResource()
    : findResource(session.resourceId)?.resource;
  if (!resource) {
    writeNotice(`Cannot reconnect ${session.title}: resource no longer exists.`);
    return;
  }
  quickConnect(resource);
}

function setActiveSession(sessionID) {
  const session = sessionID ? state.sessions.get(sessionID) : null;
  state.activeSessionId = session?.id ?? null;
  if (session) {
    state.selectedId = session.resourceId;
    rememberSessionFocus(session.id);
    if (!session.pending && !session.closed) {
      state.lastSessionByResource.set(session.resourceId, session.id);
    }
  }
  return session;
}

function rememberSessionFocus(sessionID) {
  forgetSessionFocus(sessionID);
  state.sessionFocusHistory.push(sessionID);
}

function forgetSessionFocus(sessionID) {
  state.sessionFocusHistory = state.sessionFocusHistory.filter((id) => id !== sessionID);
}

function lastFocusedSessionID() {
  for (let index = state.sessionFocusHistory.length - 1; index >= 0; index -= 1) {
    const sessionID = state.sessionFocusHistory[index];
    if (state.sessions.has(sessionID)) return sessionID;
  }
  return state.sessions.keys().next().value ?? null;
}

async function withBusy(task) {
  if (state.busy) return;

  state.busy = true;
  setDisabledState(true);
  try {
    await task();
  } catch (error) {
    writeNotice(`Error: ${error?.message ?? error}`);
  } finally {
    state.busy = false;
    setDisabledState(false);
    renderSelection();
  }
}

function schedulePeriodicUpdateCheck() {
  const startupDelayMs = 1200;
  const periodicIntervalMs = 6 * 60 * 60 * 1000;
  window.setTimeout(runAutomaticUpdateCheck, startupDelayMs);
  window.setInterval(runAutomaticUpdateCheck, periodicIntervalMs);
}

async function runAutomaticUpdateCheck() {
  try {
    const info = await apiCheckForUpdate();
    localStorage.setItem('bashes.updateCheck.lastAt', String(Date.now()));
    if (!info?.updateAvailable) return;

    const latestVersion = String(info.latestVersion ?? '').trim();
    const notifiedVersion = localStorage.getItem('bashes.updateCheck.notifiedLatest');
    if (latestVersion && notifiedVersion === latestVersion) return;

    showUpdateModal(info, false);
    if (latestVersion) {
      localStorage.setItem('bashes.updateCheck.notifiedLatest', latestVersion);
    }
  } catch {
    // Keep update failures silent and retry at the next scheduled check.
  }
}

function setDisabledState(disabled) {
  document.querySelectorAll('button, input, select, textarea').forEach((element) => {
    element.disabled = disabled;
  });
}

function adjustTerminalFontSize(delta) {
  state.terminalFontSize = Math.min(22, Math.max(10, state.terminalFontSize + delta));
  for (const session of state.sessions.values()) {
    if (session.terminal) {
      session.terminal.options.fontSize = state.terminalFontSize;
    }
  }
  scheduleTerminalFit();
  focusActiveTerminal();
}

function focusActiveTerminal() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.terminal) return;
  requestAnimationFrame(() => session.terminal.focus());
}

function restoreTerminalFocusAfterOverlay() {
  window.setTimeout(() => {
    requestAnimationFrame(() => {
      if (hasOpenBlockingOverlay()) return;
      focusActiveTerminal();
    });
  }, 0);
}

function hasOpenBlockingOverlay() {
  return Boolean(document.querySelector(
    '.slide-panel:not([hidden]), .confirm-modal:not([hidden]), .app-modal:not([hidden]), .file-transfer-modal:not([hidden])',
  ));
}

function fitActiveTerminal() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.fitAddon || !session.terminal) return;
  session.fitAddon.fit();
  const reserveCols = terminalScrollbarReserveColumns(session);
  const cols = session.terminal.cols > reserveCols + 2 ? session.terminal.cols - reserveCols : session.terminal.cols;
  const rows = session.terminal.rows > 2 ? session.terminal.rows - 1 : session.terminal.rows;
  if (cols !== session.terminal.cols || rows !== session.terminal.rows) {
    session.terminal.resize(cols, rows);
  }
}

let terminalFitFrame = 0;

function scheduleTerminalFit() {
  if (terminalFitFrame) cancelAnimationFrame(terminalFitFrame);
  terminalFitFrame = requestAnimationFrame(() => {
    terminalFitFrame = 0;
    fitActiveTerminal();
    resizeActiveSession();
  });
}

function terminalScrollbarReserveColumns(session) {
  const viewport = session.element?.querySelector('.xterm-viewport');
  const screen = session.element?.querySelector('.xterm-screen');
  const scrollbarWidth = viewport ? viewport.offsetWidth - viewport.clientWidth : 0;
  const screenWidth = screen?.getBoundingClientRect().width ?? 0;
  const cellWidth = screenWidth > 0 && session.terminal.cols > 0 ? screenWidth / session.terminal.cols : 8;
  return Math.max(3, Math.ceil((scrollbarWidth + 12) / cellWidth));
}

function resizeActiveSession() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.terminal || session.pending || session.closed) return;
  apiResizeSSHSession(session.id, session.terminal.cols, session.terminal.rows).catch(() => {});
}

function openEditContextMenu(event) {
  const target = editableContextTarget(event.target);
  if (!target) return;

  event.preventDefault();
  event.stopPropagation();
  state.editContextTarget = target;
  const menu = document.querySelector('#edit-context-menu');
  if (!menu) return;

  const canWrite = !target.readOnly && !target.disabled;
  menu.querySelector('[data-edit-command="cut"]').disabled = !canWrite;
  menu.querySelector('[data-edit-command="paste"]').disabled = !canWrite;
  menu.hidden = false;

  const rect = menu.getBoundingClientRect();
  const left = Math.min(event.clientX, window.innerWidth - rect.width - 8);
  const top = Math.min(event.clientY, window.innerHeight - rect.height - 8);
  menu.style.left = `${Math.max(8, left)}px`;
  menu.style.top = `${Math.max(8, top)}px`;
}

function editableContextTarget(target) {
  const element = target?.closest?.('input, textarea, [contenteditable="true"]');
  if (!element || element.closest('.terminal-pane')) return null;
  if (element.matches('input')) {
    const type = (element.getAttribute('type') || 'text').toLowerCase();
    if (['button', 'checkbox', 'color', 'file', 'hidden', 'radio', 'range', 'reset', 'submit'].includes(type)) return null;
  }
  return element;
}

async function runEditContextCommand(event) {
  const button = event.target.closest?.('[data-edit-command]');
  if (!button || button.disabled || !state.editContextTarget) return;
  event.preventDefault();
  event.stopPropagation();

  const target = state.editContextTarget;
  const command = button.dataset.editCommand;
  try {
    if (command === 'copy') {
      await writeClipboard(selectedEditableText(target));
    } else if (command === 'cut') {
      await writeClipboard(selectedEditableText(target));
      replaceEditableSelection(target, '');
    } else if (command === 'paste') {
      const text = await readClipboard();
      if (text) replaceEditableSelection(target, text);
    }
  } finally {
    target.focus();
    hideEditContextMenu();
  }
}

function hideEditContextMenu() {
  const menu = document.querySelector('#edit-context-menu');
  if (menu) menu.hidden = true;
  state.editContextTarget = null;
}

function selectedEditableText(target) {
  if (target.isContentEditable) {
    return String(globalThis.getSelection?.() ?? '');
  }
  const start = target.selectionStart ?? 0;
  const end = target.selectionEnd ?? start;
  return target.value.slice(start, end);
}

function replaceEditableSelection(target, text) {
  if (target.isContentEditable) {
    document.execCommand('insertText', false, text);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: text }));
    return;
  }
  const start = target.selectionStart ?? target.value.length;
  const end = target.selectionEnd ?? start;
  const before = target.value.slice(0, start);
  const after = target.value.slice(end);
  target.value = `${before}${text}${after}`;
  const next = start + text.length;
  target.setSelectionRange(next, next);
  target.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: text ? 'insertText' : 'deleteByCut', data: text }));
}

function writeNotice(message) {
  const status = document.querySelector('#app-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
}

async function writeClipboard(text) {
  if (globalThis.runtime?.ClipboardSetText) {
    const ok = await globalThis.runtime.ClipboardSetText(text);
    if (ok !== false) return;
  }
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
  }
}

async function readClipboard() {
  if (globalThis.runtime?.ClipboardGetText) {
    return await globalThis.runtime.ClipboardGetText();
  }
  if (navigator.clipboard?.readText) {
    return await navigator.clipboard.readText();
  }
  return '';
}

function registerSSHEvents() {
  const eventsOn = globalThis.runtime?.EventsOn;
  if (!eventsOn) return;

  eventsOn('ssh:output', (event) => {
    writeSSHOutput(event.sessionId, decodeSSHOutput(event));
  });
  eventsOn('ssh:status', (event) => {
    if (event?.message) writeNotice(event.message);
  });
  eventsOn('ssh:closed', (event) => {
    const session = state.sessions.get(event.sessionId);
    if (!session) {
      state.pendingSSHOutput.delete(event.sessionId);
      return;
    }
    markSessionClosed(event.sessionId, event?.message);
    if (event?.message) writeNotice(event.message);
  });
  eventsOn('ssh:tunnel-closed', (event) => {
    if (event?.sessionId) {
      state.tunnels.delete(event.sessionId);
      renderHosts(searchInput.value);
      renderTunnelStatus();
    }
    if (event?.message) writeNotice(event.message);
  });
}

function registerAppEvents() {
  const eventsOn = globalThis.runtime?.EventsOn;
  if (!eventsOn) return;

  eventsOn('database:exported', (event) => {
    if (event?.message) writeNotice(`${event.message} ${event.path ?? ''}`.trim());
  });
  eventsOn('database:imported', async (event) => {
    clearAllSessionsFromUI();
    closeAllFileTransferWorkspaces();
    state.selectedId = null;
    await refreshHosts();
    await loadTunnels();
    writeNotice(event?.message ?? 'Database imported.');
  });
  eventsOn('database:hosts-file-imported', async (event) => {
    await refreshHosts();
    const imported = event?.imported ?? 0;
    const skipped = event?.skipped ?? 0;
    writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
    restoreTerminalFocusAfterOverlay();
  });
  eventsOn('database:hosts-file-preview', (event) => {
    showHostsFileImportPreview(event);
  });
  eventsOn('app:about', (info) => showAboutModal(info));
  eventsOn('app:update-check', (event) => {
    if (event?.error) {
      showUpdateErrorModal(event.error);
      return;
    }
    if (event?.info) showUpdateModal(event.info, Boolean(event.manual));
  });
}

function clearAllSessionsFromUI() {
  for (const sessionID of [...state.sessions.keys()]) {
    removeSessionFromUI(sessionID);
  }
  state.pendingSSHOutput.clear();
  state.lastSessionByResource.clear();
  state.sessionFocusHistory = [];
  state.activeSessionId = null;
  renderTabs();
  renderSelection();
}

function showHostsFileImportPreview(result) {
  const hosts = result?.hosts ?? [];
  const details = [
    ['Hosts file', result?.path ?? ''],
    ['Default SSH user', result?.user ?? ''],
    ['Skipped duplicates', String(result?.skipped ?? 0)],
    ...hosts.map((host, index) => [
      `Host ${index + 1}`,
      `${host.user}@${host.hostname}:${host.port}${host.ip ? ` (${host.ip})` : ''}`,
    ]),
  ];

  showAppModal({
    kicker: 'Tools',
    title: 'Import From Hosts File',
    message: hosts.length > 0
      ? `${hosts.length} host${hosts.length === 1 ? '' : 's'} will be imported.`
      : 'No new remote hosts were found.',
    details,
    primaryLabel: hosts.length > 0 ? 'Import' : 'OK',
    primaryAction: hosts.length > 0 ? () => applyHostsFileImport() : null,
    secondaryLabel: hosts.length > 0 ? 'Cancel' : '',
  });
}

async function applyHostsFileImport() {
  await withBusy(async () => {
    const result = await apiImportFromHostsFile();
    await refreshHosts();
    const imported = result?.imported ?? 0;
    const skipped = result?.skipped ?? 0;
    writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
    restoreTerminalFocusAfterOverlay();
  });
}

function showAboutModal(info) {
  const appInfo = info ?? {};
  showAppModal({
    kicker: 'About',
    title: 'Bashes',
    message: 'Fast remote server session manager.',
    details: [
      ['Version', appInfo.version ?? 'dev'],
      ['Platform', `${appInfo.platform ?? 'unknown'}/${appInfo.arch ?? 'unknown'}`],
      ['Data file', appInfo.dataPath ?? ''],
    ],
    primaryLabel: 'README',
    primaryAction: () => openExternalURL(appInfo.readmeUrl ?? 'https://github.com/signoredellarete/bashes#readme'),
    secondaryLabel: 'Releases',
    secondaryAction: () => openExternalURL(appInfo.releasesUrl ?? 'https://github.com/signoredellarete/bashes/releases'),
  });
}

function showUpdateModal(info, manual = false) {
  if (!info) return;
  if (!info.updateAvailable && !manual) return;
  showAppModal({
    kicker: 'Updates',
    title: info.updateAvailable ? 'Update Available' : 'Bashes Is Up To Date',
    message: info.message ?? '',
    details: [
      ['Current version', info.currentVersion ?? 'dev'],
      ['Latest version', info.latestVersion ?? 'unknown'],
    ],
    primaryLabel: info.updateAvailable ? 'Open Releases' : 'OK',
    primaryAction: info.updateAvailable ? () => openExternalURL(info.releaseUrl ?? info.repoUrl) : null,
    secondaryLabel: info.updateAvailable ? 'Repository' : '',
    secondaryAction: info.updateAvailable ? () => openExternalURL(info.repoUrl) : null,
  });
}

function showUpdateErrorModal(message) {
  showAppModal({
    kicker: 'Updates',
    title: 'Could Not Check Updates',
    message,
    primaryLabel: 'Repository',
    primaryAction: () => openExternalURL('https://github.com/signoredellarete/bashes'),
    secondaryLabel: 'OK',
  });
}

function openExternalURL(url) {
  if (!url) return;
  if (globalThis.runtime?.BrowserOpenURL) {
    globalThis.runtime.BrowserOpenURL(url);
    return;
  }
  window.open(url, '_blank', 'noopener');
}

function writeSSHOutput(sessionID, data) {
  if (!sessionID || !data || data.length === 0) return;
  const session = state.sessions.get(sessionID);
  if (session?.terminal) {
    session.terminal.write(data);
    return;
  }

  const current = state.pendingSSHOutput.get(sessionID);
  const chunks = Array.isArray(current) ? current : current ? [current] : [];
  chunks.push(data);
  state.pendingSSHOutput.set(sessionID, trimPendingSSHOutput(chunks));
}

function flushPendingSSHOutput(sessionID) {
  const pending = state.pendingSSHOutput.get(sessionID);
  if (!pending) return;
  state.pendingSSHOutput.delete(sessionID);
  const session = state.sessions.get(sessionID);
  if (!session?.terminal) return;

  const chunks = Array.isArray(pending) ? pending : [pending];
  chunks.forEach((chunk) => session.terminal.write(chunk));
}

function decodeSSHOutput(event) {
  if (event?.bytes) return base64ToBytes(event.bytes);
  return event?.data ?? '';
}

function base64ToBytes(value) {
  const binary = atob(String(value));
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function trimPendingSSHOutput(chunks) {
  const limit = 262144;
  let total = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  while (chunks.length > 1 && total > limit) {
    const removed = chunks.shift();
    total -= removed.length;
  }
  if (total <= limit) return chunks;

  const first = chunks[0];
  const start = first.length - limit;
  chunks[0] = typeof first === 'string' ? first.slice(start) : first.slice(start);
  return chunks;
}

function wailsAPI() {
  return globalThis.go?.main?.App ?? globalThis.go?.desktop?.App;
}

async function apiListHosts() {
  const api = wailsAPI();
  if (api?.ListHosts) return (await api.ListHosts()) ?? [];
  return clone(demoStore.hosts);
}

async function apiGetAppInfo() {
  const api = wailsAPI();
  if (api?.GetAppInfo) return await api.GetAppInfo();
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
}

async function apiCheckForUpdate() {
  const api = wailsAPI();
  if (api?.CheckForUpdate) return await api.CheckForUpdate();
  return {
    currentVersion: 'dev',
    latestVersion: 'dev',
    updateAvailable: false,
    releaseUrl: 'https://github.com/signoredellarete/bashes/releases',
    repoUrl: 'https://github.com/signoredellarete/bashes',
    message: 'Update check is available only in the desktop app.',
  };
}

async function apiSupportsLocalShell() {
  const api = wailsAPI();
  if (api?.SupportsLocalShell) return Boolean(await api.SupportsLocalShell());
  return false;
}

async function apiAddHost(input) {
  const api = wailsAPI();
  if (api?.AddHost) return await api.AddHost(input);
  const host = { id: `host-${Date.now()}`, hostname: input.hostname, ip: input.ip, port: input.port, user: input.user, subsystems: [] };
  demoStore.hosts.push(host);
  return clone(host);
}

async function apiAddSubsystem(hostID, input) {
  const api = wailsAPI();
  if (api?.AddSubsystem) return await api.AddSubsystem(hostID, input);
  const parent = findDemoResource(hostID);
  if (!parent) throw new Error(`Resource ${hostID} not found`);
  if (!parent.subsystems) parent.subsystems = [];
  const subsystem = { id: `${input.type}-${Date.now()}`, type: input.type, hostname: input.hostname, ip: input.ip, port: input.port, user: input.user, subsystems: [] };
  parent.subsystems.push(subsystem);
  return clone(subsystem);
}

async function apiUpdateResource(id, input) {
  const api = wailsAPI();
  if (api?.UpdateResource) return await api.UpdateResource(id, input);
  for (const host of demoStore.hosts) {
    if (host.id === id) {
      host.hostname = input.hostname;
      host.ip = input.ip;
      host.port = input.port;
      host.user = input.user;
      return;
    }
    const resource = findDemoNestedResource(host.subsystems ?? [], id);
    if (resource) {
      resource.type = input.type;
      resource.hostname = input.hostname;
      resource.ip = input.ip;
      resource.port = input.port;
      resource.user = input.user;
      return;
    }
  }
  throw new Error(`Resource ${id} not found`);
}

async function apiReorderHosts(order) {
  const api = wailsAPI();
  if (api?.ReorderHosts) return await api.ReorderHosts(order);
  if (order.length !== demoStore.hosts.length) throw new Error('Invalid host order');
  const byID = new Map(demoStore.hosts.map((host) => [host.id, host]));
  demoStore.hosts = order.map((id) => {
    const host = byID.get(id);
    if (!host) throw new Error(`Host ${id} not found`);
    return host;
  });
}

async function apiDeleteResource(id) {
  const api = wailsAPI();
  if (api?.DeleteResource) return await api.DeleteResource(id);
  const hostIndex = demoStore.hosts.findIndex((host) => host.id === id);
  if (hostIndex >= 0) {
    demoStore.hosts.splice(hostIndex, 1);
    return;
  }
  for (const host of demoStore.hosts) {
    if (deleteDemoNestedResource(host.subsystems ?? [], id)) {
      return;
    }
  }
  throw new Error(`Resource ${id} not found`);
}

function findDemoResource(id) {
  for (const host of demoStore.hosts) {
    if (host.id === id) return host;
    const subsystem = findDemoNestedResource(host.subsystems ?? [], id);
    if (subsystem) return subsystem;
  }
  return null;
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
    if (deleteDemoNestedResource(subsystems[index].subsystems ?? [], id)) {
      return true;
    }
  }
  return false;
}

async function apiStartSSHSession(input) {
  const api = wailsAPI();
  if (api?.StartSSHSession) return await api.StartSSHSession(input);
  return `demo-session-${Date.now()}`;
}

async function apiStartLocalSession(input) {
  const api = wailsAPI();
  if (api?.StartLocalSession) return await api.StartLocalSession(input);
  throw new Error('Local shell is available only in the desktop app.');
}

async function apiWriteSSHSession(sessionID, data) {
  const api = wailsAPI();
  if (api?.WriteSSHSession) return await api.WriteSSHSession(sessionID, data);
  state.sessions.get(sessionID)?.terminal.write(data);
}

async function apiResizeSSHSession(sessionID, cols, rows) {
  const api = wailsAPI();
  if (api?.ResizeSSHSession) return await api.ResizeSSHSession(sessionID, cols, rows);
}

async function apiStopSSHSession(sessionID) {
  const api = wailsAPI();
  if (api?.StopSSHSession) return await api.StopSSHSession(sessionID);
}

async function apiListSSHTunnels() {
  const api = wailsAPI();
  if (api?.ListSSHTunnels) return (await api.ListSSHTunnels()) ?? [];
  return [];
}

async function apiStartSSHTunnel(input) {
  const api = wailsAPI();
  if (api?.StartSSHTunnel) return await api.StartSSHTunnel(input);
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
}

async function apiStopSSHTunnel(tunnelID) {
  const api = wailsAPI();
  if (api?.StopSSHTunnel) return await api.StopSSHTunnel(tunnelID);
}

async function apiListSSHKeys() {
  const api = wailsAPI();
  if (api?.ListSSHKeys) return (await api.ListSSHKeys()) ?? [];
  return clone(demoStore.keys);
}

async function apiGetSSHKeySettings() {
  const api = wailsAPI();
  if (api?.GetSSHKeySettings) return await api.GetSSHKeySettings();
  return { customDirectory: localStorage.getItem('bashes.keys.customDirectory') || '' };
}

async function apiSaveSSHKeySettings(input) {
  const api = wailsAPI();
  if (api?.SaveSSHKeySettings) return await api.SaveSSHKeySettings(input);
  localStorage.setItem('bashes.keys.customDirectory', input.customDirectory || '');
  return { customDirectory: input.customDirectory || '' };
}

async function apiImportFromHostsFile() {
  const api = wailsAPI();
  if (api?.ImportFromHostsFile) return await api.ImportFromHostsFile();
  return { imported: 0, skipped: 0, hosts: [], hostnames: [] };
}

async function apiGenerateSSHKey(input) {
  const api = wailsAPI();
  if (api?.GenerateSSHKey) return await api.GenerateSSHKey(input);
  const key = { name: input.name || `bashes-${Date.now()}`, privateKey: '', publicKey: '', source: 'bashes' };
  demoStore.keys.push(key);
  return clone(key);
}

async function apiReadSSHPublicKey(name) {
  const api = wailsAPI();
  if (api?.ReadSSHPublicKey) return await api.ReadSSHPublicKey(name);
  return `ssh-ed25519 demo ${name}`;
}

async function apiReadSSHPublicKeyPath(path) {
  const api = wailsAPI();
  if (api?.ReadSSHPublicKeyPath) return await api.ReadSSHPublicKeyPath(path);
  return `ssh-ed25519 demo ${path}`;
}

async function apiInstallSSHKey(input) {
  const api = wailsAPI();
  if (api?.InstallSSHKey) return await api.InstallSSHKey(input);
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}
