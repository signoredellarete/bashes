<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { canonicalTag } from '../resource-tags.js';
  import TagEditor from '../tags/TagEditor.svelte';
  import { normalizeTagSelection } from '../tags/editor-model.js';

  let { onready, onsubmit, oncreatetag, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let panel = $state({});
  let values = $state({});
  let catalog = $state([]);
  let editorKey = $state(0);
  let nameInput = $state();

  let tunnel = $derived(values.connectionMode === 'tunnel');
  let manualPort = $derived(tunnel && values.portMode === 'manual');
  let standardBrowser = $derived(values.browserMode === 'standard');

  async function open(data = {}) {
    panel = data;
    values = { ...data.values, tags: normalizeTagSelection(data.values?.tags) };
    catalog = data.catalog ?? [];
    editorKey += 1;
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      nameInput?.focus();
    });
  }

  function close() {
    opened = false;
    visible = false;
  }

  function setBusy(value) {
    busy = Boolean(value);
  }

  async function createTag(name) {
    const created = await oncreatetag?.(name);
    if (!created) return '';
    const key = canonicalTag(created);
    if (!catalog.some((tag) => tag.key === key)) {
      catalog = [...catalog, { key, name: created, count: 0 }];
    }
    return created;
  }

  function submit(event) {
    event.preventDefault();
    if (busy) return;
    onsubmit?.({
      editId: panel.editId,
      input: {
        name: values.name.trim(),
        url: values.url.trim(),
        tags: [...values.tags],
        connectionMode: values.connectionMode,
        gatewayResourceId: tunnel ? values.gatewayResourceId : '',
        portMode: tunnel ? values.portMode : '',
        localSocksPort: manualPort ? Number.parseInt(String(values.localSocksPort), 10) : 0,
        browserMode: values.browserMode,
        keepCookies: standardBrowser && values.keepCookies,
        keepLocalStorage: standardBrowser && values.keepLocalStorage,
        restoreSession: standardBrowser && values.restoreSession,
        clearCacheOnClose: standardBrowser && values.clearCacheOnClose,
      },
    });
  }

  onMount(() => onready?.({ open, close, setBusy, isOpen: () => visible }));
</script>

<section id="webapp-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close Web App panel" onclick={onrequestclose}></button>
  <form id="webapp-form" class="panel-card webapp-panel-card tag-editor-panel" onsubmit={submit}>
    <header class="panel-header">
      <div>
        <p class="eyebrow">Web App</p>
        <h3>{panel.title}</h3>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <label>
      <span>Name</span>
      <input bind:this={nameInput} bind:value={values.name} name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
    </label>
    <label>
      <span>URL</span>
      <input bind:value={values.url} class="monospace-input" name="url" type="url" placeholder="https://app.internal.example" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
    </label>
    <label>
      <span>Connection</span>
      <select bind:value={values.connectionMode} name="connectionMode">
        <option value="direct">Direct</option>
        <option value="tunnel">SSH tunnel</option>
      </select>
    </label>
    <section class="conditional-fields" hidden={!tunnel}>
      <label>
        <span>SSH Gateway</span>
        <select bind:value={values.gatewayResourceId} name="gatewayResourceId" required={tunnel}>
          {#each panel.gateways ?? [] as gateway (gateway.value)}
            <option value={gateway.value}>{gateway.label}</option>
          {/each}
        </select>
      </label>
      <div class="form-grid">
        <label>
          <span>SOCKS Port</span>
          <select bind:value={values.portMode} name="portMode">
            <option value="automatic">Automatic</option>
            <option value="manual">Manual</option>
          </select>
        </label>
        <label hidden={!manualPort}>
          <span>Local Port</span>
          <input bind:value={values.localSocksPort} name="localSocksPort" type="number" min="1" max="65535" required={manualPort} />
        </label>
      </div>
    </section>
    <label>
      <span>Browser Session</span>
      <select bind:value={values.browserMode} name="browserMode">
        <option value="standard">Standard</option>
        <option value="private">Private</option>
      </select>
    </label>
    <section class="conditional-fields" hidden={!standardBrowser}>
      <label class="checkbox-row"><input bind:checked={values.keepCookies} name="keepCookies" type="checkbox" /><span>Keep cookies</span></label>
      <label class="checkbox-row"><input bind:checked={values.keepLocalStorage} name="keepLocalStorage" type="checkbox" /><span>Keep local storage</span></label>
      <label class="checkbox-row"><input bind:checked={values.restoreSession} name="restoreSession" type="checkbox" /><span>Restore browser session</span></label>
      <label class="checkbox-row"><input bind:checked={values.clearCacheOnClose} name="clearCacheOnClose" type="checkbox" /><span>Clear browser cache on close</span></label>
    </section>

    {#key editorKey}
      <TagEditor
        tags={values.tags}
        {catalog}
        {busy}
        emptySummary="No tags on this Web App"
        countLabel="on this Web App"
        onchange={(tags) => { values.tags = tags; }}
        oncreate={createTag}
      />
    {/key}

    <footer class="panel-actions resource-panel-actions">
      <span>{values.tags?.length ? `${values.tags.length} tag${values.tags.length === 1 ? '' : 's'} on this Web App` : 'No tags on this Web App'}</span>
      <button class="secondary" type="button" onclick={onrequestclose}>Cancel</button>
      <button type="submit" disabled={busy}>{panel.submitLabel}</button>
    </footer>
  </form>
</section>
