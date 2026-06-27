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
  selectedId: null,
  activeSessionId: null,
  terminalFontSize: 13,
  busy: false,
  drawerMode: null,
  drawerHostId: null,
  editResourceId: null,
  sessions: new Map(),
};

const app = document.querySelector('#app');

app.innerHTML = `
  <aside class="sidebar">
    <header class="brand">
      <img src="${bashesLogo}" alt="" />
      <div>
        <h1>Bashes</h1>
        <span>Remote sessions</span>
      </div>
    </header>

    <div class="toolbar">
      <input id="search" type="search" placeholder="Search hosts" autocomplete="off" />
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
        <input name="hostname" autocomplete="off" required />
      </label>
      <label>
        <span>IP / DNS</span>
        <input name="ip" autocomplete="off" required />
      </label>
      <div class="form-grid">
        <label>
          <span>Port</span>
          <input name="port" type="number" min="1" max="65535" value="22" required />
        </label>
        <label>
          <span>User</span>
          <input name="user" autocomplete="username" required />
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

      <label>
        <span>Bashes Key</span>
        <select name="keyName"></select>
      </label>
      <label>
        <span>Password</span>
        <input name="password" type="password" autocomplete="current-password" />
      </label>
      <label>
        <span>Private Key Path</span>
        <input name="privateKeyPath" autocomplete="off" placeholder="optional path on this machine" />
      </label>
      <label>
        <span>Key Passphrase</span>
        <input name="privateKeyPassphrase" type="password" autocomplete="off" />
      </label>
      <label class="checkbox-row">
        <input name="trustHostKey" type="checkbox" checked />
        <span>Trust host key for this session</span>
      </label>

      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-connect>Cancel</button>
        <button type="submit">Connect</button>
      </footer>
    </form>
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
          <input name="name" autocomplete="off" placeholder="bashes-main" />
        </label>
        <button type="submit">Generate</button>
      </form>

      <label>
        <span>Existing Key</span>
        <select id="key-select"></select>
      </label>
      <label>
        <span>Public Key</span>
        <textarea id="public-key" rows="5" readonly></textarea>
      </label>

      <form id="key-install-form" class="compact-form">
        <p class="parent-summary" id="key-install-summary">Select a host or subsystem to install the key.</p>
        <label>
          <span>Remote Password</span>
          <input name="password" type="password" autocomplete="current-password" />
        </label>
        <label class="checkbox-row">
          <input name="trustHostKey" type="checkbox" checked />
          <span>Trust host key for this install</span>
        </label>
        <button type="submit">Install On Selected</button>
      </form>
    </section>
  </section>
`;

const stack = document.querySelector('#terminal-stack');

window.addEventListener('resize', () => {
  scheduleTerminalFit();
});

if (globalThis.ResizeObserver) {
  const terminalResizeObserver = new ResizeObserver(() => scheduleTerminalFit());
  terminalResizeObserver.observe(stack);
}

registerSSHEvents();

const searchInput = document.querySelector('#search');
searchInput.addEventListener('input', () => scheduleHostRender());
['beforeinput', 'keydown', 'keyup'].forEach((eventName) => {
  searchInput.addEventListener(eventName, (event) => event.stopPropagation());
});
document.querySelector('#open-host-panel').addEventListener('click', () => openResourcePanel('host'));
document.querySelector('#open-keys-panel').addEventListener('click', () => openKeysPanel());
document.querySelector('#edit-resource').addEventListener('click', () => openEditPanel());
document.querySelector('#header-add-subsystem').addEventListener('click', () => {
  const selected = findResource(state.selectedId);
  const hostID = selected?.type === 'host' ? selected.resource.id : selected?.parent?.id;
  if (hostID) openResourcePanel('subsystem', hostID);
});
document.querySelector('#connect').addEventListener('click', () => openConnectPanel());
document.querySelector('#disconnect').addEventListener('click', () => disconnectActiveSession());
document.querySelector('#delete-resource').addEventListener('click', () => deleteSelectedResource());
document.querySelector('#decrease-terminal-font').addEventListener('click', () => adjustTerminalFontSize(-1));
document.querySelector('#increase-terminal-font').addEventListener('click', () => adjustTerminalFontSize(1));
document.querySelector('#resource-form').addEventListener('submit', (event) => submitResource(event));
document.querySelector('#connect-form').addEventListener('submit', (event) => submitConnect(event));
document.querySelector('#key-generate-form').addEventListener('submit', (event) => submitGenerateKey(event));
document.querySelector('#key-install-form').addEventListener('submit', (event) => submitInstallKey(event));
document.querySelector('#key-select').addEventListener('change', () => renderSelectedPublicKey());
document.querySelectorAll('[data-close-panel]').forEach((element) => {
  element.addEventListener('click', () => closeResourcePanel());
});
document.querySelectorAll('[data-close-connect]').forEach((element) => {
  element.addEventListener('click', () => closeConnectPanel());
});
document.querySelectorAll('[data-close-keys]').forEach((element) => {
  element.addEventListener('click', () => closeKeysPanel());
});

