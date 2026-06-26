import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { EventsOn } from '../wailsjs/runtime/runtime.js';
import '@xterm/xterm/css/xterm.css';
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
};

const state = {
  hosts: [],
  selectedId: null,
  activeSessionId: null,
  busy: false,
  drawerMode: null,
  drawerHostId: null,
};

const app = document.querySelector('#app');

app.innerHTML = `
  <aside class="sidebar">
    <header class="brand">
      <img src="/src/assets/bashes.png" alt="" />
      <div>
        <h1>Bashes</h1>
        <span>Remote sessions</span>
      </div>
    </header>

    <div class="toolbar">
      <input id="search" type="search" placeholder="Search hosts" autocomplete="off" />
      <button id="refresh" class="secondary" type="button" title="Refresh hosts">Refresh</button>
      <button id="open-host-panel" type="button" title="Add host">Add Host</button>
    </div>

    <section id="hosts" class="hosts" aria-label="Hosts"></section>
  </aside>

  <main class="workspace">
    <section class="session-header">
      <div>
        <p class="eyebrow">Session</p>
        <h2 id="session-title">No session selected</h2>
      </div>
      <div class="session-actions">
        <button id="delete-resource" class="secondary" type="button" disabled>Delete</button>
        <button id="disconnect" class="secondary" type="button" disabled>Disconnect</button>
        <button id="connect" type="button" disabled>Connect</button>
      </div>
    </section>

    <section class="workbench">
      <section id="terminal" class="terminal" aria-label="Terminal"></section>
    </section>
  </main>

  <section id="resource-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-panel></div>
    <form id="resource-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow" id="resource-panel-kicker">Resource</p>
          <h3 id="resource-panel-title">Add Host</h3>
        </div>
        <button class="icon-only secondary" type="button" data-close-panel title="Close">x</button>
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
        <button class="icon-only secondary" type="button" data-close-connect title="Close">x</button>
      </header>

      <p id="connect-summary" class="parent-summary"></p>

      <label>
        <span>Password</span>
        <input name="password" type="password" autocomplete="current-password" />
      </label>
      <label>
        <span>Private Key Path</span>
        <input name="privateKeyPath" autocomplete="off" placeholder="optional, defaults to ~/.ssh keys" />
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
`;

const terminalElement = document.querySelector('#terminal');
const terminal = new Terminal({
  cursorBlink: true,
  convertEol: true,
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
  fontSize: 13,
  theme: {
    background: '#101418',
    foreground: '#d7dde5',
    cursor: '#f5c542',
    selectionBackground: '#3d4a58',
  },
});
const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
terminal.open(terminalElement);
fitTerminal();
terminal.writeln('Bashes terminal ready.');

window.addEventListener('resize', () => {
  fitTerminal();
  resizeActiveSession();
});

terminal.onData((data) => {
  if (!state.activeSessionId) return;
  apiWriteSSHSession(state.activeSessionId, data).catch((error) => {
    terminal.writeln(`\r\nError: ${error?.message ?? error}`);
  });
});

registerSSHEvents();

document.querySelector('#refresh').addEventListener('click', () => loadHosts());
document.querySelector('#search').addEventListener('input', (event) => renderHosts(event.target.value));
document.querySelector('#open-host-panel').addEventListener('click', () => openResourcePanel('host'));
document.querySelector('#connect').addEventListener('click', () => openConnectPanel());
document.querySelector('#disconnect').addEventListener('click', () => disconnectActiveSession());
document.querySelector('#delete-resource').addEventListener('click', () => deleteSelectedResource());
document.querySelector('#resource-form').addEventListener('submit', (event) => submitResource(event));
document.querySelector('#connect-form').addEventListener('submit', (event) => submitConnect(event));
document.querySelectorAll('[data-close-panel]').forEach((element) => {
  element.addEventListener('click', () => closeResourcePanel());
});
document.querySelectorAll('[data-close-connect]').forEach((element) => {
  element.addEventListener('click', () => closeConnectPanel());
});

