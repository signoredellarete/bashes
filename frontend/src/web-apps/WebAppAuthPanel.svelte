<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import AuthFields from '../connections/AuthFields.svelte';
  import { authInput } from '../connections/auth-model.js';

  let { onready, onsubmit, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let panel = $state({ keyGroups: [], keys: [] });
  let status = $state({ message: '', kind: '' });
  let values = $state({});
  let authFields;

  async function open(data = {}) {
    panel = data;
    status = data.status ?? { message: '', kind: '' };
    values = { ...data.values };
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      focusAuth();
    });
  }

  function close() {
    opened = false;
    visible = false;
    status = { message: '', kind: '' };
  }

  function setBusy(value) {
    busy = Boolean(value);
  }

  function setStatus(message = '', kind = '') {
    status = { message, kind };
  }

  function focusAuth() {
    authFields?.focus();
  }

  function submit(event) {
    event.preventDefault();
    if (busy) return;
    onsubmit?.({
      webAppId: panel.webAppId,
      gatewayId: panel.gatewayId,
      auth: {
        ...authInput(values, panel.keys),
        trustHostKey: Boolean(values.trustHostKey),
      },
    });
  }

  onMount(() => onready?.({ open, close, setBusy, setStatus, focusAuth, isOpen: () => visible }));
</script>

<section id="webapp-auth-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close Web App authentication panel" onclick={onrequestclose}></button>
  <form id="webapp-auth-form" class="panel-card" onsubmit={submit}>
    <header class="panel-header">
      <div>
        <p class="eyebrow">Web App Tunnel</p>
        <h3>Connect Gateway</h3>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>
    <p class="parent-summary">{panel.summary}</p>
    <p class="inline-status" data-kind={status.kind} title={status.message} hidden={!status.message}>{status.message}</p>
    <AuthFields
      {values}
      keyGroups={panel.keyGroups ?? []}
      context="Web App tunnel"
      onchange={(next) => { values = next; }}
      onready={(api) => { authFields = api; }}
    />
    <footer class="panel-actions">
      <button class="secondary" type="button" onclick={onrequestclose}>Cancel</button>
      <button type="submit" disabled={busy}>Open Web App</button>
    </footer>
  </form>
</section>
