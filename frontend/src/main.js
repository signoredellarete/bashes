import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
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
  busy: false,
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
      <button id="refresh" class="icon-button wide" type="button" title="Refresh hosts">Refresh</button>
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
        <button id="connect" type="button" disabled>Connect</button>
      </div>
    </section>

    <section class="workbench">
      <section class="forms" aria-label="Resource forms">
        <form id="host-form" class="resource-form">
          <h3>Add Host</h3>
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
          <button type="submit">Add Host</button>
        </form>

        <form id="subsystem-form" class="resource-form">
          <h3>Add Subsystem</h3>
          <label>
            <span>Parent Host</span>
            <select name="hostId" required></select>
          </label>
          <label>
            <span>Type</span>
            <select name="type" required>
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
          <button type="submit">Add Subsystem</button>
        </form>
      </section>

      <section id="terminal" class="terminal" aria-label="Terminal"></section>
    </section>
  </main>
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
fitAddon.fit();
terminal.writeln('Bashes terminal ready.');
terminal.writeln('SSH transport will be attached in the next backend milestone.');

window.addEventListener('resize', () => fitAddon.fit());

document.querySelector('#refresh').addEventListener('click', () => loadHosts());
document.querySelector('#search').addEventListener('input', (event) => renderHosts(event.target.value));
document.querySelector('#connect').addEventListener('click', () => {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  terminal.writeln('');
  terminal.writeln(`Preparing SSH session for ${selected.user}@${selected.ip}:${selected.port}`);
});
document.querySelector('#delete-resource').addEventListener('click', () => deleteSelectedResource());
document.querySelector('#host-form').addEventListener('submit', (event) => submitHost(event));
document.querySelector('#subsystem-form').addEventListener('submit', (event) => submitSubsystem(event));

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
  renderHostOptions();
  renderSelection();
}

async function submitHost(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const input = endpointInput(form);

  await withBusy(async () => {
    const host = await apiAddHost(input);
    form.reset();
    form.elements.port.value = '22';
    state.selectedId = host.id;
    await refreshHosts();
    terminal.writeln(`Added host ${host.hostname}.`);
  });
}

async function submitSubsystem(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const hostID = form.elements.hostId.value;
  const input = endpointInput(form, form.elements.type.value);

  await withBusy(async () => {
    const subsystem = await apiAddSubsystem(hostID, input);
    form.reset();
    form.elements.port.value = '22';
    form.elements.type.value = input.type;
    state.selectedId = subsystem.id;
    await refreshHosts();
    terminal.writeln(`Added ${subsystem.type} ${subsystem.hostname}.`);
  });
}

async function deleteSelectedResource() {
  const selected = findResource(state.selectedId);
  if (!selected) return;

  await withBusy(async () => {
    await apiDeleteResource(selected.resource.id);
    terminal.writeln(`Deleted ${selected.resource.hostname}.`);
    state.selectedId = selected.parent?.id ?? null;
    await refreshHosts();
  });
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
  const button = document.createElement('button');
  button.type = 'button';
  button.className = `host-row ${child ? 'child' : ''}`;
  button.dataset.id = resource.id;
  button.innerHTML = `
    <span class="type"></span>
    <span class="details">
      <strong></strong>
      <small></small>
    </span>
  `;
  button.querySelector('.type').textContent = type;
  button.querySelector('strong').textContent = resource.hostname;
  button.querySelector('small').textContent = `${resource.user}@${resource.ip}:${resource.port}`;
  button.addEventListener('click', () => {
    state.selectedId = resource.id;
    renderHostOptions();
    renderSelection();
  });

  return {
    element: button,
    search: `${resource.hostname} ${resource.ip} ${resource.user} ${type}`.toLowerCase(),
  };
}

function renderHostOptions() {
  const select = document.querySelector('#subsystem-form select[name="hostId"]');
  const selected = findResource(state.selectedId);
  const preferredHostID = selected?.parent?.id ?? (selected?.type === 'host' ? selected.resource.id : '');

  select.replaceChildren(...state.hosts.map((host) => {
    const option = document.createElement('option');
    option.value = host.id;
    option.textContent = `${host.hostname} (${host.user}@${host.ip})`;
    option.selected = host.id === preferredHostID;
    return option;
  }));
}

function renderSelection() {
  document.querySelectorAll('.host-row').forEach((row) => {
    row.classList.toggle('selected', row.dataset.id === state.selectedId);
  });

  const selected = findResource(state.selectedId);
  const title = document.querySelector('#session-title');
  const connect = document.querySelector('#connect');
  const remove = document.querySelector('#delete-resource');
  const subsystemForm = document.querySelector('#subsystem-form');

  if (!selected) {
    title.textContent = 'No session selected';
    connect.disabled = true;
    remove.disabled = true;
    subsystemForm.querySelectorAll('input, select, button').forEach((field) => {
      field.disabled = state.hosts.length === 0 || state.busy;
    });
    return;
  }

  title.textContent = `${selected.resource.user}@${selected.resource.hostname}`;
  connect.disabled = state.busy;
  remove.disabled = state.busy;
  subsystemForm.querySelectorAll('input, select, button').forEach((field) => {
    field.disabled = state.hosts.length === 0 || state.busy;
  });
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
    terminal.writeln(`Error: ${error?.message ?? error}`);
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

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}