await loadHosts();

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
  renderHosts(document.querySelector('#search').value);
  renderSelection();
}

async function submitResource(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const input = endpointInput(form, state.drawerMode === 'subsystem' ? form.elements.type.value : '');

  await withBusy(async () => {
    if (state.drawerMode === 'subsystem') {
      const subsystem = await apiAddSubsystem(state.drawerHostId, input);
      state.selectedId = subsystem.id;
      terminal.writeln(`Added ${subsystem.type} ${subsystem.hostname}.`);
    } else {
      const host = await apiAddHost(input);
      state.selectedId = host.id;
      terminal.writeln(`Added host ${host.hostname}.`);
    }
    closeResourcePanel();
    await refreshHosts();
  });
}

async function submitConnect(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;

  const form = event.currentTarget;
  await withBusy(async () => {
    if (state.activeSessionId) {
      await apiStopSSHSession(state.activeSessionId);
      state.activeSessionId = null;
    }

    terminal.clear();
    terminal.writeln(`Connecting to ${selected.user}@${selected.ip || selected.hostname}:${selected.port} ...`);
    const sessionID = await apiStartSSHSession({
      resourceId: selected.id,
      password: form.elements.password.value,
      privateKeyPath: form.elements.privateKeyPath.value.trim(),
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
      trustHostKey: form.elements.trustHostKey.checked,
      cols: terminal.cols,
      rows: terminal.rows,
    });
    state.activeSessionId = sessionID;
    closeConnectPanel();
    resizeActiveSession();
    renderSelection();
  });
}

async function deleteSelectedResource() {
  const selected = findResource(state.selectedId);
  if (!selected) return;

  await withBusy(async () => {
    if (state.activeSessionId && selected.resource.id === state.selectedId) {
      await apiStopSSHSession(state.activeSessionId);
      state.activeSessionId = null;
    }
    await apiDeleteResource(selected.resource.id);
    terminal.writeln(`Deleted ${selected.resource.hostname}.`);
    state.selectedId = selected.parent?.id ?? null;
    await refreshHosts();
  });
}

async function disconnectActiveSession() {
  if (!state.activeSessionId) return;
  const sessionID = state.activeSessionId;
  state.activeSessionId = null;
  await apiStopSSHSession(sessionID);
  renderSelection();
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
    renderSelection();
  });
  row.append(selectButton);

  if (!child) {
    const addSubsystem = document.createElement('button');
    addSubsystem.type = 'button';
    addSubsystem.className = 'row-action';
    addSubsystem.title = 'Add subsystem';
    addSubsystem.setAttribute('aria-label', `Add subsystem to ${resource.hostname}`);
    addSubsystem.textContent = '+';
    addSubsystem.addEventListener('click', () => openResourcePanel('subsystem', resource.id));
    row.append(addSubsystem);
  }

  return {
    element: row,
    search: `${resource.hostname} ${resource.ip} ${resource.user} ${type}`.toLowerCase(),
  };
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