await loadHosts();
await loadKeys();

async function loadHosts() {
  await withBusy(async () => {
    await refreshHosts();
  });
}

async function refreshHosts() {
  state.hosts = await apiListHosts();
  if (state.selectedId && !findResource(state.selectedId)) {
    state.selectedId = null;
  }
  if (!state.selectedId && state.hosts.length > 0) {
    state.selectedId = state.hosts[0].id;
  }
  renderHosts(searchInput.value);
  renderSelection();
}

async function loadKeys() {
  state.keys = await apiListSSHKeys();
  renderKeyOptions();
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
      state.selectedId = subsystem.id;
      writeNotice(`Added ${subsystem.type} ${subsystem.hostname}.`);
    } else {
      const host = await apiAddHost(input);
      state.selectedId = host.id;
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

  const existing = sessionForResource(selected.id);
  if (existing && !existing.pending) {
    focusSession(existing.id);
    closeConnectPanel();
    return;
  }

  const form = event.currentTarget;
  await withBusy(async () => {
    const target = `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
    writeNotice(`Connecting to ${target} ...`);
    const sessionID = await apiStartSSHSession({
      resourceId: selected.id,
      keyName: form.elements.keyName.value,
      password: form.elements.password.value,
      privateKeyPath: form.elements.privateKeyPath.value.trim(),
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
      trustHostKey: form.elements.trustHostKey.checked,
      cols: 120,
      rows: 32,
    });
    createSession(sessionID, selected);
    closeConnectPanel();
    await refreshHosts();
    resizeActiveSession();
  });
}

async function quickConnect(resource) {
  await withBusy(async () => {
    try {
      writeNotice(`Connecting to ${resource.user}@${resource.ip || resource.hostname}:${resource.port} ...`);
      const sessionID = await apiStartSSHSession({
        resourceId: resource.id,
        ...authInputFromPreference(resource),
        trustHostKey: true,
        cols: 120,
        rows: 32,
      });
      createSession(sessionID, resource);
      await refreshHosts();
      resizeActiveSession();
    } catch (error) {
      writeNotice(`Connection needs credentials: ${error?.message ?? error}`);
      await openConnectPanel();
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
    document.querySelector('#key-select').value = key.name;
    await renderSelectedPublicKey();
    writeNotice(`Generated SSH key ${key.name}.`);
  });
}

async function submitInstallKey(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  const keyName = document.querySelector('#key-select').value;
  if (!selected || !keyName) return;

  const form = event.currentTarget;
  await withBusy(async () => {
    await apiInstallSSHKey({
      resourceId: selected.id,
      keyName,
      password: form.elements.password.value,
      trustHostKey: form.elements.trustHostKey.checked,
    });
    form.reset();
    form.elements.trustHostKey.checked = true;
    await refreshHosts();
    writeNotice(`Installed key ${keyName} on ${selected.hostname}.`);
  });
}

async function deleteSelectedResource() {
  const selected = findResource(state.selectedId);
  if (!selected) return;

  await withBusy(async () => {
    for (const session of sessionsForResource(selected.resource.id)) {
      await stopSession(session.id);
    }
    await apiDeleteResource(selected.resource.id);
    writeNotice(`Deleted ${selected.resource.hostname}.`);
    state.selectedId = selected.parent?.id ?? null;
    await refreshHosts();
  });
}

async function disconnectActiveSession() {
  if (!state.activeSessionId) return;
  await stopSession(state.activeSessionId);
}

async function stopSession(sessionID) {
  const session = state.sessions.get(sessionID);
  state.sessions.delete(sessionID);
  if (session?.element) session.element.remove();
  if (!session?.pending) {
    await apiStopSSHSession(sessionID);
  }
  if (state.activeSessionId === sessionID) {
    state.activeSessionId = firstSessionID();
  }
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
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
  const rows = [];

  for (const host of state.hosts) {
    rows.push(resourceRow(host, 'host'));
    for (const subsystem of host.subsystems ?? []) {
      rows.push(resourceRow(subsystem, subsystem.type, true));
    }
  }

  const visibleRows = rows.filter((row) => row.search.includes(query));
  container.replaceChildren(...visibleRows.map((row) => row.element));
}

function resourceRow(resource, type, child = false) {
  const row = document.createElement('div');
  row.className = `host-row ${child ? 'child' : ''}`;
  row.dataset.id = resource.id;

  const selectButton = document.createElement('button');
  selectButton.type = 'button';
  selectButton.className = 'host-select';
  selectButton.innerHTML = `
    <span class="type"></span>
    <span class="details">
      <strong></strong>
      <small></small>
    </span>
  `;
  selectButton.querySelector('.type').textContent = type;
  selectButton.querySelector('strong').textContent = resource.hostname;
  selectButton.querySelector('small').textContent = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  selectButton.addEventListener('click', () => {
    state.selectedId = resource.id;
    const session = sessionForResource(resource.id);
    state.activeSessionId = session?.id ?? createPendingTab(resource);
    renderTabs();
    renderSelection();
    scheduleTerminalFit();
  });
  selectButton.addEventListener('dblclick', () => {
    state.selectedId = resource.id;
    const session = sessionForResource(resource.id);
    if (session && !session.pending) {
      state.activeSessionId = session.id;
      renderTabs();
      renderSelection();
      scheduleTerminalFit();
      return;
    }
    createPendingTab(resource);
    quickConnect(resource);
  });
  row.append(selectButton);

  return {
    element: row,
    search: `${resource.hostname} ${resource.ip} ${resource.user} ${type}`.toLowerCase(),
  };
}

function createPendingTab(resource) {
  clearPendingTabs(resource.id);
  const existing = sessionForResource(resource.id);
  if (existing) {
    state.activeSessionId = existing.id;
    return existing.id;
  }

  const sessionID = `pending-${resource.id}`;
  const pane = document.createElement('section');
  pane.className = 'terminal-pane pending-pane';
  pane.dataset.sessionId = sessionID;
  pane.textContent = `Ready to connect to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  stack.append(pane);

  state.sessions.set(sessionID, {
    id: sessionID,
    resourceId: resource.id,
    title: resource.hostname,
    target: `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`,
    element: pane,
    pending: true,
    closed: false,
  });
  state.activeSessionId = sessionID;
  return sessionID;
}

function clearPendingTabs(exceptResourceId = '') {
  for (const session of [...state.sessions.values()]) {
    if (!session.pending || session.resourceId === exceptResourceId) continue;
    if (session.element) session.element.remove();
    state.sessions.delete(session.id);
    if (state.activeSessionId === session.id) state.activeSessionId = null;
  }
}

function createSession(sessionID, resource) {
  const pending = sessionForResource(resource.id);
  if (pending?.pending) {
    pending.element.remove();
    state.sessions.delete(pending.id);
  }

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
  terminal.onData((data) => {
    apiWriteSSHSession(sessionID, data).catch((error) => {
      terminal.writeln(`\r\nError: ${error?.message ?? error}`);
    });
  });
  terminal.onSelectionChange(() => {
    const selected = terminal.getSelection();
    if (!selected) return;
    writeClipboard(selected).catch(() => {});
  });
  pane.addEventListener('contextmenu', (event) => {
    event.preventDefault();
    readClipboard()
      .then((text) => {
        if (text) return apiWriteSSHSession(sessionID, text);
      })
      .catch(() => {});
  });

  state.sessions.set(sessionID, {
    id: sessionID,
    resourceId: resource.id,
    title: resource.hostname,
    target: `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`,
    terminal,
    fitAddon,
    element: pane,
    closed: false,
  });
  state.activeSessionId = sessionID;
  state.selectedId = resource.id;
  terminal.writeln(`Connected to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
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
    tab.innerHTML = '<span></span><strong></strong>';
    tab.querySelector('span').textContent = session.closed ? 'closed' : session.pending ? 'new' : 'ssh';
    tab.querySelector('strong').textContent = session.title;
    tab.addEventListener('click', () => {
      state.activeSessionId = session.id;
      state.selectedId = session.resourceId;
      renderTabs();
      renderSelection();
      scheduleTerminalFit();
    });

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
    wrapper.append(tab, close);
    return wrapper;
  }));

  document.querySelectorAll('.terminal-pane').forEach((pane) => {
    pane.hidden = pane.dataset.sessionId !== state.activeSessionId;
  });
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

  const host = state.hosts.find((candidate) => candidate.id === hostID);
  const subsystemMode = mode === 'subsystem';
  typeField.hidden = !subsystemMode;
  parentSummary.hidden = !subsystemMode;
  if (subsystemMode) {
    kicker.textContent = 'Subsystem';
    title.textContent = 'Add Subsystem';
    submit.textContent = 'Add Subsystem';
    parentSummary.textContent = host ? `Parent host: ${host.hostname} (${host.user}@${host.ip || host.hostname})` : '';
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
      ? `Parent host: ${selected.parent.hostname} (${selected.parent.user}@${selected.parent.ip || selected.parent.hostname})`
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
}

async function openConnectPanel() {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;

  await loadKeys();
  const panel = document.querySelector('#connect-panel');
  const form = document.querySelector('#connect-form');
  form.reset();
  form.elements.trustHostKey.checked = true;
  renderKeyOptions(form.elements.keyName, true);
  applyConnectDefaults(form, selected);
  document.querySelector('#connect-summary').textContent =
    `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  form.elements.password.focus();
}

function closeConnectPanel() {
  const panel = document.querySelector('#connect-panel');
  panel.classList.remove('open');
  panel.hidden = true;
}

async function openKeysPanel() {
  await loadKeys();
  renderKeyInstallSummary();
  const panel = document.querySelector('#keys-panel');
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
}

function closeKeysPanel() {
  const panel = document.querySelector('#keys-panel');
  panel.classList.remove('open');
  panel.hidden = true;
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
  const connect = document.querySelector('#connect');
  const disconnect = document.querySelector('#disconnect');
  const remove = document.querySelector('#delete-resource');

  if (activeSession) {
    title.textContent = activeSession.target;
  } else if (selected) {
    title.textContent = `${selected.resource.user}@${selected.resource.hostname}`;
  } else {
    title.textContent = 'No session selected';
  }

  if (!selected) {
    edit.disabled = true;
    addSubsystem.disabled = true;
    keys.disabled = true;
    connect.disabled = true;
    disconnect.disabled = true;
    remove.disabled = true;
    return;
  }

  edit.disabled = state.busy || !selected;
  addSubsystem.disabled = state.busy || !selected;
  keys.disabled = state.busy || !selected;
  connect.disabled = state.busy || !selected;
  disconnect.disabled = state.busy || !activeSession;
  remove.disabled = state.busy || !selected;
  renderKeyInstallSummary();
}

function renderKeyOptions(select = document.querySelector('#key-select'), includeEmpty = false) {
  if (!select) return;
  const options = [];
  if (includeEmpty) {
    const empty = document.createElement('option');
    empty.value = '';
    empty.textContent = 'Use agent/default/path';
    options.push(empty);
  }
  for (const key of state.keys) {
    const option = document.createElement('option');
    option.value = key.name;
    option.textContent = key.name;
    options.push(option);
  }
  select.replaceChildren(...options);
  renderSelectedPublicKey();
}

function applyConnectDefaults(form, resource) {
  const auth = resource?.auth;
  if (!auth) return;

  if (auth.trustHostKey) {
    form.elements.trustHostKey.checked = true;
  }
  if (auth.method === 'key' && auth.keyName) {
    const hasKey = [...form.elements.keyName.options].some((option) => option.value === auth.keyName);
    if (hasKey) form.elements.keyName.value = auth.keyName;
  }
  if (auth.method === 'path' && auth.privateKeyPath) {
    form.elements.privateKeyPath.value = auth.privateKeyPath;
  }
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

async function renderSelectedPublicKey() {
  const select = document.querySelector('#key-select');
  const output = document.querySelector('#public-key');
  if (!select || !output) return;
  if (!select.value) {
    output.value = '';
    return;
  }
  output.value = await apiReadSSHPublicKey(select.value);
}

function renderKeyInstallSummary() {
  const summary = document.querySelector('#key-install-summary');
  if (!summary) return;
  const selected = findResource(state.selectedId)?.resource;
  summary.textContent = selected
    ? `Install selected key on ${selected.user}@${selected.ip || selected.hostname}:${selected.port}`
    : 'Select a host or subsystem to install the key.';
}

function findResource(id) {
  for (const host of state.hosts) {
    if (host.id === id) return { type: 'host', resource: host, parent: null };
    for (const subsystem of host.subsystems ?? []) {
      if (subsystem.id === id) return { type: subsystem.type, resource: subsystem, parent: host };
    }
  }
  return null;
}

function sessionForResource(resourceId) {
  return [...state.sessions.values()].find((session) => session.resourceId === resourceId && !session.closed);
}

function sessionsForResource(resourceId) {
  return [...state.sessions.values()].filter((session) => session.resourceId === resourceId);
}

function focusSession(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session) return;
  state.activeSessionId = session.id;
  state.selectedId = session.resourceId;
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
}

function firstSessionID() {
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

function fitActiveTerminal() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.fitAddon || !session.terminal) return;
  session.fitAddon.fit();
  const reserveCols = terminalScrollbarReserveColumns(session);
  if (session.terminal.cols > reserveCols + 2) {
    session.terminal.resize(session.terminal.cols - reserveCols, session.terminal.rows);
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
  if (!session?.terminal || session.pending) return;
  apiResizeSSHSession(session.id, session.terminal.cols, session.terminal.rows).catch(() => {});
}

function writeNotice(message) {
  const session = state.sessions.get(state.activeSessionId);
  if (session?.terminal) {
    session.terminal.writeln(`\r\n${message}`);
    return;
  }
  if (session?.element) {
    session.element.textContent = message;
    return;
  }
  const empty = document.querySelector('#empty-terminal');
  empty.textContent = message;
}

async function writeClipboard(text) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
  }
}

async function readClipboard() {
  if (navigator.clipboard?.readText) {
    return await navigator.clipboard.readText();
  }
  return '';
}

function registerSSHEvents() {
  const eventsOn = globalThis.runtime?.EventsOn;
  if (!eventsOn) return;

  eventsOn('ssh:output', (event) => {
    const session = state.sessions.get(event.sessionId);
    if (session?.terminal) session.terminal.write(event.data ?? '');
  });
  eventsOn('ssh:status', (event) => {
    const session = state.sessions.get(event.sessionId);
    if (session?.terminal) session.terminal.writeln(`\r\n${event.message}`);
  });
  eventsOn('ssh:closed', (event) => {
    const session = state.sessions.get(event.sessionId);
    if (!session) return;
    session.closed = true;
    if (session.terminal) session.terminal.writeln(`\r\n${event.message}`);
    renderTabs();
    renderSelection();
  });
}

function wailsAPI() {
  return globalThis.go?.main?.App ?? globalThis.go?.desktop?.App;
}

async function apiListHosts() {
  const api = wailsAPI();
  if (api?.ListHosts) return (await api.ListHosts()) ?? [];
  return clone(demoStore.hosts);
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
  const host = demoStore.hosts.find((candidate) => candidate.id === hostID);
  if (!host) throw new Error(`Host ${hostID} not found`);
  const subsystem = { id: `${input.type}-${Date.now()}`, type: input.type, hostname: input.hostname, ip: input.ip, port: input.port, user: input.user };
  host.subsystems.push(subsystem);
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
    const subsystem = host.subsystems.find((candidate) => candidate.id === id);
    if (subsystem) {
      subsystem.type = input.type;
      subsystem.hostname = input.hostname;
      subsystem.ip = input.ip;
      subsystem.port = input.port;
      subsystem.user = input.user;
      return;
    }
  }
  throw new Error(`Resource ${id} not found`);
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
    const index = host.subsystems.findIndex((subsystem) => subsystem.id === id);
    if (index >= 0) {
      host.subsystems.splice(index, 1);
      return;
    }
  }
  throw new Error(`Resource ${id} not found`);
}

async function apiStartSSHSession(input) {
  const api = wailsAPI();
  if (api?.StartSSHSession) return await api.StartSSHSession(input);
  return `demo-session-${Date.now()}`;
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

async function apiListSSHKeys() {
  const api = wailsAPI();
  if (api?.ListSSHKeys) return (await api.ListSSHKeys()) ?? [];
  return clone(demoStore.keys);
}

async function apiGenerateSSHKey(input) {
  const api = wailsAPI();
  if (api?.GenerateSSHKey) return await api.GenerateSSHKey(input);
  const key = { name: input.name || `bashes-${Date.now()}`, privateKey: '', publicKey: '' };
  demoStore.keys.push(key);
  return clone(key);
}

async function apiReadSSHPublicKey(name) {
  const api = wailsAPI();
  if (api?.ReadSSHPublicKey) return await api.ReadSSHPublicKey(name);
  return `ssh-ed25519 demo ${name}`;
}

async function apiInstallSSHKey(input) {
  const api = wailsAPI();
  if (api?.InstallSSHKey) return await api.InstallSSHKey(input);
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}
