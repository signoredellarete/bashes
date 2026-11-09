<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';

  let { onready, onpreview, onsave, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let terminalFontSize = $state('');
  let terminalScrollback = $state('');
  let terminalFontFamily = $state('');
  let terminalCopyOnSelect = $state(false);
  let webAppBrowser = $state('');
  let customBrowserPath = $state('');
  let fontOptions = $state([]);
  let browserOptions = $state([]);
  let minTerminalFontSize = $state(8);
  let maxTerminalFontSize = $state(36);
  let fontSizeInput;

  function update(data = {}) {
    terminalFontSize = String(data.terminalFontSize ?? '');
    terminalScrollback = String(data.terminalScrollback ?? '');
    terminalFontFamily = data.terminalFontFamily ?? '';
    terminalCopyOnSelect = Boolean(data.terminalCopyOnSelect);
    webAppBrowser = data.webAppBrowser ?? '';
    customBrowserPath = data.customBrowserPath ?? '';
    fontOptions = data.fontOptions ?? [];
    browserOptions = data.browserOptions ?? [];
    minTerminalFontSize = data.minTerminalFontSize ?? 8;
    maxTerminalFontSize = data.maxTerminalFontSize ?? 36;
  }

  async function open(data) {
    update(data);
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      fontSizeInput?.focus();
    });
  }

  function close() {
    opened = false;
    visible = false;
  }

  onMount(() => onready?.({ open, close, update, isOpen: () => visible }));

  function preview() {
    onpreview?.({ terminalFontSize, terminalFontFamily });
  }

  async function submit(event) {
    event.preventDefault();
    await onsave?.({
      terminalFontSize,
      terminalScrollback,
      terminalFontFamily,
      terminalCopyOnSelect,
      webAppBrowser,
      customBrowserPath,
    });
  }
</script>

<section id="settings-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close settings" onclick={onrequestclose}></button>
  <form id="settings-form" class="panel-card" onsubmit={submit}>
    <header class="panel-header">
      <div>
        <p class="eyebrow">Bashes</p>
        <h3>Settings</h3>
      </div>
      <button class="close-panel" type="button" title="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <fieldset class="settings-group">
      <legend>Terminal Appearance</legend>
      <div class="form-grid">
        <label>
          <span>Terminal Font Size</span>
          <input
            bind:this={fontSizeInput}
            bind:value={terminalFontSize}
            oninput={preview}
            name="terminalFontSize"
            type="number"
            min={minTerminalFontSize}
            max={maxTerminalFontSize}
            step="1"
          />
        </label>
        <label>
          <span>Terminal Font Family</span>
          <select bind:value={terminalFontFamily} onchange={preview} name="terminalFontFamily">
            {#each fontOptions as option (option.value)}
              <option value={option.value}>{option.label}</option>
            {/each}
          </select>
        </label>
      </div>
    </fieldset>

    <fieldset class="settings-group">
      <legend>Terminal Behavior</legend>
      <label>
        <span>Scrollback Lines</span>
        <input bind:value={terminalScrollback} name="terminalScrollback" type="number" min="1000" max="500000" step="1000" />
      </label>
      <label class="checkbox-row">
        <input bind:checked={terminalCopyOnSelect} name="terminalCopyOnSelect" type="checkbox" />
        <span>Copy selected terminal text to clipboard</span>
      </label>
    </fieldset>

    <fieldset class="settings-group">
      <legend>Web App Browser</legend>
      <label>
        <span>Preferred Browser</span>
        <select bind:value={webAppBrowser} name="webAppBrowser">
          {#each browserOptions as option (option.value)}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Custom Browser Executable</span>
        <input
          bind:value={customBrowserPath}
          name="customBrowserPath"
          autocomplete="off"
          autocapitalize="none"
          autocorrect="off"
          spellcheck="false"
          placeholder="Optional executable path"
        />
      </label>
    </fieldset>

    <footer class="panel-actions">
      <button class="secondary" type="button" onclick={onrequestclose}>Cancel</button>
      <button type="submit">Save Settings</button>
    </footer>
  </form>
</section>