function closeResourcePanel() {
  const panel = document.querySelector('#resource-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  state.drawerMode = null;
  state.drawerHostId = null;
}

function openConnectPanel() {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;

  const panel = document.querySelector('#connect-panel');
  const form = document.querySelector('#connect-form');
  form.reset();
  form.elements.trustHostKey.checked = true;
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

function renderSelection() {
  document.querySelectorAll('.host-row').forEach((row) => {
    row.classList.toggle('selected', row.dataset.id === state.selectedId);
  });

  const selected = findResource(state.selectedId);
  const title = document.querySelector('#session-title');
  const connect = document.querySelector('#connect');
  const disconnect = document.querySelector('#disconnect');
  const remove = document.querySelector('#delete-resource');

  if (!selected) {
    title.textContent = 'No session selected';
    connect.disabled = true;
    disconnect.disabled = true;
    remove.disabled = true;
    return;
  }

  title.textContent = `${selected.resource.user}@${selected.resource.hostname}`;
  connect.disabled = state.busy;
  disconnect.disabled = state.busy || !state.activeSessionId;
  remove.disabled = state.busy;
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

async function withBusy(task) {
  if (state.busy) return;

  state.busy = true;
  setDisabledState(true);
  try {
    await task();
  } catch (error) {
    terminal.writeln(`\r\nError: ${error?.message ?? error}`);
  } finally {
    state.busy = false;
    setDisabledState(false);
    renderSelection();
  }
}

function setDisabledState(disabled) {
  document.querySelectorAll('button, input, select').forEach((element) => {
    element.disabled = disabled;
  });
}

function fitTerminal() {
  fitAddon.fit();
}

function resizeActiveSession() {
  if (!state.activeSessionId) return;
  apiResizeSSHSession(state.activeSessionId, terminal.cols, terminal.rows).catch(() => {});
}

function registerSSHEvents() {
  if (!globalThis.runtime?.EventsOn) return;

  EventsOn('ssh:output', (event) => {
    if (!state.activeSessionId || event.sessionId === state.activeSessionId) {
      terminal.write(event.data ?? '');
    }
  });
  EventsOn('ssh:status', (event) => {
    if (!state.activeSessionId || event.sessionId === state.activeSessionId) {
      terminal.writeln(`\r\n${event.message}`);
    }
  });
  EventsOn('ssh:closed', (event) => {
    if (!state.activeSessionId || event.sessionId === state.activeSessionId) {
      terminal.writeln(`\r\n${event.message}`);
      state.activeSessionId = null;
      renderSelection();
    }
  });
}

function wailsAPI() {
  return globalThis.go?.main?.App ?? globalThis.go?.desktop?.App;
}

async function apiListHosts() {
  const api = wailsAPI();
  if (api?.ListHosts) {
    return (await api.ListHosts()) ?? [];
  }
  return clone(demoStore.hosts);
}

async function apiAddHost(input) {
  const api = wailsAPI();
  if (api?.AddHost) {
    return await api.AddHost(input);
  }

  const host = {
    id: `host-${Date.now()}`,
    hostname: input.hostname,
    ip: input.ip,
    port: input.port,
    user: input.user,
    subsystems: [],
  };
  demoStore.hosts.push(host);
  return clone(host);
}

async function apiAddSubsystem(hostID, input) {
  const api = wailsAPI();
  if (api?.AddSubsystem) {
    return await api.AddSubsystem(hostID, input);
  }

  const host = demoStore.hosts.find((candidate) => candidate.id === hostID);
  if (!host) throw new Error(`Host ${hostID} not found`);
  const subsystem = {
    id: `${input.type}-${Date.now()}`,
    type: input.type,
    hostname: input.hostname,
    ip: input.ip,
    port: input.port,
    user: input.user,
  };
  host.subsystems.push(subsystem);
  return clone(subsystem);
}

async function apiDeleteResource(id) {
  const api = wailsAPI();
  if (api?.DeleteResource) {
    return await api.DeleteResource(id);
  }

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
  if (api?.StartSSHSession) {
    return await api.StartSSHSession(input);
  }
  terminal.writeln('Browser fallback: SSH is available only in the Wails desktop build.');
  return `demo-session-${Date.now()}`;
}

async function apiWriteSSHSession(sessionID, data) {
  const api = wailsAPI();
  if (api?.WriteSSHSession) {
    return await api.WriteSSHSession(sessionID, data);
  }
  terminal.write(data);
}

async function apiResizeSSHSession(sessionID, cols, rows) {
  const api = wailsAPI();
  if (api?.ResizeSSHSession) {
    return await api.ResizeSSHSession(sessionID, cols, rows);
  }
}

async function apiStopSSHSession(sessionID) {
  const api = wailsAPI();
  if (api?.StopSSHSession) {
    return await api.StopSSHSession(sessionID);
  }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}
