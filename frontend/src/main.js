import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import './styles.css';

const state = {
  hosts: [],
  selectedId: null,
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
      <button id="refresh" type="button" title="Refresh hosts">Refresh</button>
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
        <button id="connect" type="button" disabled>Connect</button>
      </div>
    </section>
    <section id="terminal" class="terminal" aria-label="Terminal"></section>
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
  const selected = findResource(state.selectedId);
  if (!selected) return;
  terminal.writeln('');
  terminal.writeln(`Preparing SSH session for ${selected.user}@${selected.ip}:${selected.port}`);
});

await loadHosts();

async function loadHosts() {
  state.hosts = await listHosts();
  if (!state.selectedId && state.hosts.length > 0) {
    state.selectedId = state.hosts[0].id;
  }
  renderHosts(document.querySelector('#search').value);
  renderSelection();
}

async function listHosts() {
  const api = globalThis.go?.desktop?.App;
  if (api?.ListHosts) {
    return await api.ListHosts();
  }

  return [
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
  ];
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
    <span class="type">${type}</span>
    <span class="details">
      <strong></strong>
      <small></small>
    </span>
  `;
  button.querySelector('strong').textContent = resource.hostname;
  button.querySelector('small').textContent = `${resource.user}@${resource.ip}:${resource.port}`;
  button.addEventListener('click', () => {
    state.selectedId = resource.id;
    renderSelection();
  });

  return {
    element: button,
    search: `${resource.hostname} ${resource.ip} ${resource.user} ${type}`.toLowerCase(),
  };
}

function renderSelection() {
  document.querySelectorAll('.host-row').forEach((row) => {
    row.classList.toggle('selected', row.dataset.id === state.selectedId);
  });

  const selected = findResource(state.selectedId);
  const title = document.querySelector('#session-title');
  const connect = document.querySelector('#connect');
  if (!selected) {
    title.textContent = 'No session selected';
    connect.disabled = true;
    return;
  }

  title.textContent = `${selected.user}@${selected.hostname}`;
  connect.disabled = false;
}

function findResource(id) {
  for (const host of state.hosts) {
    if (host.id === id) return host;
    for (const subsystem of host.subsystems ?? []) {
      if (subsystem.id === id) return subsystem;
    }
  }
  return null;
}
