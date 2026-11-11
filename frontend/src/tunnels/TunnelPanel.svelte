<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { SAVED_PASSWORD_MASK } from '../password-field.js';
  import AuthFields from '../connections/AuthFields.svelte';
  import { authInput } from '../connections/auth-model.js';
  import { tunnelMode, tunnelStatus, tunnelSummary, tunnelValuesForMode } from './model.js';

  let { onready, onsubmit, onstop, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let panel = $state({ keyGroups: [], keys: [], resource: null, tunnel: null });
  let values = $state({});
  let localPortInput = $state();

  let mode = $derived(tunnelMode(values.type));
  let summary = $derived(tunnelSummary(panel.resource, values));
  let status = $derived(tunnelStatus(panel.tunnel));

  async function open(data = {}) {
    panel = data;
    values = { ...data.values };
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      localPortInput?.focus();
      localPortInput?.select?.();
    });
  }

  function close() {
    opened = false;
    visible = false;
  }

  function setBusy(value) {
    busy = Boolean(value);
  }

  function setTunnel(tunnel) {
    panel = { ...panel, tunnel: tunnel ?? null };
  }

  function markPasswordSaved(saved) {
    values = {
      ...values,
      hadSavedPassword: Boolean(saved),
      savePassword: Boolean(saved),
      password: saved ? SAVED_PASSWORD_MASK : values.password === SAVED_PASSWORD_MASK ? '' : values.password,
    };
  }

  function changeMode(event) {
    values = tunnelValuesForMode(values, event.currentTarget.value);
  }

  function submit(event) {
    event.preventDefault();
    if (busy || panel.tunnel) return;
    onsubmit?.({
      resourceId: panel.resourceId,
      input: {
        resourceId: panel.resourceId,
        type: values.type,
        localHost: values.localHost.trim(),
        localPort: Number.parseInt(String(values.localPort), 10),
        remoteHost: values.remoteHost.trim(),
        remotePort: Number.parseInt(String(values.remotePort), 10),
        ...authInput(values, panel.keys),
        trustHostKey: Boolean(values.trustHostKey),
      },
    });
  }

  onMount(() => onready?.({ open, close, setBusy, setTunnel, markPasswordSaved, isOpen: () => visible }));
</script>

<section id="tunnel-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close tunnel panel" onclick={onrequestclose}></button>
  <form id="tunnel-form" class="panel-card" onsubmit={submit}>
    <header class="panel-header">
      <div>
        <p class="eyebrow">SSH Tunnel</p>
        <h3>{mode.title}</h3>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <p class="parent-summary">{summary}</p>
    <p class="tunnel-status" hidden={!status}>{status}</p>

    <label>
      <span>Mode</span>
      <select value={values.type} name="type" onchange={changeMode}>
        <option value="socks">SOCKS proxy (-D)</option>
        <option value="local">Local forward (-L)</option>
        <option value="remote">Remote forward (-R)</option>
      </select>
    </label>
    <div class="form-grid">
      <label>
        <span>{mode.bindLabel}</span>
        <input bind:value={values.localHost} name="localHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" />
      </label>
      <label>
        <span>{mode.portLabel}</span>
        <input bind:this={localPortInput} bind:value={values.localPort} name="localPort" type="number" min="1" max="65535" required />
      </label>
    </div>
    <div class="form-grid" hidden={!mode.hasTarget}>
      <label>
        <span>{mode.targetHostLabel}</span>
        <input bind:value={values.remoteHost} name="remoteHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" />
      </label>
      <label>
        <span>{mode.targetPortLabel}</span>
        <input bind:value={values.remotePort} name="remotePort" type="number" min="1" max="65535" />
      </label>
    </div>

    <AuthFields
      {values}
      keyGroups={panel.keyGroups ?? []}
      context="tunnel"
      onchange={(next) => { values = next; }}
    />

    <footer class="panel-actions">
      <button class="secondary" type="button" onclick={onrequestclose}>Close</button>
      <button class="secondary" type="button" disabled={busy || !panel.tunnel} onclick={() => onstop?.(panel.resourceId)}>Stop</button>
      <button type="submit" disabled={busy || Boolean(panel.tunnel)}>Start Tunnel</button>
    </footer>
  </form>
</section>
