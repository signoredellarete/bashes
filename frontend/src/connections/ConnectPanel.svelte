<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { placeCaretAfterSavedPassword } from '../password-field.js';
  import { authInput, CUSTOM_KEY_PATH_HELP } from './auth-model.js';

  let { onready, onsubmit, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let panel = $state({ keyGroups: [], keys: [] });
  let status = $state({ message: '', kind: '' });
  let values = $state({
    authMethod: 'agent', keyChoice: '', password: '', hadSavedPassword: false,
    savePassword: false, privateKeyPath: '', privateKeyPassphrase: '', trustHostKey: false,
  });
  let authMethodInput = $state();
  let keyInput = $state();
  let passwordInput = $state();
  let pathInput = $state();

  let passwordAuth = $derived(values.authMethod === 'password');
  let keyAuth = $derived(values.authMethod === 'key');
  let pathAuth = $derived(values.authMethod === 'path');
  let passphraseAuth = $derived(keyAuth || pathAuth);

  async function open(data = {}) {
    panel = data;
    status = data.status ?? { message: '', kind: '' };
    values = { ...values, ...data.values };
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
    window.setTimeout(() => {
      requestAnimationFrame(() => {
        const target = passwordAuth
          ? passwordInput
          : keyAuth
            ? keyInput
            : pathAuth
              ? pathInput
              : authMethodInput;
        target?.focus();
        if (!(passwordAuth && placeCaretAfterSavedPassword(target))) target?.select?.();
      });
    }, 0);
  }

  function chooseKey() {
    if (!values.keyChoice) return;
    values.authMethod = 'key';
    values.privateKeyPath = '';
  }

  function enterPath() {
    if (!values.privateKeyPath.trim()) return;
    values.authMethod = 'path';
    values.keyChoice = '';
  }

  function submit(event) {
    event.preventDefault();
    if (busy) return;
    onsubmit?.({
      resourceId: panel.resourceId,
      input: {
        resourceId: panel.resourceId,
        ...authInput(values, panel.keys),
        trustHostKey: Boolean(values.trustHostKey),
        cols: 120,
        rows: 32,
      },
    });
  }

  onMount(() => onready?.({ open, close, setBusy, setStatus, focusAuth, isOpen: () => visible }));
</script>

<section id="connect-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close connection panel" onclick={onrequestclose}></button>
  <form id="connect-form" class="panel-card" onsubmit={submit}>
    <header class="panel-header">
      <div>
        <p class="eyebrow">SSH</p>
        <h3>Connect</h3>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <p class="parent-summary">{panel.summary}</p>
    <p class="inline-status" data-kind={status.kind} title={status.message} hidden={!status.message}>{status.message}</p>

    <label>
      <span>Authentication</span>
      <select bind:this={authMethodInput} bind:value={values.authMethod} name="authMethod">
        <option value="agent">Agent / system keys</option>
        <option value="password">Password</option>
        <option value="key">Existing key</option>
        <option value="path">Key file</option>
      </select>
    </label>
    <label hidden={!keyAuth}>
      <span>Existing Key</span>
      <select bind:this={keyInput} bind:value={values.keyChoice} name="keyName" onchange={chooseKey}>
        <option value="">Use password/agent or custom path</option>
        {#each panel.keyGroups ?? [] as group (group.label)}
          <optgroup label={group.label}>
            {#each group.options as option (option.value)}
              <option value={option.value} title={option.title}>{option.label}</option>
            {/each}
          </optgroup>
        {/each}
      </select>
    </label>
    <label hidden={!passwordAuth}>
      <span>Password</span>
      <input bind:this={passwordInput} bind:value={values.password} name="password" type="password" autocomplete="current-password" />
    </label>
    <label class="checkbox-row" hidden={!passwordAuth}>
      <input bind:checked={values.savePassword} name="savePassword" type="checkbox" />
      <span>Save password in system keyring</span>
    </label>
    <label hidden={!pathAuth}>
      <span class="field-label-with-help">
        Custom Key Path
        <span class="field-help-trigger" role="button" tabindex="0" aria-label="Custom key path help" aria-expanded="false">?</span>
        <span class="field-help-popover">{CUSTOM_KEY_PATH_HELP}</span>
      </span>
      <input
        bind:this={pathInput}
        bind:value={values.privateKeyPath}
        name="privateKeyPath"
        autocomplete="off"
        autocapitalize="none"
        autocorrect="off"
        spellcheck="false"
        placeholder="optional private key path"
        oninput={enterPath}
      />
    </label>
    <label hidden={!passphraseAuth}>
      <span>Key Passphrase</span>
      <input bind:value={values.privateKeyPassphrase} name="privateKeyPassphrase" type="password" autocomplete="off" />
    </label>
    <label class="checkbox-row">
      <input bind:checked={values.trustHostKey} name="trustHostKey" type="checkbox" />
      <span>Skip host key verification for this session (insecure)</span>
    </label>

    <footer class="panel-actions">
      <button class="secondary" type="button" onclick={onrequestclose}>Cancel</button>
      <button type="submit" disabled={busy}>Connect</button>
    </footer>
  </form>
</section>
