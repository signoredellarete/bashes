<script>
  import { onMount } from 'svelte';
  import { placeCaretAfterSavedPassword } from '../password-field.js';
  import { CUSTOM_KEY_PATH_HELP } from './auth-model.js';

  let { values, keyGroups = [], context = 'session', onchange, onready } = $props();

  let authMethodInput = $state();
  let keyInput = $state();
  let passwordInput = $state();
  let pathInput = $state();

  let passwordAuth = $derived(values.authMethod === 'password');
  let keyAuth = $derived(values.authMethod === 'key');
  let pathAuth = $derived(values.authMethod === 'path');
  let passphraseAuth = $derived(keyAuth || pathAuth);

  function update(patch) {
    onchange?.({ ...values, ...patch });
  }

  function chooseKey(event) {
    const keyChoice = event.currentTarget.value;
    update({ keyChoice, ...(keyChoice ? { authMethod: 'key', privateKeyPath: '' } : {}) });
  }

  function enterPath(event) {
    const privateKeyPath = event.currentTarget.value;
    update({ privateKeyPath, ...(privateKeyPath.trim() ? { authMethod: 'path', keyChoice: '' } : {}) });
  }

  function focus() {
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

  onMount(() => onready?.({ focus }));
</script>

<label>
  <span>Authentication</span>
  <select bind:this={authMethodInput} value={values.authMethod} name="authMethod" onchange={(event) => update({ authMethod: event.currentTarget.value })}>
    <option value="agent">Agent / system keys</option>
    <option value="password">Password</option>
    <option value="key">Existing key</option>
    <option value="path">Key file</option>
  </select>
</label>
<label hidden={!keyAuth}>
  <span>Existing Key</span>
  <select bind:this={keyInput} value={values.keyChoice} name="keyName" onchange={chooseKey}>
    <option value="">Use password/agent or custom path</option>
    {#each keyGroups as group (group.label)}
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
  <input
    bind:this={passwordInput}
    value={values.password}
    name="password"
    type="password"
    autocomplete="current-password"
    oninput={(event) => update({ password: event.currentTarget.value })}
  />
</label>
<label class="checkbox-row" hidden={!passwordAuth}>
  <input
    checked={values.savePassword}
    name="savePassword"
    type="checkbox"
    onchange={(event) => update({ savePassword: event.currentTarget.checked })}
  />
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
    value={values.privateKeyPath}
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
  <input
    value={values.privateKeyPassphrase}
    name="privateKeyPassphrase"
    type="password"
    autocomplete="off"
    oninput={(event) => update({ privateKeyPassphrase: event.currentTarget.value })}
  />
</label>
<label class="checkbox-row">
  <input
    checked={values.trustHostKey}
    name="trustHostKey"
    type="checkbox"
    onchange={(event) => update({ trustHostKey: event.currentTarget.checked })}
  />
  <span>Skip host key verification for this {context} (insecure)</span>
</label>
